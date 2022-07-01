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
			"role_name": {
				Description:  "Role name",
				Type:         schema.TypeString,
				Computed:     true,
			},
			"description": {
				Description:  "Role description",
				Type:         schema.TypeString,
				Computed:     true,
			},
			"effect": {
				Description:  "Role effect",
				Type:         schema.TypeString,
				Computed:     true,
			},
			"resources": {
				Description:  "Resources for which role is applicable (format is \"drn:astra:org:<org UUID>\", followed by optional resource criteria. See example usage above).",
				Type:         schema.TypeList,
				Computed:     true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateDiagFunc: validateRoleResources,
				},
			},
			"policy": {
				Description:  "List of policies for the role. See https://docs.datastax.com/en/astra/docs/user-permissions.html#_operational_roles_detail for supported policies.",
				Type:         schema.TypeList,
				Computed:     true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
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

	d.SetId(roleID)
	if err := setRoleData(d, role); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func listRole(ctx context.Context, client *astra.ClientWithResponses, roleID string) (*astra.Role, error) {
	resp, err := client.GetOrganizationRoleWithResponse(ctx, astra.RoleIdParam(roleID))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() > 200 {
		return nil, fmt.Errorf("Fetching role \"%s\" was not successful. Message: %s", roleID, string(resp.Body))
	}

	return resp.JSON200, err
}

func flattenRole(role astra.Role) map[string]interface{} {
	flatRole := map[string]interface{}{
		"role_id":     *role.Id,
		"role_name":   *role.Name,
		"description": "",
		"effect":      "",
		"resources":   []string{},
		"policy":      []string{},
	}

	if role.Policy != nil {
		policy := *role.Policy
		flatRole["description"] = policy.Description
		flatRole["effect"] = policy.Effect
		flatRole["resources"] = policy.Resources
		if policy.Actions != nil {
			policies := make([]string, len(policy.Actions))
			for index, p := range policy.Actions {
				policies[index] = string(p)
			}
			flatRole["policy"] = policies
		}
	}

	return flatRole
}

func setRoleData(d *schema.ResourceData, role *astra.Role) error {
	flatRole := flattenRole(*role)
	for k, v := range flatRole {
		if err := d.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}