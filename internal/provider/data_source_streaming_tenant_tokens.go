package provider

import (
	"context"
	"fmt"
	"net/http"

	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStreamingTenantTokens() *schema.Resource {
	return &schema.Resource {
		Description: "`astra_streaming_tenant_tokens` provides a datasource that lists streaming tenant tokens.",

		ReadContext: dataSourceStreamingTenantTokensRead,

		Schema: map[string]*schema.Schema{
			// Required
			"cluster_name": {
				Description:  "Name of the Pulsar Cluster. Format: `pulsar-<cloud provider>-<cloud region>`. Example: `pulsar-gcp-useast1`",
				Type:         schema.TypeString,
				Required:     true,
			},
			"tenant_name": {
				Description:  "Name of the streaming tenant for which to fetch tokens.",
				Type:         schema.TypeString,
				Required:     true,
			},
			// Computed
			"tokens": {
				Type:        schema.TypeList,
				Description: "The list of tokens for the specified tenant.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"iat": {
							Description: "IAT of the token.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"iss": {
							Description: "ISS of the token.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"sub": {
							Description: "Client subscriber of the token.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"token_id": {
							Description: "ID of the token.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"token": {
							Description: "Pulsar JWT token.",
							Type:        schema.TypeString,
							Computed:    true,
							Sensitive:   true,
						},
					},
				},
			},
		},
	}
}

func dataSourceStreamingTenantTokensRead (ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	tenantName := d.Get("tenant_name").(string)
	clusterName := d.Get("cluster_name").(string)

	params := astrastreaming.IdListTenantTokensParams{
		XDataStaxPulsarCluster: clusterName,
	}
	tokenResponse, err := streamingClient.IdListTenantTokensWithResponse(ctx, tenantName, &params)

	if err != nil {
		return diag.FromErr(err)
	}

	if tokenResponse.StatusCode() != http.StatusOK {
		return diag.Errorf("Failed to get list of tenants. ResponseCode: %d, message = %s.", tokenResponse.StatusCode(), string(tokenResponse.Body))
	}

	if err := setTenantTokensData(ctx, d, streamingClient, *tokenResponse.JSON200); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(tenantName)

	return nil
}

func setTenantTokensData(ctx context.Context, d *schema.ResourceData, streamingClient *astrastreaming.ClientWithResponses, tokenList []astrastreaming.TenantToken) error {
	tenantName := d.Get("tenant_name").(string)
	clusterName := d.Get("cluster_name").(string)
	params := astrastreaming.GetTokenByIDParams{
		XDataStaxPulsarCluster: clusterName,
	}
	tokens := make([]map[string]interface{}, 0, len(tokenList))
	for _, token := range tokenList {
		tokenMap := map[string]interface{} {
			"token_id" : token.TokenID,
			"iat"      : token.Iat,
			"iss"      : token.Iss,
			"sub"      : token.Sub,
		}
		// now fetch the JWT token
		tokenResponse, err := streamingClient.GetTokenByIDWithResponse(ctx, tenantName, *token.TokenID, &params)
		if err != nil {
			return err
		}
		if tokenResponse.StatusCode() != http.StatusOK {
			return fmt.Errorf("Failed to fetch token by ID. Token ID: %s, Tenant Name: %s, Cluster Name: %s.", *token.TokenID, tenantName, clusterName)
		}
		tokenMap["token"] = string(tokenResponse.Body)
		tokens = append(tokens, tokenMap)
	}
	if err := d.Set("tokens", tokens); err != nil {
		return err
	}

	return nil
}