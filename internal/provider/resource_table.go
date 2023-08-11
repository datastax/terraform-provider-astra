package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	astrarestapi "github.com/datastax/astra-client-go/v2/astra-rest-api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceTable() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_table` provides a table resource which represents a table in cassandra.",
		CreateContext: resourceTableCreate,
		ReadContext:   resourceTableRead,
		DeleteContext: resourceTableDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"keyspace": {
				Description:      "Keyspace name can have up to 48 alpha-numeric characters and contain underscores; only letters are supported as the first character.",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateKeyspace,
			},
			"table": {
				Description:      "Table name can have up to 48 alpha-numeric characters and contain underscores; only letters are supported as the first character.",
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
			"region": {
				Description: "region.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"clustering_columns": {
				Description: "Clustering column(s), separated by :",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"partition_keys": {
				Description: "Partition key(s), separated by :",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"column_definitions": {
				Description: "A list of table Definitions",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				Elem: &schema.Schema{
					Type: schema.TypeMap,
					Elem: &schema.Schema{
						Type: schema.TypeString,
					},
				},
			},
		},
	}
}

func resourceTableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	providerVersion := meta.(astraClients).providerVersion
	userAgent := meta.(astraClients).userAgent
	token := meta.(astraClients).token

	stargateCache := meta.(astraClients).stargateClientCache

	databaseID := d.Get("database_id").(string)
	keyspaceName := d.Get("keyspace").(string)
	tableName := d.Get("table").(string)
	region := d.Get("region").(string)
	partitionKeys := strings.Split(d.Get("partition_keys").(string), ":")
	clusteringColumns := strings.Split(d.Get("clustering_columns").(string), ":")
	columnDefsRaw := d.Get("column_definitions").([]interface{})

	tableParams := astrarestapi.CreateTableParams{
		XCassandraToken: token,
	}

	ifnotexists := true

	var columnDefinitions = make([]astrarestapi.ColumnDefinition, len(columnDefsRaw))
	for i := 0; i < len(columnDefsRaw); i++ {
		defMap := columnDefsRaw[i].(map[string]interface{})
		var name string
		var static bool
		var typeDef astrarestapi.ColumnDefinitionTypeDefinition
		for key, value := range defMap {
			switch key {
			case "Name":
				name = value.(string)
			case "Static":
				static, _ = strconv.ParseBool(value.(string))
			case "TypeDefinition":
				typeDef = astrarestapi.ColumnDefinitionTypeDefinition(value.(string))
			default:
				return diag.Errorf("bad column definition")
			}
		}
		columnDefinitions[i].Name = name
		columnDefinitions[i].Static = &static
		columnDefinitions[i].TypeDefinition = typeDef
	}

	primaryKey := astrarestapi.PrimaryKey{
		ClusteringKey: &clusteringColumns,
		PartitionKey:  partitionKeys,
	}

	createJSON := astrarestapi.CreateTableJSONRequestBody{
		ColumnDefinitions: columnDefinitions,
		IfNotExists:       &ifnotexists,
		Name:              tableName,
		PrimaryKey:        primaryKey,
		TableOptions:      nil,
	}

	var restClient astrarestapi.Client
	if val, ok := stargateCache[databaseID]; ok {
		restClient = val
	} else {
		var err error
		restClient, err = newRestClient(databaseID, providerVersion, userAgent, region)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	fmt.Printf("%v", restClient)

	//Wait for DB to be in Active status
	if err := retry.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *retry.RetryError {
		res, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
		// Errors sending request should be retried and are assumed to be transient
		if err != nil {
			return retry.RetryableError(err)
		}

		// Status code >=5xx are assumed to be transient
		if res.StatusCode() >= 500 {
			return retry.RetryableError(fmt.Errorf("error while fetching database: %s", string(res.Body)))
		}

		// Status code > 200 NOT retried
		if res.StatusCode() > 200 || res.JSON200 == nil {
			return retry.NonRetryableError(fmt.Errorf("unexpected response fetching database: %s", string(res.Body)))
		}

		// Success fetching database
		db := res.JSON200
		switch db.Status {
		case astra.ERROR, astra.TERMINATED, astra.TERMINATING:
			// If the database reached a terminal state it will never become active
			return retry.NonRetryableError(fmt.Errorf("database failed to reach active status: status=%s", db.Status))
		case astra.ACTIVE:
			resp, err := restClient.CreateTable(ctx, keyspaceName, &tableParams, createJSON)
			if err != nil {
				b := []byte{}
				if resp != nil {
					b, _ = io.ReadAll(resp.Body)
				}
				return retry.NonRetryableError(fmt.Errorf("Error adding table (not retrying) err: %s,  body: %s", err, b))
			} else if resp.StatusCode == 409 {
				// DevOps API returns 409 for concurrent modifications, these need to be retried.
				b, _ := io.ReadAll(resp.Body)
				return retry.RetryableError(fmt.Errorf("error adding table (retrying): %s", b))
			} else if resp.StatusCode >= 400 {
				b, _ := io.ReadAll(resp.Body)
				return retry.NonRetryableError(fmt.Errorf("error adding table (not retrying): %s", b))
			}

			if err := setTableResourceData(d, databaseID, keyspaceName, tableName); err != nil {
				return retry.NonRetryableError(fmt.Errorf("Error setting keyspace data (not retrying) %s", err))
			}

			return nil
		default:
			return retry.RetryableError(fmt.Errorf("expected database to be active but is %s", db.Status))
		}
	}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceTableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerVersion := meta.(astraClients).providerVersion
	userAgent := meta.(astraClients).userAgent
	token := meta.(astraClients).token

	tableName := d.Get("table").(string)
	region := d.Get("region").(string)

	id := d.Id()
	databaseID, keyspaceName, tableName, err := parseTableID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	stargateCache := meta.(astraClients).stargateClientCache

	var restClient astrarestapi.Client
	if val, ok := stargateCache[databaseID]; ok {
		restClient = val
	} else {
		var err error
		restClient, err = newRestClient(databaseID, providerVersion, userAgent, region)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	fmt.Printf("%v", restClient)

	params := astrarestapi.GetTableParams{
		Raw:             nil,
		XCassandraToken: token,
	}
	resp, err := restClient.GetTable(ctx, keyspaceName, tableName, &params)
	if err != nil {
		b, _ := io.ReadAll(resp.Body)
		return diag.FromErr(fmt.Errorf("Error getting table (not retrying) err: %s,  body: %s", err, b))
	} else if resp.StatusCode == 409 {
		// DevOps API returns 409 for concurrent modifications, these need to be retried.
		b, _ := io.ReadAll(resp.Body)
		return diag.FromErr(fmt.Errorf("error getting table (retrying): %s", b))
	} else if resp.StatusCode >= 400 {
		//table not found
		d.SetId("")
		return nil
	}

	if err := setTableResourceData(d, databaseID, keyspaceName, tableName); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting keyspace data (not retrying) %s", err))
	}

	return nil
}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerVersion := meta.(astraClients).providerVersion
	userAgent := meta.(astraClients).userAgent
	token := meta.(astraClients).token

	tableName := d.Get("table").(string)
	region := d.Get("region").(string)

	id := d.Id()
	databaseID, keyspaceName, tableName, err := parseTableID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	stargateCache := meta.(astraClients).stargateClientCache

	var restClient astrarestapi.Client
	if val, ok := stargateCache[databaseID]; ok {
		restClient = val
	} else {
		var err error
		restClient, err = newRestClient(databaseID, providerVersion, userAgent, region)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	fmt.Printf("%v", restClient)

	params := astrarestapi.DeleteTableParams{
		XCassandraToken: token,
	}
	resp, err := restClient.DeleteTable(ctx, keyspaceName, tableName, &params)
	if err != nil {
		b, _ := io.ReadAll(resp.Body)
		return diag.FromErr(fmt.Errorf("Error getting table (not retrying) err: %s,  body: %s", err, b))
	} else if resp.StatusCode == 409 {
		// DevOps API returns 409 for concurrent modifications, these need to be retried.
		b, _ := io.ReadAll(resp.Body)
		return diag.FromErr(fmt.Errorf("error getting table (retrying): %s", b))
	} else if resp.StatusCode >= 400 {
		//table not found
		d.SetId("")
		return nil
	}
	d.SetId("")
	return nil
}

func setTableResourceData(d *schema.ResourceData, databaseID string, keyspaceName string, table string) error {
	d.SetId(fmt.Sprintf("%s/%s/%s", databaseID, keyspaceName, table))
	if err := d.Set("table", table); err != nil {
		return err
	}
	if err := d.Set("keyspace", keyspaceName); err != nil {
		return err
	}
	if err := d.Set("database_id", databaseID); err != nil {
		return err
	}

	return nil
}

func parseTableID(id string) (string, string, string, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) != 3 {
		return "", "", "", errors.New("invalid keyspace id format: expected database_id/keyspace/table")
	}
	return idParts[0], idParts[1], idParts[2], nil
}
