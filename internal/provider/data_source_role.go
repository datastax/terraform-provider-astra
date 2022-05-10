package provider

import (
	"context"
	"fmt"
	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_role` provides a datasource that lists the custom roles for an org.",

		ReadContext: dataSourceRoleRead,

		Schema: map[string]*schema.Schema{
			// Required
			"role_id": {
				Description:  "Role ID, system generated",
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
						"role_name": {
							Description:  "Role name",
							Type:         schema.TypeString,
							Required:     true,
							ForceNew: true,
						},
						"description": {
							Description:  "Role description",
							Type:         schema.TypeString,
							Required:     true,
							ForceNew: true,
						},
						"effect": {
							Description:  "Role effect",
							Type:         schema.TypeString,
							Required:     true,
							ForceNew: true,
						},
						"resources": {
							Description:  "Resources for which role is applicable",
							Type:         schema.TypeList,
							Required:     true,
							ForceNew: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},

						"policy": {
							Description:  "List of policies for the role.",
							Type:         schema.TypeList,
							Required:     true,
							ForceNew:     true,
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

func dataSourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fmt.Printf("role data source")

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	roleID := d.Get("role_id").(string)

	role, err := listRole(ctx, client, roleID)
	if err != nil {
		return diag.FromErr(err)
	}

	id := role["id"].(string)
	d.SetId(id)
	if err := d.Set("results", role); err != nil {
		return diag.FromErr(err)
	}


	return nil
}

func listRole(ctx context.Context, client *astra.ClientWithResponses, roleID string) (map[string]interface{}, error) {
	resp, err := client.GetOrganizationRoleWithResponse(ctx, astra.RoleIdParam(roleID))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() > 200 {
		return nil, fmt.Errorf("Fetching role \"%s\" was not successful. Message: %s", roleID, string(resp.Body))
	}
	roleRaw := (*resp.JSON200).(map[string]interface{})

	return roleRaw, err
}