package provider

import (
	"context"
	"fmt"
	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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
			"enabled": {
				Description: "The Access list is enabled or disabled.",
				Type:        schema.TypeBool,
				Computed:    true,
			},
			"addresses": {
				Description: "Addresses in the access list.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description:  "IP Address/CIDR group that should have access",
							Type:         schema.TypeString,
							Required:     true,
						},
						"description": {
							Description:  "Description for the IP Address/CIDR group",
							Type:         schema.TypeString,
							Optional:     true,
						},
						"enabled": {
							Description:  "Enable/disable this IP Address/CIDR group's access",
							Type:         schema.TypeBool,
							Required:     true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAccessListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fmt.Printf("testing")

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := d.Get("database_id").(string)

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}
	if accessList == nil {
		return nil
	}

	d.SetId(databaseID)
	if err := setAccessListData(d, accessList); err != nil {
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
	if db.Status == astra.TERMINATING || db.Status == astra.TERMINATED {
		return nil, nil
	}

	alResponse, err := client.GetAccessListForDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))

	if err != nil {
		return nil, err
	}

	accessListOutput := alResponse.JSON200


	return accessListOutput, err
}
