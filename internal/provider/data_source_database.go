package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/joeandaverde/astra-client-go/v2/astra"
)

func dataSourceDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "Datasource for Astra database.",

		ReadContext: dataSourceDatabaseRead,

		Schema: map[string]*schema.Schema{
			// Required inputs
			"database_id": {
				Description:  "The ID of the Astra database.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			// computed outputs
			"name": {
				Description: "The database name.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"owner_id": {
				Description: "The owner id.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"organization_id": {
				Description: "The org id.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cloud_provider": {
				Description: "The cloud provider",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"region": {
				Description: "The cloud provider region",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"status": {
				Description: "The status",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cqlsh_url": {
				Description: "The cqlsh_url",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"grafana_url": {
				Description: "The grafana_url",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_endpoint_url": {
				Description: "The data_endpoint_url",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"graphql_url": {
				Description: "The graphql_url",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"keyspace": {
				Description: "The keyspace",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"node_count": {
				Description: "The node_count",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"replication_factor": {
				Description: "The replication_factor",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"total_storage": {
				Description: "The total_storage",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"additional_keyspaces": {
				Description: "The total_storage",
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
	client := meta.(*astra.ClientWithResponses)

	resp, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	db := resp.JSON200
	if db == nil {
		return diag.Errorf("error fetching database: %s", string(resp.Body))
	}

	if err := setDatabaseResourceData(d, db); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
