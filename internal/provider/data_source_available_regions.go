package provider

import (
	"context"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var regionTypes = []string{"serverless", "vector", "all"}

func dataSourceAvailableRegions() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieve a list of available cloud regions in Astra",

		ReadContext: dataSourceRegionsRead,

		Schema: map[string]*schema.Schema{
			"cloud_provider": {
				Type:        schema.TypeString,
				Description: "The cloud provider to filter by",
				Optional:    true,
				ValidateFunc:     validation.StringInSlice(availableCloudProviders, true),
				DiffSuppressFunc: ignoreCase,
			},
			"region_type": {
				Type:        schema.TypeString,
				Description: "The region type to filter by (currently either 'serverless', 'vector' or 'all'). If omitted, the default is 'serverless'",
				Optional:    true,
				ValidateFunc:    validation.StringInSlice(regionTypes, true),
				DiffSuppressFunc: ignoreCase,
			},
			"only_enabled": {
				Type:        schema.TypeBool,
				Description: "Whether to filter by enabled regions. If 'false' or omitted, all regions are returned, enabled or not",
				Optional:    true,
				Default:     false,
			},
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
						"region_type": {
							Description: "The region type, either serverless or vector",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"enabled": {
							Description: "Whether the region is enabled",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						"reserved_for_qualified_users": {
							Description: "Whether the region is reserved for qualified users",
							Type:        schema.TypeBool,
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

	params := &astra.ListServerlessRegionsParams{}
	if d, ok := d.GetOk("region_type"); ok {
		regionType := d.(string)
		params.RegionType = &regionType
	}
	cloud_provider := d.Get("cloud_provider").(string)
	enabled := d.Get("only_enabled").(bool)
	regionsResp, err := client.ListServerlessRegionsWithResponse(ctx, params)
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
		if cloud_provider != "" && !strings.EqualFold(string(region.CloudProvider), cloud_provider) {
			// skip
			continue
		}
		if enabled && !*region.Enabled {
			// skip
			continue
		}
		flatRegions = append(flatRegions, flattenRegion(&region))
	}

	d.SetId(id.UniqueId())
	if err := d.Set("results", flatRegions); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func flattenRegion(region *astra.ServerlessRegion) map[string]interface{} {
	return map[string]interface{}{
		"cloud_provider":               region.CloudProvider,
		"region":                       region.Name,
		"zone":                         region.Zone,
		"display_name":                 region.DisplayName,
		"region_type":                  region.RegionType,
		"enabled":                      region.Enabled,
		"reserved_for_qualified_users": region.ReservedForQualifiedUsers,
	}
}
