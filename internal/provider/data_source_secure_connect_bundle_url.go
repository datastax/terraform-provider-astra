package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceSecureConnectBundleURL() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_secure_connect_bundle_url` provides a datasource that generates a temporary secure connect bundle URL. This URL lasts five minutes. Secure connect bundles are used to connect to Astra using cql cassandra drivers. See the [docs](https://docs.datastax.com/en/astra/docs/connecting-to-database.html) for more information on how to connect.",

		ReadContext: dataSourceSecureConnectBundleURLRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			// Optional inputs
			"datacenter_id": {
				Description:  "The ID of the Astra datacenter. If omitted, all bundles will be fetched.",
				Type:         schema.TypeString,
				Optional:     true,
			},
			// Computed
			"secure_bundles": {
				Description: "A list of Secure Connect Bundle info",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"datacenter_id": {
							Description:  "The ID of the Astra datacenter.",
							Type:         schema.TypeString,
							Computed:     true,
						},
						"url": {
							Description: "The temporary download url to the secure connect bundle zip file.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"internal_url": {
							Description: "The temporary internal download url to the secure connect bundle zip file.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"migration_proxy_url": {
							Description: "The temporary migration proxy url.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"internal_migration_proxy_url": {
							Description: "The temporary internal migration proxy url.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"custom_domain_bundles" : {
							Description: "Bundles for custom domain.",
							Type:        schema.TypeList,
							Computed:    true,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"domain": {
										Description: "Custom Domain.",
										Type:         schema.TypeString,
										Computed:     true,
									},
									"url": {
										Description: "The temporary download url to the secure connect bundle zip file. This one is for the Custom Domain",
										Type:         schema.TypeString,
										Computed:     true,
									},
									"api_fqdn": {
										Description: "The FQDN for API requests",
										Type:         schema.TypeString,
										Computed:     true,
									},
									"cql_fqdn": {
										Description: "The FQDN for CQL requests",
										Type:         schema.TypeString,
										Computed:     true,
									},
									"dashboard_fqdn": {
										Description: "The FQDN for the Dashboard",
										Type:         schema.TypeString,
										Computed:     true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceSecureConnectBundleURLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)
	datacenterID := d.Get("datacenter_id").(string)
	if datacenterID != "" {
		tflog.Debug(ctx, fmt.Sprintf("Datacenter ID %s was specified, will filter later\n", datacenterID))
	}

	creds, err := getSecureConnectBundles(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}
	setSecureConnectBundleData(ctx, d, databaseID, datacenterID, creds)
	return nil
}

func getSecureConnectBundles(ctx context.Context, client astra.ClientWithResponsesInterface, databaseID string) ([]astra.CredsURL, error) {
	// fetch all the download bundles
	allBool := true
	secureBundleParams := &astra.GenerateSecureBundleURLParams{
		All: &allBool,
	}
	secureBundlesResp, err := client.GenerateSecureBundleURLWithResponse(ctx, databaseID, secureBundleParams)
	if err != nil {
		return nil, err
	}
	if secureBundlesResp.StatusCode() > http.StatusOK {
		return nil, fmt.Errorf("Failed to generate Secure Connect Bundle for Database ID %s. Response code: %d, msg = %s", databaseID, secureBundlesResp.StatusCode(), string(secureBundlesResp.Body))
	}
	return *secureBundlesResp.JSON200, nil
}

func setSecureConnectBundleData(ctx context.Context, d *schema.ResourceData, databaseID, datacenterID string, bundleData []astra.CredsURL) error {
	bundles := make([]map[string]interface{}, 0, len(bundleData))
	downloadURLs := make([]string, len(bundleData))
	for _, bundle := range(bundleData) {
		// DevOps APi has a misspelling that they might fix
		var bundleDatacenter string
		if bundle.DatacenterID != nil {
			bundleDatacenter = *bundle.DatacenterID
		} else if bundle.DatcenterID != nil {
			bundleDatacenter = *bundle.DatcenterID
		}
		if datacenterID != "" && bundleDatacenter != datacenterID {
			// skip adding this one because it doesn't match
			tflog.Debug(ctx, fmt.Sprintf("Skipping SCB info for non-matching DC: %s\n", bundleDatacenter))
			continue
		}
		bundleMap := map[string]interface{}{
			"datacenter_id":                bundleDatacenter,
			"url":                          bundle.DownloadURL,
			"internal_url":                 bundle.DownloadURLInternal,
			"migration_proxy_url":          bundle.DownloadURLMigrationProxy,
			"internal_migration_proxy_url": bundle.DownloadURLMigrationProxyInternal,
		}
		downloadURLs = append(downloadURLs, bundleMap["url"].(string))
		// see if the bundle has custom domain info
		if bundle.CustomDomainBundles != nil {
			customDomainBundleArray := *bundle.CustomDomainBundles
			customDomains := make([]map[string]interface{}, 0, len(customDomainBundleArray))
			for _, customDomain := range(customDomainBundleArray) {
				customDomainMap := map[string]interface{}{
					"domain": customDomain.Domain,
					"url":    customDomain.DownloadURL,
					"api_fqdn": customDomain.ApiFQDN,
					"cql_fqdn": customDomain.CqlFQDN,
					"dashboard_fqdn": customDomain.DashboardFQDN,
				}
				customDomains = append(customDomains, customDomainMap)
			}
			bundleMap["custom_domain_bundles"] = customDomains
		}
		bundles = append(bundles, bundleMap)
	}
	// set the ID using the Database ID and the download URLs
	d.SetId(fmt.Sprintf("%s/secure-connect-bundle/%s", databaseID, keyFromStrings(downloadURLs)))
	d.Set("secure_bundles", bundles)
	return nil
}