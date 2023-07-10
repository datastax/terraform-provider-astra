package astra

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	astrarestapi "github.com/datastax/astra-client-go/v2/astra-rest-api"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/datastax/pulsar-admin-client-go/src/pulsaradmin"
	"github.com/datastax/terraform-provider-astra/v2/internal/util"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	fullProviderName = "terraform-astra-provider"

	DefaultAstraAPIURL     = astra.ServerURL
	DefaultStreamingAPIURL = "https://api.streaming.datastax.com/"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &astraProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &astraProvider{
			Version: version,
		}
	}
}

// astraProvider is the provider implementation.
type astraProvider struct {
	Version string
}

type astraProviderModel struct {
	Token                   types.String `tfsdk:"token"`
	AstraServerURL          types.String `tfsdk:"astra_api_url"`
	AstraStreamingServerURL types.String `tfsdk:"streaming_api_url"`
}

type astraClients struct {
	token                string
	astraClient          *astra.ClientWithResponses
	astraStreamingClient *astrastreaming.ClientWithResponses
	pulsarAdminClient    *pulsaradmin.ClientWithResponses
	stargateClientCache  map[string]astrarestapi.Client
	providerVersion      string
	userAgent            string
}

// Metadata returns the provider type name.
func (p *astraProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "astra"
	resp.Version = p.Version
}

// Schema defines the provider-level schema for configuration data.
func (p *astraProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Description: "Interact with Astra.",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				MarkdownDescription: "Authentication token for Astra API. May also be provided via ASTRA_API_TOKEN environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"astra_api_url": schema.StringAttribute{
				MarkdownDescription: "URL for Astra API. May also be provided via ASTRA_API_URL environment variable.",
				Optional:            true,
			},
			"streaming_api_url": schema.StringAttribute{
				MarkdownDescription: "URL for Astra Streaming API. May also be provided via ASTRA_STREAMING_API_URL environment variable.",
				Optional:            true,
			},
		},
	}
}

// Configure prepares a HashiCups API client for data sources and resources.
func (p *astraProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Configuring Astra client")

	// Retrieve provider data from configuration
	var config astraProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	token := util.FirstNonEmptyString(config.Token.ValueString(), os.Getenv("ASTRA_API_TOKEN"))
	if token == "" {
		resp.Diagnostics.AddError("missing required Astra API token",
			"missing required Astra API token.  Please set the ASTRA_API_TOKEN environment variable or provide a token in the provider configuration")
		return
	}

	astraAPIServerURL := util.FirstNonEmptyString(config.AstraServerURL.ValueString(), os.Getenv("ASTRA_API_URL"), DefaultAstraAPIURL)
	if _, err := url.Parse(astraAPIServerURL); err != nil {
		resp.Diagnostics.AddError("invalid Astra server API URL", err.Error())
		return
	}

	streamingAPIServerURL := util.FirstNonEmptyString(config.AstraStreamingServerURL.ValueString(), os.Getenv("ASTRA_STREAMING_API_URL"), DefaultStreamingAPIURL)
	if _, err := url.Parse(astraAPIServerURL); err != nil {
		resp.Diagnostics.AddError("invalid Astra streaming server API URL", err.Error())
		return
	}

	// TODO: when we switch to go 1.19, this should use url.JoinPath
	streamingAPIServerURLPulsarAdmin := streamingAPIServerURL + "/admin/v2"
	if _, err := url.Parse(streamingAPIServerURLPulsarAdmin); err != nil {
		resp.Diagnostics.AddError("invalid Pulsar admin server API URL", err.Error())
		return
	}

	// Build a retryable http astraClient to automatically
	// handle intermittent api errors
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Never retry POST requests because of side effects
		if resp.Request.Method == "POST" {
			return false, err
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	// TODO: can we get this version at compile time?
	pluginFrameworkVersion := "1.2.0"
	userAgent := p.UserAgent(req.TerraformVersion, pluginFrameworkVersion)
	authorization := fmt.Sprintf("Bearer %s", token)
	clientVersion := fmt.Sprintf("go/%s", astra.Version)
	astraClient, err := astra.NewClientWithResponses(astraAPIServerURL, func(c *astra.Client) error {
		c.Client = retryClient.StandardClient()
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", authorization)
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("X-Astra-Provider-Version", p.Version)
			req.Header.Set("X-Astra-Client-Version", clientVersion)
			return nil
		})
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to create Astra client", err.Error())
		return
	}

	streamingClient, err := astrastreaming.NewClientWithResponses(streamingAPIServerURL, func(c *astrastreaming.Client) error {
		c.Client = retryClient.StandardClient()
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			req.Header.Set("Authorization", authorization)
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("X-Astra-Provider-Version", p.Version)
			req.Header.Set("X-Astra-Client-Version", clientVersion)
			return nil
		})
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to create Astra Streaming client", err.Error())
		return
	}

	// The streaming API server can handle Pulsar admin requests under the '/admin/v2' path, and these are passed through to a backend Pulsar cluster
	pulsarAdminClient, err := pulsaradmin.NewClientWithResponses(streamingAPIServerURLPulsarAdmin, func(c *pulsaradmin.Client) error {
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("X-Astra-Provider-Version", p.Version)
			req.Header.Set("X-Astra-Client-Version", clientVersion)
			return nil
		})
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("failed to create Pulsar Admin client", err.Error())
		return
	}

	var clientCache = make(map[string]astrarestapi.Client)

	clients := &astraClients{
		astraClient:          astraClient,
		astraStreamingClient: streamingClient,
		pulsarAdminClient:    pulsarAdminClient,
		token:                token,
		stargateClientCache:  clientCache,
		providerVersion:      p.Version,
		userAgent:            userAgent,
	}
	resp.ResourceData = clients
	resp.DataSourceData = clients
}

// DataSources defines the data sources implemented in the provider.
func (p *astraProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

// Resources defines the resources implemented in the provider.
func (p *astraProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewStreamingNamespaceResource,
		NewStreamingPulsarTokenResource,
	}
}

const uaEnvVar = "TF_APPEND_USER_AGENT"

// UserAgent returns a string suitable for use in the User-Agent header of
// requests generated by the provider. This is similar to the UserAgent function in
// the Terraform SDK and is implemented here because it is not yet available in
// the Terraform Plugin Framework.  See https://github.com/hashicorp/terraform-plugin-framework/issues/280
//
// If TF_APPEND_USER_AGENT is set, its value will be appended to the returned
// string.
func (p *astraProvider) UserAgent(terraformVersion, pluginFrameworkVersion string) string {
	ua := fmt.Sprintf("Terraform/%s (+https://www.terraform.io) Terraform-Plugin-Framework/%s %s/%s",
		terraformVersion, pluginFrameworkVersion, fullProviderName, p.Version)

	if add := os.Getenv(uaEnvVar); add != "" {
		add = strings.TrimSpace(add)
		if len(add) > 0 {
			ua += " " + add
			log.Printf("[DEBUG] Using modified User-Agent: %s", ua)
		}
	}

	return ua
}
