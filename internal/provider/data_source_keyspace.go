package provider

import (
	"context"
	"fmt"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceKeyspace() *schema.Resource {
	return &schema.Resource{
		Description: "Keyspace for an Astra database.",

		ReadContext: dataSourceKeyspaceRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"name": {
				Description: "The keyspace name.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},
	}
}

func dataSourceKeyspaceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)
	keyspaceName := d.Get("name").(string)

	keyspaces, err := listKeyspaces(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, ks := range keyspaces {
		if ks == keyspaceName {
			d.SetId(fmt.Sprintf("%s/keyspace/%s", databaseID, ks))
			return nil
		}
	}

	d.SetId("")
	return nil
}
