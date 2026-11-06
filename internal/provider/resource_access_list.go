package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const accessListCreateTimeout = 20 * time.Second

func resourceAccessList() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_access_list` resource represents a database access list, used to limit the ip's / CIDR groups that have access to a database.",
		CreateContext: resourceAccessListCreate,
		ReadContext:   resourceAccessListRead,
		UpdateContext: resourceAccessListUpdate,
		DeleteContext: resourceAccessListDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				ForceNew:     true,
			},
			"addresses": {
				Description: "Set of address requests that should have access to database endpoints.",
				Type:        schema.TypeSet,
				Optional:    true,
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
			"enabled": {
				Description: "Public access restrictions enabled or disabled",
				Type:        schema.TypeBool,
				Optional:    true,
			},
		},
	}
}

func resourceAccessListCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)

	if err := upsertAccessList(ctx, client, databaseID, d); err != nil {
		return diag.Errorf("error creating access list: %s", err)
	}

	d.SetId(databaseID)

	// The API can briefly report a newly created access list as not found.
	err := retry.RetryContext(ctx, accessListCreateTimeout, func() *retry.RetryError {
		accessList, err := listAccessList(ctx, client, databaseID)
		if err != nil {
			return retry.NonRetryableError(err)
		} else if accessList == nil {
			return retry.RetryableError(errors.New("waiting for access list to be created"))
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAccessListUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID, err := parseAccessListID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := upsertAccessList(ctx, client, databaseID, d); err != nil {
		return diag.Errorf("error updating access list: %s", err)
	}

	return nil
}

// upsertAccessList replaces the entire access list (addresses and configuration) in a single call.
func upsertAccessList(ctx context.Context, client *astra.ClientWithResponses, databaseID string, d *schema.ResourceData) error {
	addressList := getAddressList(d.Get("addresses").(*schema.Set).List())

	resp, err := client.UpsertAccessListForDatabaseWithResponse(ctx,
		databaseID,
		astra.UpsertAccessListForDatabaseJSONRequestBody{
			Addresses: &addressList,
			Configurations: &astra.AccessListConfigurations{
				AccessListEnabled: d.Get("enabled").(bool),
			},
		},
	)

	if err != nil {
		return err
	} else if resp.StatusCode() >= 400 {
		return fmt.Errorf("%s: %s", resp.Status(), string(resp.Body))
	}
	return nil
}

func resourceAccessListDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID, err := parseAccessListID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}
	if accessList == nil {
		return nil
	}

	updResp, err := client.UpdateAccessListForDatabaseWithResponse(ctx,
		databaseID,
		astra.UpdateAccessListForDatabaseJSONRequestBody{
			Configurations: &astra.AccessListConfigurations{
				AccessListEnabled: false,
			},
		},
	)
	if err != nil {
		return diag.FromErr(err)
	} else if updResp.StatusCode() >= 400 {
		return diag.Errorf("error disabling access list: %s", string(updResp.Body))
	}

	delResp, err := client.DeleteAddressesOrAccessListForDatabaseWithResponse(ctx,
		databaseID,
		&astra.DeleteAddressesOrAccessListForDatabaseParams{},
	)
	if err != nil {
		return diag.FromErr(err)
	} else if delResp.StatusCode() >= 400 {
		return diag.Errorf("error deleting access list: %s", string(delResp.Body))
	}

	return nil
}

func resourceAccessListRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	id := d.Id()

	databaseID, err := parseAccessListID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	if accessList != nil && *accessList.DatabaseId == databaseID {
		if err := setAccessListData(d, accessList); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Not found. Remove from state.
	d.SetId("")

	return nil
}

func setAccessListData(d *schema.ResourceData, accessList *astra.AccessListResponse) error {
	if err := d.Set("database_id", *accessList.DatabaseId); err != nil {
		return err
	}
	if err := d.Set("enabled", accessList.Configurations.AccessListEnabled); err != nil {
		return err
	}
	addresses := *accessList.Addresses
	requests := make([]map[string]any, 0, len(addresses))
	for _, addr := range addresses {
		reqMap := map[string]any{
			"address":     *addr.Address,
			"description": *addr.Description,
			"enabled":     *addr.Enabled,
		}
		requests = append(requests, reqMap)
	}

	if err := d.Set("addresses", requests); err != nil {
		return err
	}
	return nil
}

func parseAccessListID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "", errors.New("invalid access list id format: expected databaseId/")
	}
	return idParts[0], nil
}

func getAddressList(addresses []any) []astra.AddressRequest {
	addressList := make([]astra.AddressRequest, len(addresses))
	for index, addrRequest := range addresses {
		address := addrRequest.(map[string]any)
		addressList[index] = astra.AddressRequest{
			Address:     address["address"].(string),
			Enabled:     address["enabled"].(bool),
			Description: address["description"].(string),
		}
	}
	return addressList
}
