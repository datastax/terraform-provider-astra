package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAccessList() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_access_list` resource represents a database access list, used to limit the ip's / CIDR groups that have access to a database.",
		CreateContext: resourceAccessListCreate,
		ReadContext:   resourceAccessListRead,
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
				Description: "List of address requests that should have access to database endpoints.",
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": {
							Description: "IP Address/CIDR group that should have access",
							Type:        schema.TypeString,
							Required:    true,
							ForceNew:    true,
						},
						"description": {
							Description: "Description for the IP Address/CIDR group",
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
						},
						"enabled": {
							Description: "Enable/disable this IP Address/CIDR group's access",
							Type:        schema.TypeBool,
							Required:    true,
							ForceNew:    true,
						},
					},
				},
			},
			"enabled": {
				Description: "Public access restrictions enabled or disabled",
				Type:        schema.TypeBool,
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceAccessListCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)
	addresses := d.Get("addresses").([]interface{})
	restricted := d.Get("enabled").(bool)
	addressList := getAddressList(addresses)

	addResp, err := client.AddAddressesToAccessListForDatabaseWithResponse(ctx,
		astra.DatabaseIdParam(databaseID),
		addressList,
	)

	if err != nil {
		return diag.FromErr(err)
	} else if addResp.StatusCode() >= 400 {
		return diag.Errorf("error adding access list to database: (%d) %s", addResp.StatusCode(), addResp.Body)
	}

	d.SetId(databaseID)

	accessListConfig := astra.AccessListConfigurations{AccessListEnabled: restricted}
	updResp, err := client.UpdateAccessListForDatabaseWithResponse(ctx,
		astra.DatabaseIdParam(databaseID),
		astra.UpdateAccessListForDatabaseJSONRequestBody{
			Addresses:      &addressList,
			Configurations: &accessListConfig,
		},
	)
	if err != nil {
		return diag.FromErr(err)
	} else if updResp.StatusCode() >= 400 {
		return diag.Errorf("error updating access list configuration: %d\n%s", updResp.StatusCode(), updResp.Body)
	}

	return nil
}

func resourceAccessListDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	// aResp := *accessList.Addresses
	// addressesQP := astra.AddressesQueryParam{}
	// for _, v := range aResp {
	// 	address:= *v.Address
	// 	addressesQP = append(addressesQP, address)
	// }

	// params :=  &astra.DeleteAddressesOrAccessListForDatabaseParams{
	// 	&addressesQP,
	// }
	// client.DeleteAddressesOrAccessListForDatabase(ctx, astra.DatabaseIdParam(databaseID), params)

	// The above code should work, but Astra seems to only delete the first address passed as a query param
	// Until it's fixed in Astra, call DELETE for each address
	aResp := *accessList.Addresses
	if len(aResp) > 0 {
		for _, v := range aResp {
			addressesQP := astra.AddressesQueryParam{*v.Address}
			params := &astra.DeleteAddressesOrAccessListForDatabaseParams{Addresses: &addressesQP}
			client.DeleteAddressesOrAccessListForDatabase(ctx, astra.DatabaseIdParam(databaseID), params)
		}
	} else {
		params := &astra.DeleteAddressesOrAccessListForDatabaseParams{Addresses: nil}
		client.DeleteAddressesOrAccessListForDatabase(ctx, astra.DatabaseIdParam(databaseID), params)
	}

	return nil
}

func resourceAccessListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
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

	if accessList != nil && string(*accessList.DatabaseId) == databaseID {
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
	requests := make([]map[string]interface{}, 0, len(addresses))
	for _, addr := range addresses {
		reqMap := map[string]interface{}{
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

func getAddressList(addresses []interface{}) []astra.AddressRequest {
	addressList := make([]astra.AddressRequest, len(addresses))
	for index, addrRequest := range addresses {
		address := addrRequest.(map[string]interface{})
		addressList[index] = astra.AddressRequest{
			Address:     address["address"].(string),
			Enabled:     address["enabled"].(bool),
			Description: address["description"].(string),
		}
	}
	return addressList
}
