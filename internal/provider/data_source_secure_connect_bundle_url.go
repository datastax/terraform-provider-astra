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
				Description:  "The ID of the Astra datacenter. If omitted, only the primary datacenter will be used.",
				Type:         schema.TypeString,
				Optional:     true,
			},
			// Computed
			"url": {
				Description: "The temporary download url to the secure connect bundle zip file.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceSecureConnectBundleURLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)
	datacenterID := d.Get("datacenter_id").(string)

	credsURL, err := getSecureConnectBundleURL(ctx, client, databaseID, datacenterID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/secure-connect-bundle/%s", databaseID, keyFromStrings([]string{credsURL.DownloadURL})))
	d.Set("url", credsURL.DownloadURL)
	return nil
}

func getSecureConnectBundleURL(ctx context.Context, client astra.ClientWithResponsesInterface, databaseID, datacenterID string) (*astra.CredsURL, error) {
	var credsURL *astra.CredsURL

	// fetch dataceneters for the specified DB ID
	datacenterResp, err := client.ListDatacentersWithResponse(ctx, databaseID, &astra.ListDatacentersParams{})
	if err != nil {
		return nil, err
	}
	if datacenterResp.StatusCode() > http.StatusOK {
		return nil, fmt.Errorf("Failed to retrieve datacenters for Database with ID: %s. Response code: %d, msg = %s", databaseID, datacenterResp.StatusCode(), string(datacenterResp.Body))
	}

	// if no datacenter ID specified, then use the primary datacenter ID, should be <dbid-1>
	if datacenterID == "" {
		datacenterID = databaseID+"-1"
	}

	// find the URL for the datacenter ID
	for _, dc := range *datacenterResp.JSON200 {
		if datacenterID == *dc.Id {
			return &astra.CredsURL{
				DownloadURL: *dc.SecureBundleUrl,
				DownloadURLInternal: dc.SecureBundleInternalUrl,
				DownloadURLMigrationProxy: dc.SecureBundleMigrationProxyUrl,
				DownloadURLMigrationProxyInternal: dc.SecureBundleMigrationProxyInternalUrl,
			}, nil
		}
	}

	tflog.Error(ctx, fmt.Sprintf("Could not find Datacenter with ID: %s", databaseID))
	return credsURL, nil
}

