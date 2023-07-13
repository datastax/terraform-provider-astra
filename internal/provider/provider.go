package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	astrarestapi "github.com/datastax/astra-client-go/v2/astra-rest-api"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/datastax/terraform-provider-astra/v2/internal/util"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	DefaultAstraAPIURL     = astra.ServerURL
	DefaultStreamingAPIURL = "https://api.streaming.datastax.com/"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			DataSourcesMap: map[string]*schema.Resource{
				"astra_database":                  dataSourceDatabase(),
				"astra_databases":                 dataSourceDatabases(),
				"astra_keyspace":                  dataSourceKeyspace(),
				"astra_keyspaces":                 dataSourceKeyspaces(),
				"astra_secure_connect_bundle_url": dataSourceSecureConnectBundleURL(),
				"astra_available_regions":         dataSourceAvailableRegions(),
				"astra_private_links":             dataSourcePrivateLinks(),
				"astra_private_link_endpoints":    dataSourcePrivateLinkEndpoints(),
				"astra_access_list":               dataSourceAccessList(),
				"astra_role":                      dataSourceRole(),
				"astra_roles":                     dataSourceRoles(),
				"astra_users":                     dataSourceUsers(),
				"astra_streaming_tenant_tokens":   dataSourceStreamingTenantTokens(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"astra_database":              resourceDatabase(),
				"astra_keyspace":              resourceKeyspace(),
				"astra_private_link":          resourcePrivateLink(),
				"astra_private_link_endpoint": resourcePrivateLinkEndpoint(),
				"astra_access_list":           resourceAccessList(),
				"astra_role":                  resourceRole(),
				"astra_token":                 resourceToken(),
				"astra_cdc":                   resourceCDC(),
				"astra_streaming_tenant":      resourceStreamingTenant(),
				"astra_streaming_sink":        resourceStreamingSink(),
				"astra_streaming_topic":       resourceStreamingTopic(),
				"astra_table":                 resourceTable(),
			},
			Schema: map[string]*schema.Schema{
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Authentication token for Astra API. May also be provided via ASTRA_API_TOKEN environment variable.",
					Sensitive:   true,
				},
				"astra_api_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "URL for Astra API. May also be provided via ASTRA_API_URL environment variable.",
				},
				"streaming_api_url": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "URL for Astra Streaming API. May also be provided via ASTRA_STREAMING_API_URL environment variable.",
				},
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

func configure(providerVersion string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		userAgent := p.UserAgent("terraform-provider-astra", providerVersion)

		token := resourceDataOrDefaultString(d, "token", os.Getenv("ASTRA_API_TOKEN"))
		if token == "" {
			return nil, diag.FromErr(errors.New("missing required Astra API token.  Please set the ASTRA_API_TOKEN environment variable or provide a token in the provider configuration"))
		}

		astraAPIServerURL := resourceDataOrDefaultString(d, "astra_api_url", util.EnvVarOrDefault("ASTRA_API_URL", DefaultAstraAPIURL))
		if _, err := url.Parse(astraAPIServerURL); err != nil {
			return nil, diag.FromErr(fmt.Errorf("invalid Astra server API URL: %w", err))
		}

		streamingAPIServerURL := resourceDataOrDefaultString(d, "streaming_api_url", util.EnvVarOrDefault("ASTRA_STREAMING_API_URL", DefaultAstraAPIURL))
		if _, err := url.Parse(astraAPIServerURL); err != nil {
			return nil, diag.FromErr(fmt.Errorf("invalid Astra Streaming server API URL: %w", err))
		}

		authorization := fmt.Sprintf("Bearer %s", token)
		clientVersion := fmt.Sprintf("go/%s", astra.Version)

		// Build a retryable http astraClient to automatically
		// handle intermittent api errors
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = 10
		retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
			// Never retry POST requests because of side effects
			if resp.Request.Method == "POST" {
				return false, err
			}
			return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
		}

		astraClient, err := astra.NewClientWithResponses(astraAPIServerURL, func(c *astra.Client) error {
			c.Client = retryClient.StandardClient()
			c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Authorization", authorization)
				req.Header.Set("User-Agent", userAgent)
				req.Header.Set("X-Astra-Provider-Version", providerVersion)
				req.Header.Set("X-Astra-Client-Version", clientVersion)
				return nil
			})
			return nil
		})
		if err != nil {
			return nil, diag.FromErr(err)
		}

		streamingClient, err := astrastreaming.NewClientWithResponses(astraAPIServerURL, func(c *astrastreaming.Client) error {
			c.Client = retryClient.StandardClient()
			c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("Authorization", authorization)
				req.Header.Set("User-Agent", userAgent)
				req.Header.Set("X-Astra-Provider-Version", providerVersion)
				req.Header.Set("X-Astra-Client-Version", clientVersion)
				return nil
			})
			return nil
		})
		if err != nil {
			return nil, diag.FromErr(err)
		}

		streamingV3Client, err := astrastreaming.NewClientWithResponses(streamingAPIServerURL, func(c *astrastreaming.Client) error {
			c.Client = retryClient.StandardClient()
			c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
				req.Header.Set("User-Agent", userAgent)
				req.Header.Set("X-Astra-Provider-Version", providerVersion)
				req.Header.Set("X-Astra-Client-Version", clientVersion)
				return nil
			})
			return nil
		})
		if err != nil {
			return nil, diag.FromErr(err)
		}

		var clientCache = make(map[string]astrarestapi.Client)

		clients := astraClients{
			astraClient:            astraClient,
			astraStreamingClient:   streamingClient,
			astraStreamingClientv3: streamingV3Client,
			token:                  token,
			stargateClientCache:    clientCache,
			providerVersion:        providerVersion,
			userAgent:              userAgent,
		}
		return clients, nil
	}
}

func newRestClient(dbid string, providerVersion string, userAgent string, region string) (astrarestapi.Client, error) {
	clientVersion := fmt.Sprintf("go/%s", astra.Version)
	// Build a retryable http astraClient to automatically
	// handle intermittent api errors
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Never retry POST requests because of side effects
		if resp.Request.Method == "POST" {
			return false, err
		}
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	serverURL := fmt.Sprintf("https://%s-%s.apps.astra.datastax.com/api/rest/", dbid, region)
	restClient, err := astrarestapi.NewClient(serverURL, func(c *astrarestapi.Client) error {
		c.Client = retryClient.StandardClient()
		c.RequestEditors = append(c.RequestEditors, func(ctx context.Context, req *http.Request) error {
			req.Header.Set("User-Agent", userAgent)
			req.Header.Set("X-Astra-Provider-Version", providerVersion)
			req.Header.Set("X-Astra-Client-Version", clientVersion)
			return nil
		})
		return nil
	})
	if err != nil {
		return *restClient, err
	}
	return *restClient, nil
}

type astraClients struct {
	astraClient            interface{}
	astraStreamingClient   interface{}
	token                  string
	astraStreamingClientv3 *astrastreaming.ClientWithResponses
	stargateClientCache    map[string]astrarestapi.Client
	providerVersion        string
	userAgent              string
}

// resourceDataOrDefaultString returns the value in the given ResourceData variable or a default value
func resourceDataOrDefaultString(d *schema.ResourceData, resourceVar, defaultValue string) string {
	if v := d.Get(resourceVar).(string); v != "" {
		return v
	}
	return defaultValue
}
