package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRole() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_role` resource represents custom roles for a particular Astra Org. Custom roles can be assigned to an Astra user is to grant them granular permissions when the default roles in the UI are not specific enough. Roles are composed of policies which are granted to resources.",
		CreateContext: resourceRoleCreate,
		ReadContext:   resourceRoleRead,
		DeleteContext: resourceRoleDelete,
		UpdateContext: resourceRoleUpdate,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
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
			},
			"effect": {
				Description:  "Role effect",
				Type:         schema.TypeString,
				Required:     true,
			},
			"resources": {
				Description:  "Resources for which role is applicable (format is \"drn:astra:org:<org UUID>\", followed by optional resource criteria. See example usage above).",
				Type:         schema.TypeList,
				Required:     true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateDiagFunc: validateRoleResources,
				},
			},

			"policy": {
				Description:  "List of policies for the role. See https://docs.datastax.com/en/astra/docs/user-permissions.html#_operational_roles_detail for supported policies.",
				Type:         schema.TypeList,
				Required:     true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"role_id": {
				Description:  "Role ID, system generated",
				Type:         schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceRoleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	roleName := d.Get("role_name").(string)
	description := d.Get("description").(string)
	effect := d.Get("effect").(string)
	resourcesRaw := d.Get("resources").([]interface{})
	policyRaw := d.Get("policy")

	actions := policyRaw.([]interface{})
	resourcesList := make([]string, len(resourcesRaw))
	actionsList := make([]astra.PolicyAction, len(actions))

	for k, v := range resourcesRaw {
		resourcesList[k] = v.(string)
	}
	for k, v := range policyRaw.([]interface{}) {
		actionsList[k] = astra.PolicyAction(v.(string))
	}
	policy := astra.Policy{
		Actions:     actionsList,
		Description: description,
		Effect:      astra.PolicyEffect(effect),
		Resources:   resourcesList,
	}


	roleJSON := astra.AddOrganizationRoleJSONRequestBody{
		Name:   roleName,
		Policy: policy,
	}
	resp, err := client.AddOrganizationRoleWithResponse(ctx,
		roleJSON,
	)

	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() >= 400 {
		return diag.Errorf("error adding role to org: Status: %s, %s", resp.Status(), resp.Body)
	}

	role := resp.JSON201
	if err := setRoleData(d, *role.Id); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceRoleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	id := d.Id()

	roleID, err := parseRoleID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	roleParam := astra.RoleIdParam(roleID)
	client.DeleteOrganizationRoleWithResponse(ctx, roleParam)

	return nil
}

func resourceRoleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	id := d.Id()

	roleID, err := parseRoleID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	role, err := listRole(ctx, client, roleID)
	if err != nil {
		return diag.FromErr(err)
	}


	if role["id"].(string) == roleID {
		if err := setRoleData(d, roleID); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Not found. Remove from state.
	d.SetId("")

	return nil
}

func resourceRoleUpdate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	id := resourceData.Id()

	// fetch the role
	roleID, err := parseRoleID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	oldRole, err := listRole(ctx, client, roleID)
	if err != nil {
		return diag.FromErr(err)
	}

	roleName := resourceData.Get("role_name").(string)
	description := resourceData.Get("description").(string)
	effect := resourceData.Get("effect").(string)
	resourcesRaw := resourceData.Get("resources").([]interface{})
	policyRaw := resourceData.Get("policy")

	actions := policyRaw.([]interface{})
	resourcesList := make([]string, len(resourcesRaw))
	actionsList := make([]astra.PolicyAction, len(actions))

	for k, v := range resourcesRaw {
		resourcesList[k] = v.(string)
	}
	for k, v := range policyRaw.([]interface{}) {
		actionsList[k] = astra.PolicyAction(v.(string))
	}
	policy := astra.Policy{
		Actions:     actionsList,
		Description: description,
		Effect:      astra.PolicyEffect(effect),
		Resources:   resourcesList,
	}

	roleJSON := astra.UpdateRoleJSONRequestBody{
		Name:   roleName,
		Policy: policy,
	}
	resp, err := client.UpdateRoleWithResponse(ctx,
		roleID,
		roleJSON,
	)

	if err != nil {
		revertRole(oldRole, resourceData)
		return diag.FromErr(err)
	} else if resp.StatusCode() >= 400 {
		revertRole(oldRole, resourceData)
		return diag.Errorf("error adding role to org: Status: %s, %s", resp.Status(), resp.Body)
	}

	return nil
}

func setRoleData(d *schema.ResourceData, roleID string) error {
	d.SetId(fmt.Sprintf("%s", roleID))

	if err := d.Set("role_id", roleID); err != nil {
		return err
	}

	return nil
}

func parseRoleID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  errors.New("invalid role id format: expected roleID/")
	}
	return idParts[0],  nil
}

func revertRole(oldRole map[string]interface{}, resourceData *schema.ResourceData) {
	oldRolePolicy := oldRole["policy"].(map[string]interface{})
	oldDescription := oldRolePolicy["description"]
	oldEffect := oldRolePolicy["effect"]
	oldResources := oldRolePolicy["resources"]
	oldPolicy := oldRolePolicy["actions"]
	resourceData.Set("description", oldDescription)
	resourceData.Set("effect", oldEffect)
	resourceData.Set("resources", oldResources)
	resourceData.Set("policy", oldPolicy)
}