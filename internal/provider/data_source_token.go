package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceToken() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_token` provides a datasource that lists client tokens.",

		ReadContext: dataSourceTokenRead,

		Schema: map[string]*schema.Schema{
			// Required
			"client_id": {
				Description:  "Client ID, system generated",
				Type:         schema.TypeString,
				Required:     true,
			},

			// Computed
			"results": {
				Type:        schema.TypeList,
				Description: "The list of roles for the specified client id.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"client_id": {
							Description:  "Role name",
							Type:         schema.TypeString,
							Required:     true,
							ForceNew: true,
						},
						"roles": {
							Description:  "Roles for this client",
							Type:         schema.TypeList,
							Required:     true,
							ForceNew: true,
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

func dataSourceTokenRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	fmt.Printf("token data source")

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	clientID := d.Get("client_id").(string)

	token, err := listToken(ctx, client, clientID)
	if err != nil {
		return diag.FromErr(err)
	}

	id := token["client_id"].(string)
	d.SetId(id)
	if err := d.Set("results", token); err != nil {
		return diag.FromErr(err)
	}


	return nil
}

func listToken(ctx context.Context, client *astra.ClientWithResponses, clientID string) (map[string]interface{}, error) {
	resp, err := client.GetClientsForOrgWithResponse(ctx)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("Error fetching client tokens, Status code: %d, msg: %s", resp.StatusCode(), string(resp.Body))
	}

	tokens := (*resp.JSON200).(map[string]interface{})["clients"].([]interface{})

	for _, v := range tokens {
		token := v.(map[string]interface{})
		if strings.EqualFold(token["clientId"].(string), clientID) {
			token["client_id"] = token["clientId"]
			delete(token, "clientId")
			return token, nil
		}
	}

	return nil, err
}