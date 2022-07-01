package provider

import (
	"net/http"
	"context"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceUsers() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_users` provides a datasource for a list of Astra users. This can be used to select users within your Astra Organization.",

		ReadContext: dataSourceUsersRead,

		Schema: map[string]*schema.Schema{
			// Computed
			"org_id": {
				Type:        schema.TypeString,
				Description: "Organization ID.",
				Computed:    true,
			},
			"org_name": {
				Description: "Organization Name.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"users": {
				Type:        schema.TypeList,
				Description: "The list of Astra users.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"user_id": {
							Description: "The user id.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"email": {
							Description: "The user's email address.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"status": {
							Description: "User's status",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"roles": {
							Description: "Roles associated with the user",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"role_id": {
										Description: "The role id.",
										Type:        schema.TypeString,
										Computed:    true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceUsersRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	resp, err := client.GetOrganizationUsersWithResponse(ctx)

	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return diag.Errorf("Unable to retrieve organization users: (%s) %s", resp.Status(), string(resp.Body))
	}

	orgUsers := *resp.JSON200
	orgId := orgUsers.OrgID
	orgName := orgUsers.OrgName
	userList := resp.JSON200.Users
	users := make([]map[string]interface{}, 0, len(userList))
	for _, v := range userList {
		user, err := flattenUser(ctx, client, v)
		if err != nil {
			return err
		}
		users = append(users, user)
	}
	d.SetId(resource.UniqueId())
	d.Set("org_id", orgId)
	d.Set("org_name", orgName)
	if err := d.Set("users", users); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func flattenUser(ctx context.Context, client *astra.ClientWithResponses, user astra.UserResponse) (map[string]interface{}, diag.Diagnostics) {
	flatUser := map[string]interface{}{
		"user_id": user.UserID,
		"status":  *user.Status,
		"email":   *user.Email,
		"roles":   []map[string]interface{}{},
	}

	if user.Roles != nil {
		roles := make([]map[string]interface{}, 0, len(*user.Roles))
		for _, p := range *user.Roles {
			// The Roles on a UserResponse will only have the Role ID, not any of the Role details
			flatRole := make(map[string]interface{}, 1)
			flatRole["role_id"] = *p.Id
			roles = append(roles, flatRole)
		}
		flatUser["roles"] = roles
	}

	return flatUser, nil
}