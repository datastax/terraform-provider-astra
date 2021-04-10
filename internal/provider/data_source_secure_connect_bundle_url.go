package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/joeandaverde/astra-client-go/v2/astra"
)

func dataSourceSecureConnectBundleURL() *schema.Resource {
	return &schema.Resource{
		Description: "Generate a temporary secure connect bundle URL for an Astra database.",

		ReadContext: dataSourceSecureConnectBundleURLRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
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
	client := meta.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)

	resp, err := client.GenerateSecureBundleURLWithResponse(ctx, astra.DatabaseIdParam(databaseID))
	if err != nil {
		return diag.FromErr(err)
	}

	bundleURL := resp.JSON200
	if bundleURL == nil {
		return diag.Errorf("failed to get secure connect bundle: %s", string(resp.Body))
	}

	d.SetId(fmt.Sprintf("%s/secure-connect-bundle/%s", databaseID, keyFromStrings([]string{bundleURL.DownloadURL})))
	d.Set("url", bundleURL.DownloadURL)

	return nil
}
