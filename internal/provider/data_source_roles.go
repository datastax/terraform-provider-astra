package provider

import (
	"context"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceRoles() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_roles` provides a datasource for a list of Astra roles. This can be used to select roles within your Astra Organization.",

		ReadContext: dataSourceRolesRead,

		Schema: map[string]*schema.Schema{

			// Computed
			"results": {
				Type:        schema.TypeList,
				Description: "The list of Astra roles.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_id": {
							Description: "The role id.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"role_name": {
							Description: "The role name.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"description": {
							Description: "Role description",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"effect": {
							Description: "Role effect",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"resources": {
							Description: "Resources for which role is applicable",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"policy": {
							Description: "List of policies for the role.",
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

func dataSourceRolesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	resp, err := client.GetOrganizationRolesWithResponse(ctx)

	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return diag.Errorf("Unable to retrieve organization roles: (%s) %s", resp.Status(), string(resp.Body))
	}

	roleList := getRoleSlice(resp.JSON200)
	roles := make([]map[string]interface{}, 0, len(roleList))
	for _, v := range roleList {
		roles = append(roles, flattenRole(v))
	}
	d.SetId(id.UniqueId())
	if err := d.Set("results", roles); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getRoleSlice(roleResp *[]astra.Role) []astra.Role {
	if roleResp == nil {
		return nil
	}
	return *roleResp
}
