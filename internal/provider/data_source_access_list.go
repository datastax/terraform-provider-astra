package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
							Description: "IP Address/CIDR group that should have access",
							Type:        schema.TypeString,
							Required:    true,
						},
						"description": {
							Description: "Description for the IP Address/CIDR group",
							Type:        schema.TypeString,
							Optional:    true,
						},
						"enabled": {
							Description: "Enable/disable this IP Address/CIDR group's access",
							Type:        schema.TypeBool,
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAccessListRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {

	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(databaseID)

	if accessList == nil {
		accessList = &astra.AccessListResponse{
			DatabaseId:     &databaseID,
			Addresses:      &[]astra.AddressResponse{},
			Configurations: &astra.AccessListConfigurations{AccessListEnabled: false},
		}
	}

	if err := setAccessListData(d, accessList); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func listAccessList(ctx context.Context, client *astra.ClientWithResponses, databaseID string) (*astra.AccessListResponse, error) {
	dbResp, err := client.GetDatabaseWithResponse(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	db := dbResp.JSON200
	if db == nil {
		return nil, fmt.Errorf("error fetching database: %s", string(dbResp.Body))
	}

	if db.Status == astra.TERMINATING || db.Status == astra.TERMINATED {
		return nil, nil
	}

	// no with-response since astra returns a plain string on some errors for some reason
	resp, err := client.GetAccessListForDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %v, message: '%s'", resp.StatusCode, body)
	}
	accessList := astra.AccessListResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&accessList); err != nil {
		return nil, fmt.Errorf("failed to parse access list response: %w", err)
	}

	return &accessList, nil
}
