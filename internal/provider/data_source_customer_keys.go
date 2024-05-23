package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceCustomerKeys() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieve a list of Customer Keys within an Organization",

		ReadContext: dataSourceCustomerKeysRead,

		Schema: map[string]*schema.Schema{
			"results": {
				Type:        schema.TypeList,
				Description: "The list of Customer Keys for the given Organization.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"organization_id": {
							Description: "Organization ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"cloud_provider": {
							Description: "The cloud provider",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"key_id": {
							Description: "The Customer Key ID",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"region": {
							Description: "The cloud provider region",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceCustomerKeysRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	customerKeys, err := listCustomerKeys(ctx, client)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("results", customerKeys); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(id.UniqueId())
	return nil
}

func listCustomerKeys(ctx context.Context, client *astra.ClientWithResponses) ([]map[string]interface{}, error) {
	resp, err := client.ListKeysWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Error fetching Customer Keys: %s", string(resp.Body))
	}
	customerKeys := resp.JSON200
	result := make([]map[string]interface{}, 0, len(*customerKeys))
	for _, key := range *customerKeys {
		result = append(result, map[string]interface{}{
			"organization_id" : *key.OrganizationID,
			"cloud_provider"  : *key.CloudProvider,
			"region"          : *key.Region,
			"key_id"          : *key.KeyID,
		})
	}
	return result, nil
}
