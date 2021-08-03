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

func dataSourceAccessList() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_access_list` provides a datasource that lists the access lists for an Astra database.",

		ReadContext: dataSourceAccessListRead,

		Schema: map[string]*schema.Schema{
			// Required
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Computed
			"results": {
				Type:        schema.TypeList,
				Description: "The list of private links that match the search criteria.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"puppies": {
							Description: "The private link service name.",
							Type:        schema.TypeString,
							Computed:    true,
						},

					},
				},
			},
		},
	}
}

func dataSourceAccessListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fmt.Printf("testing")

	client := meta.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}
	if accessList == nil {
		return nil
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("results", accessListToMaps(accessList)); err != nil {
		fmt.Printf("testing")
		return diag.FromErr(err)
	}

	return nil
}

func listAccessList(ctx context.Context, client *astra.ClientWithResponses, databaseID string) (*astra.AccessListResponse, error) {
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

	alResponse, err := client.GetAccessListForDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))

	if err != nil {
		return nil, err
	}

	accessListOutput := alResponse.JSON200


	return accessListOutput, err
}

func accessListToMaps(accessList *astra.AccessListResponse) []map[string]interface{} {
	configurations := *accessList.Configurations
	databaseId := *accessList.DatabaseId
	organizationId := *accessList.OrganizationId
	addresses := *accessList.Addresses

	var addressList = make([]string, len(addresses))
	for i, n := range addresses{
		addressList[i] = string(*n.Address)
	}


	results := make([]map[string]interface{}, 0, 1)
	results = append(results, map[string]interface{}{
		"enabled": configurations.AccessListEnabled,
		"datacenterID": databaseId,
		"organizationId": organizationId,
		"addresses": addresses,
	})

	return results
}
