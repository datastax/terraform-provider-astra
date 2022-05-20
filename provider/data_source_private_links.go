package provider

import (
	"context"
	"fmt"
	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePrivateLinks() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_private_links` provides a datasource that lists the private links in an Astra database.",

		ReadContext: dataSourcePrivateLinksRead,

		Schema: map[string]*schema.Schema{
			// Required
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			// Required
			"datacenter_id": {
				Description:  "The datacenter where of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
			},

			// Computed
			"results": {
				Type:        schema.TypeList,
				Description: "The list of private links that match the search criteria.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"service_name": {
							Description: "Service name.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"datacenter_id": {
							Description: "DataCenter ID.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"endpoints": {
							Description: "Endpoints.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"allowed_principals": {
							Description: "Allowed principals.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
		},
	}
}

func dataSourcePrivateLinksRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)
	datacenterID := d.Get("datacenter_id").(string)

	privateLinks, err := listPrivateLinks(ctx, client, databaseID, datacenterID)
	if err != nil {
		return diag.FromErr(err)
	}
	if privateLinks == nil || privateLinks.AllowedPrincipals == nil {
		return nil
	}

	plMap := privateLinksToMap(privateLinks)

	d.SetId(resource.UniqueId())
	if err := d.Set("results", plMap); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func listPrivateLinks(ctx context.Context, client *astra.ClientWithResponses, databaseID string, datacenterID string) (*astra.PrivateLinkDatacenterOutput, error) {
	resp, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
	if err != nil {
		return nil, err
	}

	db := resp.JSON200
	if db == nil {
		return nil, fmt.Errorf("error fetching database: %s", string(resp.Body))
	}

	// If the database is terminated then the private links have been deleted.
	if db.Status == astra.StatusEnumTERMINATING || db.Status == astra.StatusEnumTERMINATED {
		return nil, nil
	}

	plResponse, err := client.GetPrivateLinksForDatacenterWithResponse(ctx, databaseID, datacenterID)

	privateLinkOutput := plResponse.JSON200


	return privateLinkOutput, err
}

func privateLinksToMap(privateLinks *astra.PrivateLinkDatacenterOutput) []map[string]interface{} {
	allowedPrincipals := *privateLinks.AllowedPrincipals
	endpoints := make([]astra.PrivateLinkEndpoint, 0)
	if privateLinks.Endpoints != nil {
		endpoints = *privateLinks.Endpoints
	}

	var apList = make([]string, len(allowedPrincipals))
	for i, n := range allowedPrincipals{
		apList[i] = string(n)
	}

	var endpointList = make([]string, len(endpoints))
	for i, n := range endpoints{
		endpointList[i] = string(*n.EndpointID)
	}


	results := make([]map[string]interface{}, 0, 1)
	results = append(results, map[string]interface{}{
		"service_name": string(*privateLinks.ServiceName),
		"datacenter_id": string(*privateLinks.DatacenterID),
		"allowed_principals": apList,
		"endpoints": endpointList,
	})

	return results
}
