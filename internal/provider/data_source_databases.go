package provider

import (
	"context"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatabases() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_databases` provides a datasource for a list of Astra databases. This can be used to select databases within your Astra Organization.",

		ReadContext: dataSourceDatabasesRead,

		Schema: map[string]*schema.Schema{
			// Optional
			"status": {
				Type:        schema.TypeString,
				Description: "The list of Astra databases that match the search criteria.",
				Optional:    true,
			},
			"cloud_provider": {
				Description: "The cloud provider",
				Type:        schema.TypeString,
				Optional:    true,
			},

			// Computed
			"results": {
				Type:        schema.TypeList,
				Description: "The list of Astra databases that match the search criteria.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The database id.",
							Type:        schema.TypeString,
							Computed:    true,
						},
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
						"regions": {
							Description: "List of cloud provider supported regions",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
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
				},
			},
		},
	}
}

func dataSourceDatabasesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	params := &astra.ListDatabasesParams{
		Include:       nil,
		Provider:      nil,
		StartingAfter: nil,
		Limit:         nil,
	}

	if v, ok := d.GetOk("status"); ok {
		statusParam := astra.ListDatabasesParamsInclude(v.(string))
		params.Include = &statusParam
	}
	if v, ok := d.GetOk("cloud_provider"); ok {
		providerParam := astra.ListDatabasesParamsProvider(v.(string))
		params.Provider = &providerParam
	}

	resp, err := client.ListDatabasesWithResponse(ctx, params)
	if err != nil {
		return diag.FromErr(err)
	} else if resp.StatusCode() != 200 {
		return diag.Errorf("unexpected list databases response: %s", string(resp.Body))
	}

	dbs := astra.DatabaseSlice(resp.JSON200)
	flatDbs := make([]map[string]interface{}, 0, len(dbs))
	for _, db := range dbs {
		flatDbs = append(flatDbs, flattenDatabase(&db))
	}

	d.SetId(resource.UniqueId())
	if err := d.Set("results", flatDbs); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
