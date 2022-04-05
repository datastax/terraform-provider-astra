package provider

import (
	"context"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_database` provides a datasource for Astra an Astra database. This can be used to select an existing database within your Astra Organization.",

		ReadContext: dataSourceDatabaseRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"database_id": {
				Description:  "Astra Database ID (system generated)",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			// computed outputs
			"name": {
				Description: "Database name (user provided)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"owner_id": {
				Description: "Owner id (system generated)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"organization_id": {
				Description: "Ordg id (system generated)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cloud_provider": {
				Description: "Cloud provider (AWS, GCP, AZURE)",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"regions": {
				Description: "Cloud provider region. Get list of supported regions from regions data-source",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"status": {
				Description: "Database status",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cqlsh_url": {
				Description: "URL for cqlsh web",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"grafana_url": {
				Description: "URL for the grafana dashboard for this database",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_endpoint_url": {
				Description: "REST API URL",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"graphql_url": {
				Description: "Graphql URL",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"keyspace": {
				Description: "Initial keyspace",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"node_count": {
				Description: "Node count (not relevant for serverless databases)",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"replication_factor": {
				Description: "Replication Factor (not relevant for serverless databases)",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"total_storage": {
				Description: "Storage Capacity (not relevant for serverelss databases)",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"additional_keyspaces": {
				Description: "Additional keyspaces",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	databaseID := d.Get("database_id").(string)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	db, diagnostics, done := getDatabase(ctx, d, client, databaseID)
	if done {
		return diagnostics
	}

	if err := setDatabaseResourceData(d, db); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getDatabase(ctx context.Context, d *schema.ResourceData, client *astra.ClientWithResponses, databaseID string) (*astra.Database, diag.Diagnostics, bool) {
	resp, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
	if err != nil {
		return nil, diag.FromErr(err), true
	}
	if resp.StatusCode() == http.StatusNotFound {
		d.SetId("")
		return nil, nil, true
	}

	db := resp.JSON200
	if db == nil {
		return nil, diag.Errorf("error fetching database: %s", string(resp.Body)), true
	}
	return db, nil, false
}
