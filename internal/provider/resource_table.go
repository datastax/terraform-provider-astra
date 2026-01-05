package provider

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	astrarestapi "github.com/datastax/astra-client-go/v2/astra-rest-api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
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

	tableParams := astrarestapi.CreateTableParams{
		XCassandraToken: token,
	}

	ifnotexists := true

	var columnDefinitions, err = makeColumDefinitionsFromResourceData(d)
	if err != nil {
		return diag.FromErr(err)
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

	var restClient *astrarestapi.ClientWithResponses
	if val, ok := stargateCache[databaseID]; ok {
		restClient = val
	} else {
		var err error
		restClient, err = newRestClient(databaseID, providerVersion, userAgent, region)
		if err != nil {
			return diag.FromErr(err)
		}
	}

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
			return retry.NonRetryableError(fmt.Errorf("unexpected response fetching database, status code: %d, message %s", res.StatusCode(), string(res.Body)))
		}

		// Success fetching database
		db := res.JSON200
		switch db.Status {
		case astra.ERROR, astra.TERMINATED, astra.TERMINATING:
			// If the database reached a terminal state it will never become active
			return retry.NonRetryableError(fmt.Errorf("database failed to reach active status: status=%s", db.Status))
		case astra.ACTIVE:
			resp, err := restClient.CreateTableWithResponse(ctx, keyspaceName, &tableParams, createJSON)
			if err != nil {
				return retry.NonRetryableError(fmt.Errorf("error adding table (not retrying) err: %s,  body: %s", err, resp.Body))
			} else if resp.StatusCode() == 409 {
				// DevOps API returns 409 for concurrent modifications, these need to be retried.
				return retry.RetryableError(fmt.Errorf("error adding table (retrying): %s", resp.Body))
			} else if resp.StatusCode() >= 400 {
				return retry.NonRetryableError(fmt.Errorf("error adding table (not retrying): %s", resp.Body))
			}
			if err := setTableResourceData(d, databaseID, region, keyspaceName, tableName, primaryKey, columnDefinitions); err != nil {
				return retry.NonRetryableError(fmt.Errorf("Error setting table data (not retrying) %s", err))
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

	id := d.Id()
	databaseID, region, keyspaceName, tableName, err := parseTableID(id)
	if err != nil {
		return diag.FromErr(err)
	}
	if region == "" {
		region = d.Get("region").(string)
	}
	if region == "" {
		return diag.Errorf("missing region for table %s/<region>/%s/%s", databaseID, keyspaceName, tableName)
	}

	stargateCache := meta.(astraClients).stargateClientCache

	var restClient *astrarestapi.ClientWithResponses
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

	raw := true
	params := astrarestapi.GetTableParams{
		Raw:             &raw,
		XCassandraToken: token,
	}
	resp, err := restClient.GetTableWithResponse(ctx, keyspaceName, tableName, &params)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error getting table (not retrying) err: %w", err))
	} else if resp.StatusCode() == 409 {
		// DevOps API returns 409 for concurrent modifications, these need to be retried.
		return diag.Errorf("error getting table (retrying): %s", string(resp.Body))
	} else if resp.StatusCode() >= 400 {
		//table not found
		d.SetId("")
		return nil
	}

	tableData := resp.JSON200
	if err := setTableResourceData(d, databaseID, region, keyspaceName, tableName, tableData.PrimaryKey, tableData.ColumnDefinitions); err != nil {
		return diag.FromErr(fmt.Errorf("Error setting keyspace data (not retrying) %s", err))
	}

	return nil
}

func resourceTableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	providerVersion := meta.(astraClients).providerVersion
	userAgent := meta.(astraClients).userAgent
	token := meta.(astraClients).token

	id := d.Id()
	databaseID, _, keyspaceName, tableName, err := parseTableID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	region := d.Get("region").(string)

	stargateCache := meta.(astraClients).stargateClientCache

	var restClient *astrarestapi.ClientWithResponses
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
	resp, err := restClient.DeleteTableWithResponse(ctx, keyspaceName, tableName, &params)
	if err != nil {
		b := "none"
		if resp != nil && resp.Body != nil {
			b = string(resp.Body)
		}
		return diag.FromErr(fmt.Errorf("Error getting table (not retrying) err: %s,  body: %s", err, b))
	} else if resp.StatusCode() == 409 {
		// DevOps API returns 409 for concurrent modifications, these need to be retried.
		b := "none"
		if resp.Body != nil {
			b = string(resp.Body)
		}
		return diag.FromErr(fmt.Errorf("error getting table (retrying): %s", b))
	} else if resp.StatusCode() >= 400 {
		//table not found
		d.SetId("")
		return nil
	}
	d.SetId("")
	return nil
}

