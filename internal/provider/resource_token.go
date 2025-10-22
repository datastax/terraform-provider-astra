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

func resourceToken() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_token` resource represents a token with a specific role assigned.",
		CreateContext: resourceTokenCreate,
		ReadContext:   resourceTokenRead,
		DeleteContext: resourceTokenDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"roles": {
				Description: "List of Role IDs to be assigned to the generated token",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Optional
			"org_id": {
				Description: "The UUID of the organization under which the token will be created. If not provided, the token will be created under the organization/enterprise of the token making the request.",
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
			},
			"client_id": {
				Description: "Client id, use as username in cql to connect",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"secret": {
				Description: "Secret, use as password in cql to connect",
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
			},
			"token": {
				Description: "Token, use as auth bearer for API calls or as password in combination with the word `token` in cql",
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
			},
		},
	}
}

func resourceTokenCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	roles := d.Get("roles").([]interface{})
	orgId := d.Get("org_id").(string)

	if len(orgId) == 0 {
		// no orgId provided, use the one associated with the effective token
		currentOrg, err := getCurrentOrgID(ctx, client)
		if err != nil {
			return diag.Errorf("No Organization ID provided for token creation and an error occurred trying to fetch the Organization associated with the current API token.")
		}
		// use the org associated with the API token making the call if not provided
		orgId = currentOrg
	}

	rolesList := make([]string, len(roles))

	for k, v := range roles {
		roleId := v.(string)
		// ensure the role exists
		_, err := listRole(ctx, client, roleId)
		if err != nil {
			return diag.Errorf("Failed to create token. Role ID not found: %s", roleId)
		}
		rolesList[k] = roleId
	}

	tokenJSON := astra.GenerateTokenForClientJSONRequestBody{
		Roles: rolesList,
		OrgId: &orgId,
	}
	resp, err := client.GenerateAppTokenForClientWithResponse(ctx,
		tokenJSON,
	)

	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() >= 400 {
		return diag.Errorf("error adding role to org: %s", resp.Body)
	}

	token := (*resp.JSON200).(map[string]interface{})
	if err := setTokenData(d, token); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceTokenDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	id := d.Id()

	clientID, err := parseTokenID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	client.DeleteTokenForClient(ctx, astra.ClientIdParam(clientID))

	return nil
}

func resourceTokenRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	id := d.Id()

	clientID, err := parseTokenID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	token, err := listToken(ctx, client, clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s", clientID))
	if err := d.Set("client_id", token["client_id"]); err != nil {
		return diag.FromErr(err)
	}

	return nil

}

func setTokenData(d *schema.ResourceData, tokenMap map[string]interface{}) error {
	clientID := tokenMap["clientId"].(string)
	secret := tokenMap["secret"].(string)
	token := tokenMap["token"].(string)
	responseOrgId := tokenMap["orgId"].(string)

	d.SetId(fmt.Sprintf("%s", clientID))

	if err := d.Set("client_id", clientID); err != nil {
		return err
	}
	if err := d.Set("secret", secret); err != nil {
		return err
	}
	if err := d.Set("token", token); err != nil {
		return err
	}
	if err := d.Set("org_id", responseOrgId); err != nil {
		return err
	}

	return nil
}

func parseTokenID(id string) (string, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) != 1 {
		return "", errors.New("invalid token id format: expected clientID/")
	}
	return idParts[0], nil
}
