package provider

import (
	"context"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAvailableRegions() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieve a list of available cloud regions in Astra",

		ReadContext: dataSourceRegionsRead,

		Schema: map[string]*schema.Schema{
			"results": {
				Type:        schema.TypeList,
				Description: "The list of supported Astra regions by cloud provider and tier.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"display_name": {
							Description: "display name",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"zone": {
							Description: "zone",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"cloud_provider": {
							Description: "The cloud provider",
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

func dataSourceRegionsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	regionsResp, err := client.ListServerlessRegionsWithResponse(ctx)
	if err != nil {
		return diag.FromErr(err)
	} else if regionsResp.StatusCode() != 200 {
		return diag.Errorf("unexpected list available regions response: %s", string(regionsResp.Body))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	regions := *regionsResp.JSON200
	flatRegions := make([]map[string]interface{}, 0, len(regions))
	for _, region := range regions {
		flatRegions = append(flatRegions, flattenRegion(&region))
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("results", flatRegions); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func flattenRegion(region *astra.ServerlessRegion) map[string]interface{} {
	return map[string]interface{}{
		"cloud_provider": region.CloudProvider,
		"region": region.Name,
		"zone": region.Zone,
		"display_name": region.DisplayName,
	}
}
