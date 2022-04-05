package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
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

	credsURL, err := generateSecureBundleURL(ctx, time.Minute, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/secure-connect-bundle/%s", databaseID, keyFromStrings([]string{credsURL.DownloadURL})))
	d.Set("url", credsURL.DownloadURL)
	return nil
}

func generateSecureBundleURL(ctx context.Context, timeout time.Duration, client astra.ClientWithResponsesInterface, databaseID string) (*astra.CredsURL, error) {
	var credsURL *astra.CredsURL
	if err := resource.RetryContext(ctx, timeout, func() *resource.RetryError {
		resp, err := client.GenerateSecureBundleURLWithResponse(ctx, astra.DatabaseIdParam(databaseID))
		if err != nil || resp.StatusCode() >= 500 {
			return resource.RetryableError(err)
		}

		// 409 Conflict can be returned if the database is not yet ready
		if resp.JSON409 != nil {
			return resource.RetryableError(fmt.Errorf("cannot create secure bundle url: %s", string(resp.Body)))
		}

		// Any other 400 status code is not retried
		if resp.StatusCode() >= 400 {
			return resource.NonRetryableError(fmt.Errorf("error trying to create secure bundle url: %s", string(resp.Body)))
		}

		// Any response other than 200 is unexpected
		credsURL = resp.JSON200
		if credsURL == nil {
			return resource.NonRetryableError(fmt.Errorf("unexpected response creating secure bundle url: %s", string(resp.Body)))
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return credsURL, nil
}
