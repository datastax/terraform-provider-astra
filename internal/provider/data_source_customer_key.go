package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceCustomerKey() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieve a Customer Key for a given cloud provider and region",

		ReadContext: dataSourceCustomerKeyRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"cloud_provider": {
				Description:      "The cloud provider where the Customer Key exists (Currently supported: aws, gcp)",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.StringInSlice(availableBYOKCloudProviders, true),
				DiffSuppressFunc: ignoreCase,
			},
            "region": {
				Description:      "Cloud provider region",
				Type:             schema.TypeString,
				Required:         true,
			},
			// Computed outputs
			"organization_id": {
				Description: "Organization ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"key_id": {
				Description: "The Customer Key ID",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func dataSourceCustomerKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	cloudProvider := d.Get("cloud_provider").(string)
	region := d.Get("region").(string)

	customerKeys, err := listCustomerKeys(ctx, client)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range customerKeys {
		if strings.EqualFold(cloudProvider, key["cloud_provider"].(string)) &&
		    region == key["region"].(string) {
				orgId := key["organization_id"].(string)
				keyId := key["key_id"].(string)
				d.Set("organization_id", orgId)
				d.Set("key_id", keyId)
				d.SetId(fmt.Sprintf("%s/cloudProvider/%s/region/%s/keyId/%s", orgId, cloudProvider, region, keyId))
				return nil
		}
	}
	// key not found
	return diag.Errorf("No Customer Key found for provider: %s, region: %s", cloudProvider, region)
}
