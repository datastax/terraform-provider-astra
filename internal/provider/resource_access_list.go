package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceAccessList() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_access_list` resource represents a database access list, used to limit the ip's / CIDR groups that have access to a database.",
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
				ForceNew: true,
			},
			"addresses": {
				Description:  "List of address requests that should have access to database endpoints.",
				Type:         schema.TypeList,
				Required:     true,
				ForceNew:     true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"request": {
							Type:  schema.TypeSet,
							Required:     true,
							Elem:  &schema.Resource{
								Schema: map[string]*schema.Schema{
									"address": {
										Description:  "Address",
										Type:         schema.TypeString,
										Required:     true,
									},
									"description": {
										Description:  "Description",
										Type:         schema.TypeString,
										Optional:     true,
									},
									"enabled": {
										Description:  "Description",
										Type:         schema.TypeBool,
										Required:     true,
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

func resourceAccessListCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)
	addresses := d.Get("addresses")

	addressList := make([]astra.AddressRequest, len(addresses.([]interface{})))

	for k, v := range d.Get("addresses").([]interface{}) {
		a := v.(map[string]interface{})["request"]
		request := a.(*schema.Set)
		for _, val := range request.List() {
			rMap := val.(map[string]interface{})
			addressList[k] = astra.AddressRequest{
				Address:    rMap["address"].(string),
				Enabled: rMap["enabled"].(bool),
				Description:  rMap["description"].(string),
			}

		}
	}

	resp, err := client.AddAddressesToAccessListForDatabaseWithResponse(ctx,
		astra.DatabaseIdParam(databaseID),
		addressList,
	)

	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() >= 400 {
		return diag.Errorf("error adding private link to database: %s", resp.Body)
	}

	if err := setAccessListData(d, databaseID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceAccessListDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*astra.ClientWithResponses)

	id := d.Id()

	databaseID, err := parseAccessListID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	aResp := *accessList.Addresses
	addressesQP := astra.AddressesQueryParam{}
	for _, v := range aResp {
		address:= *v.Address
		addressesQP = append(addressesQP, address)
	}

	params :=  &astra.DeleteAddressesOrAccessListForDatabaseParams{
		&addressesQP,
	}
	client.DeleteAddressesOrAccessListForDatabase(ctx, astra.DatabaseIdParam(databaseID), params)

	return nil
}

func resourceAccessListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*astra.ClientWithResponses)

	id := d.Id()

	databaseID, err := parseAccessListID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	accessList, err := listAccessList(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	if string(*accessList.DatabaseId) == databaseID {
		if err := setAccessListData(d, databaseID); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Not found. Remove from state.
	d.SetId("")

	return nil
}

func setAccessListData(d *schema.ResourceData, databaseID string) error {
	d.SetId(fmt.Sprintf("%s", databaseID))

	if err := d.Set("database_id", databaseID); err != nil {
		return err
	}

	return nil
}

func parseAccessListID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  errors.New("invalid access list id format: expected databaseId/")
	}
	return idParts[0],  nil
}
