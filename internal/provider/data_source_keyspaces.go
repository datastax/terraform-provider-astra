package provider

import (
	"context"
	"fmt"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceKeyspaces() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_keyspaces` provides a datasource that lists the keyspaces in an Astra database.",

		ReadContext: dataSourceKeyspacesRead,

		Schema: map[string]*schema.Schema{
			// Required
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Computed
			"results": {
				Type:        schema.TypeList,
				Description: "The list of keyspaces that match the search criteria.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description: "The keyspace name.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func dataSourceKeyspacesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID := d.Get("database_id").(string)

	keyspaces, err := listKeyspaces(ctx, client, databaseID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("results", keyspacesToMap(keyspaces)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func listKeyspaces(ctx context.Context, client *astra.ClientWithResponses, databaseID string) ([]string, error) {
	resp, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
	if err != nil {
		return nil, err
	}

	db := resp.JSON200
	if db == nil {
		return nil, fmt.Errorf("error fetching database: %s", string(resp.Body))
	}

	// If the database is terminated then the keyspaces have been deleted.
	if db.Status == astra.StatusEnumTERMINATING || db.Status == astra.StatusEnumTERMINATED {
		return nil, nil
	}

	allKeyspaces := astra.StringSlice(db.Info.AdditionalKeyspaces)
	if k := astra.StringValue(db.Info.Keyspace); k != "" {
		allKeyspaces = append(allKeyspaces, k)
	}

	return allKeyspaces, nil
}

func keyspacesToMap(keyspaces []string) []map[string]interface{} {
	results := make([]map[string]interface{}, 0, len(keyspaces))
	for _, n := range keyspaces {
		results = append(results, map[string]interface{}{
			"name": n,
		})
	}
	return results
}
