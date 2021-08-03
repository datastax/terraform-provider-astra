package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				"astra_database":                      dataSourceDatabase(),
				"astra_databases":                     dataSourceDatabases(),
				"astra_keyspace":                      dataSourceKeyspace(),
				"astra_keyspaces":                     dataSourceKeyspaces(),
				"astra_secure_connect_bundle_url":     dataSourceSecureConnectBundleURL(),
				"astra_available_regions":			   dataSourceAvailableRegions(),
				"astra_private_links":			       dataSourcePrivateLinks(),
				"astra_private_link_endpoints":		   dataSourcePrivateLinkEndpoints(),
				"astra_access_list":				   dataSourceAccessList(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"astra_database": resourceDatabase(),
				"astra_keyspace": resourceKeyspace(),
				"astra_private_link": resourcePrivateLink(),
				"astra_private_link_endpoint": resourcePrivateLinkEndpoint(),
				"astra_access_list": resourceAccessList(),
			},
			Schema: map[string]*schema.Schema{
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc("ASTRA_API_TOKEN", nil),
					Description: "Authentication token for Astra API.",
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
		token := d.Get("token").(string)
		authorization := fmt.Sprintf("Bearer %s", token)
		clientVersion := fmt.Sprintf("go/%s", astra.Version)

		// Build a retryable http client to automatically
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

		client, err := astra.NewClientWithResponses(astra.ServerURL, func(c *astra.Client) error {
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

		return client, nil
	}
}
