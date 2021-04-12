package provider

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/joeandaverde/astra-client-go/v2/astra"
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
						"tier": {
							Description: "Supported tier",
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
	client := meta.(*astra.ClientWithResponses)

	regionsResp, err := client.ListAvailableRegionsWithResponse(ctx)
	if err != nil {
		return diag.FromErr(err)
	} else if regionsResp.StatusCode() != 200 {
		return diag.Errorf("unexpected list available regions response: %s", string(regionsResp.Body))
	}

	if err != nil {
		return diag.FromErr(err)
	}

	regions := astra.AvailableRegionCombinationSlice(regionsResp.JSON200)
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


func flattenRegion(region *astra.AvailableRegionCombination) map[string]interface{} {
	return map[string]interface{}{
		"cloud_provider": region.CloudProvider,
		"region": region.Region,
		"tier":region.Tier,
	}
}