func makeColumDefinitionsFromResourceData(d *schema.ResourceData) ([]astrarestapi.ColumnDefinition, error) {
	columnDefsRaw := d.Get("column_definitions").([]interface{})

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
				return nil, fmt.Errorf("bad column definition. Key =%s is not one of [Name, Static, TypeDefinition]", key)
			}
		}
		columnDefinitions[i].Name = name
		columnDefinitions[i].Static = &static
		columnDefinitions[i].TypeDefinition = typeDef
	}
	return columnDefinitions, nil
}

func columnDefinitionsMatch(d *schema.ResourceData, columnDefinitions []astrarestapi.ColumnDefinition) bool {
	// get any existing definitions
	existingDefs := d.Get("column_definitions").([]interface{})
	if len(existingDefs) != len(columnDefinitions) {
		return false
	}
	// map of existing definitions by name
	var existingColumnDefinitions, err = makeColumDefinitionsFromResourceData(d)
	if err != nil {
		return false
	}
	existingDefsMap := make(map[string]astrarestapi.ColumnDefinition, len(existingDefs))
	for _, def := range existingColumnDefinitions {
		existingDefsMap[def.Name] = def
	}
	// check if the column definitions we want to set match the existing ones
	for _, def := range columnDefinitions {
		existingDef, ok := existingDefsMap[def.Name]
		if !ok {
			return false
		}
		if string(existingDef.TypeDefinition) != string(def.TypeDefinition) {
			return false
		}
		if strconv.FormatBool(*existingDef.Static) != strconv.FormatBool(*def.Static) {
			return false
		}
	}
	return true
}

func setColumnDefinitions(d *schema.ResourceData, columnDefinitions []astrarestapi.ColumnDefinition) error {
	// column definitions we will eventually store on the Resource
	cdefs := make([]map[string]string, len(columnDefinitions))

	// get any existing definitions
	existingDefs := d.Get("column_definitions").([]interface{})

	// if we have existing definitions, and they are the same as the ones we are about to set, we need
	// to preserve the order of the existing definitions to avoid it being detected as a change.
	if columnDefinitionsMatch(d, columnDefinitions) {
		apiDefsByName := make(map[string]astrarestapi.ColumnDefinition)
		for _, cdef := range columnDefinitions {
			apiDefsByName[cdef.Name] = cdef
		}
		// Preserve the order from existing state
		for index, existingDef := range existingDefs {
			defMap := existingDef.(map[string]interface{})
			name := defMap["Name"].(string)

			// Get the current data from API for this column
			if cdef, exists := apiDefsByName[name]; exists {
				cdefs[index] = map[string]string{
					"Name":           cdef.Name,
					"TypeDefinition": string(cdef.TypeDefinition),
					"Static":         strconv.FormatBool(*cdef.Static),
				}
			}
		}
	} else {
		// the column definitions we want do not match exactly the existing ones
		// sort them so they are easily comparable
		sortedDefs := make([]astrarestapi.ColumnDefinition, len(columnDefinitions))
		copy(sortedDefs, columnDefinitions)
		sort.Slice(sortedDefs, func(i, j int) bool {
			return sortedDefs[i].Name < sortedDefs[j].Name
		})

		for index, cdef := range sortedDefs {
			cdefs[index] = map[string]string{
				"Name":           cdef.Name,
				"TypeDefinition": string(cdef.TypeDefinition),
				"Static":         strconv.FormatBool(*cdef.Static),
			}
		}
	}

	// set the resulting column definitions on the Resource
	if err := d.Set("column_definitions", cdefs); err != nil {
		return err
	}
	return nil
}

func setTableResourceData(d *schema.ResourceData, databaseID, region, keyspaceName, table string, primaryKey astrarestapi.PrimaryKey, columnDefinitions []astrarestapi.ColumnDefinition) error {
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
	if err := d.Set("region", region); err != nil {
		return err
	}
	if primaryKey.PartitionKey == nil || len(primaryKey.PartitionKey) == 0 {
		return errors.New("primary key partition key is missing")
	}
	if err := d.Set("partition_keys", strings.Join(primaryKey.PartitionKey, ":")); err != nil {
		return err
	}
	if primaryKey.ClusteringKey == nil || len(*primaryKey.ClusteringKey) == 0 {
		return errors.New("primary key clustering key is missing")
	}
	if err := d.Set("clustering_columns", strings.Join(*primaryKey.ClusteringKey, ":")); err != nil {
		return err
	}
	// handle column_definitions
	if err := setColumnDefinitions(d, columnDefinitions); err != nil {
		return err
	}
	return nil
}

// parseTableID returns the databaseID, region, keyspace, tablename, error (if the format is invalid).
func parseTableID(id string) (string, string, string, string, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) == 3 {
		return idParts[0], "", idParts[1], idParts[2], nil
	} else if len(idParts) == 4 {
		return idParts[0], idParts[1], idParts[2], idParts[3], nil
	}
	return "", "", "", "", errors.New("invalid keyspace id format: expected database_id/keyspace/table")
}
