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
			"datacenters": {
				Description: "List of Datacenter IDs",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description:  "Datacenter ID",
							Type:         schema.TypeString,
							Computed:     true,
						},
						"cloud_provider": {
							Description:  "The cloud provider in which the datacenter is deployed. (Currently supported: aws, azure, gcp)",
							Type:         schema.TypeString,
							Computed:     true,
						},
						"region": {
							Description:  "The region in which the datacenter is deployed. (see https://docs.datastax.com/en/astra/docs/database-regions.html for supported regions)",
							Type:         schema.TypeString,
							Computed:     true,
						},
					},
				},
			},
		},
	}
}

func dataSourceDatabaseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	databaseID := d.Get("database_id").(string)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	db, err := getDatabase(ctx, d, client, databaseID)
	if err != nil {
		diag.Errorf("error fetching database: %s", err.Error())
		return nil
	}

	if err := setDatabaseResourceData(d, db); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getDatabase(ctx context.Context, d *schema.ResourceData, client *astra.ClientWithResponses, databaseID string) (*astra.Database, error) {
	resp, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		d.SetId("")
		return nil, err
	}

	db := resp.JSON200
	if db == nil {
		diag.Errorf("error fetching database: %s", string(resp.Body))
		return nil, err
	}
	return db, nil
}