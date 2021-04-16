package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/joeandaverde/astra-client-go/v2/astra"
)

func resourceKeyspace() *schema.Resource {
	return &schema.Resource{
		Description: "Astra database Keyspace.",

		CreateContext: resourceKeyspaceCreate,
		ReadContext:   resourceKeyspaceRead,
		DeleteContext: resourceKeyspaceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"name": {
				Description:      "Keyspace name can have up to 48 alpha-numeric characters and contain underscores; only letters and numbers are supported as the first character.",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateKeyspace,
			},
			"database_id": {
				Description:  "Astra database to create the keyspace.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
		},
	}
}

func resourceKeyspaceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)
	keyspaceName := d.Get("name").(string)

	resp, err := client.AddKeyspaceWithResponse(ctx, astra.DatabaseIdParam(databaseID), astra.KeyspaceNameParam(keyspaceName))
	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() >= 400 {
		return diag.Errorf("error adding keyspace to database: %s", string(resp.Body))
	}

	if err := setKeyspaceResourceData(d, databaseID, keyspaceName); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceKeyspaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*astra.ClientWithResponses)

	id := d.Id()
	databaseID, keyspaceName, err := parseKeyspaceID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	keyspaces, err := listKeyspaces(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, k := range keyspaces {
		if k == keyspaceName {
			if err := setKeyspaceResourceData(d, databaseID, keyspaceName); err != nil {
				return diag.FromErr(err)
			}
			return nil
		}
	}

	// Keyspace not found. Remove from state.
	d.SetId("")

	return nil
}

func resourceKeyspaceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func setKeyspaceResourceData(d *schema.ResourceData, databaseID string, keyspaceName string) error {
	d.SetId(fmt.Sprintf("%s/keyspace/%s", databaseID, keyspaceName))
	if err := d.Set("name", keyspaceName); err != nil {
		return err
	}
	if err := d.Set("database_id", databaseID); err != nil {
		return err
	}

	return nil
}

func parseKeyspaceID(id string) (string, string, error) {
	idParts := strings.Split(strings.ToLower(id), "/keyspace/")
	if len(idParts) != 2 {
		return "", "", errors.New("invalid keyspace id format: expected database_id/keyspace")
	}
	return idParts[0], idParts[1], nil
}
