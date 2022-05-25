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

func dataSourcePrivateLinkEndpoints() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_private_link_endpoints` provides a datasource that lists the private link endpoints for an Astra database.",

		ReadContext: dataSourcePrivateLinkEndpointsRead,

		Schema: map[string]*schema.Schema{
			// Required
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datacenter_id": {
				Description:  "The Datacenter ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
			},
			"endpoint_id": {
				Description:  "Endpoint ID.",
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
						"endpoint_id": {
							Description: "The private link endpoint id",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"description": {
							Description: "Description for private link endpoint",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status": {
							Description: "Private link endpoint status",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"create_time": {
							Description: "Crate time for private link endpoint",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourcePrivateLinkEndpointsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fmt.Printf("testing")

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)
	datacenterID := d.Get("datacenter_id").(string)
	endpointID := d.Get("endpoint_id").(string)

	privateLinks, err := listPrivateLinkEndpoints(ctx, client, databaseID, datacenterID, endpointID)
	if err != nil {
		return diag.FromErr(err)
	}
	if privateLinks == nil {
		return nil
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("results", privateLinkEndpointsToMap(privateLinks)); err != nil {
		fmt.Printf("testing")
		return diag.FromErr(err)
	}

	return nil
}

func listPrivateLinkEndpoints(ctx context.Context, client *astra.ClientWithResponses, databaseID string, datacenterID string, endpointID string) (*astra.PrivateLinkEndpoint, error) {
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

	plResponse, err := client.GetPrivateLinkEndpointWithResponse(ctx, databaseID, datacenterID, endpointID)
	if err != nil{
		return nil, err
	}

	privateLinkEndpointOutput := plResponse.JSON200


	return privateLinkEndpointOutput, err
}

func privateLinkEndpointsToMap(privateLinkEndpoints *astra.PrivateLinkEndpoint) []map[string]interface{} {
	endpointID := privateLinkEndpoints.EndpointID
	description := privateLinkEndpoints.Description
	status := privateLinkEndpoints.Status
	createTime := privateLinkEndpoints.CreatedDateTime

	results := make([]map[string]interface{}, 0, 1)
	results = append(results, map[string]interface{}{
		"endpoint_id": endpointID,
		"description": description,
		"status": status,
		"create_time": createTime,
	})

	return results
}
