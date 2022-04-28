package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var availableCloudProviders = []string{
	"aws",
	"gcp",
	"azure",
}

var databaseCreateTimeout = time.Minute * 20
var databaseReadTimeout = time.Minute * 5

func resourceDatabase() *schema.Resource {
	return &schema.Resource{
		Description: "`astra_database` provides an Astra Serverless Database resource. You can create and delete databases. Note: Classic Tier databases are not supported by the Terraform provider. (see https://docs.datastax.com/en/astra/docs/index.html for more about Astra DB)",
		CreateContext: resourceDatabaseCreate,
		ReadContext:   resourceDatabaseRead,
		DeleteContext: resourceDatabaseDelete,
		UpdateContext: resourceDatabaseUpdate,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: &databaseCreateTimeout,
			Read:   &databaseReadTimeout,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"name": {
				Description:  "Astra database name.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"keyspace": {
				Description:      "Initial keyspace name. For additional keyspaces, use the astra_keyspace resource.",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validateKeyspace,
			},
			"cloud_provider": {
				Description:      "The cloud provider to launch the database. (Currently supported: aws, azure, gcp)",
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validation.StringInSlice(availableCloudProviders, true),
				DiffSuppressFunc: ignoreCase,
			},
			"regions": {
				Description: "Cloud regions to launch the database. (see https://docs.datastax.com/en/astra/docs/database-regions.html for supported regions)",
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			// Computed
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

func resourceDatabaseCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	name := resourceData.Get("name").(string)
	keyspace := resourceData.Get("keyspace").(string)
	cloudProvider := resourceData.Get("cloud_provider").(string)
	regions := resourceData.Get("regions").([]interface{})

	if len(regions) < 1 {
		return diag.Errorf("\"region\" array must have at least 1 region specified")
	}

	// Make sure all regions are valid
	if err := ensureValidRegions(ctx, client, resourceData); err != nil {
		return err
	}
	// get the first region in the list to use as the region in which to create the database
	region := regions[0].(string)

	// make an array of additonal regions to add if more than one specified
	additionalRegions := make([]string, len(regions) -1)
	if len(additionalRegions) > 0 {
		for i:=0; i<len(additionalRegions); i++ {
			additionalRegions[i] = regions[i+1].(string)
		}
	}

	resp, err := client.CreateDatabaseWithResponse(ctx, astra.CreateDatabaseJSONRequestBody{
		Name:          name,
		Keyspace:      keyspace,
		CloudProvider: astra.CloudProvider(cloudProvider),
		CapacityUnits: 1,
		Region:        region,
		Tier:          astra.Tier("serverless"),
	})
	if err != nil {
		return diag.FromErr(err)
	}
	if resp.StatusCode() != http.StatusCreated {
		return diag.Errorf("unexpected create database response: %s", string(resp.Body))
	}

	databaseID := resp.HTTPResponse.Header.Get("location")

	// Wait for the database to be ACTIVE then set resource data
	if err := waitForDatabaseAndUpdateResource(ctx, resourceData, client, databaseID); err != nil {
		return err
	}

	// Add any additional regions/datacenters
	if len(additionalRegions) > 0 {
		if err:= addRegionsToDatabase(ctx, resourceData, client, additionalRegions, databaseID, cloudProvider); err != nil {
			return err
		}
	}

	return nil
}

func resourceDatabaseRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID := resourceData.Id()

	if err := resource.RetryContext(ctx, resourceData.Timeout(schema.TimeoutRead), func() *resource.RetryError {
		resp, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
		if err != nil {
			return resource.RetryableError(fmt.Errorf("unable to fetch database (%s): %v", databaseID, err))
		}

		// Remove from state when database not found
		if resp.JSON404 != nil || resp.StatusCode() == http.StatusNotFound {
			resourceData.SetId("")
			return nil
		}

		// Retry on 5XX errors
		if resp.StatusCode() >= http.StatusInternalServerError {
			return resource.RetryableError(fmt.Errorf("error fetching database (%s): %v", databaseID, err))
		}

		// Don't retry for non 200 status code
		db := resp.JSON200
		if db == nil {
			return resource.NonRetryableError(fmt.Errorf("unexpected response fetching database (%s): %s", databaseID, string(resp.Body)))
		}

		// If the database is TERMINATING or TERMINATED then remove it from the state
		if db.Status == astra.StatusEnumTERMINATING || db.Status == astra.StatusEnumTERMINATED {
			resourceData.SetId("")
			return nil
		}

		// Add the database to state
		if err := setDatabaseResourceData(resourceData, db); err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDatabaseDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	databaseID := resourceData.Id()
	alreadyDeleted := false

	if err := resource.RetryContext(ctx, resourceData.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		resp, err := client.TerminateDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID), &astra.TerminateDatabaseParams{})
		if err != nil {
			return resource.RetryableError(err)
		}

		// Status code 5XX are considered transient
		if resp.StatusCode() >= http.StatusInternalServerError {
			return resource.RetryableError(fmt.Errorf("error terminating database: %s", string(resp.Body)))
		}

		// If the database cannot be found then it has been deleted
		if resp.StatusCode() == http.StatusNotFound {
			alreadyDeleted = true
			return nil
		}

		// All other 4XX status codes are NOT retried
		if resp.StatusCode() >= http.StatusBadRequest {
			return resource.NonRetryableError(fmt.Errorf("unexpected response attempting to terminate database: %s", string(resp.Body)))
		}

		return nil
	}); err != nil {
		return diag.FromErr(err)
	}

	// Return early since it has been determined that the database no longer exists
	if alreadyDeleted {
		resourceData.SetId("")
		return nil
	}

	// Wait for the database to be TERMINATED or not found
	if err := resource.RetryContext(ctx, resourceData.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		res, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
		// Errors sending request should be retried and are assumed to be transient
		if err != nil {
			return resource.RetryableError(err)
		}

		// Status code >=5xx are assumed to be transient
		if res.StatusCode() >= http.StatusInternalServerError {
			return resource.RetryableError(fmt.Errorf("error while fetching database: %s", string(res.Body)))
		}

		// If the database cannot be found. It has been deleted.
		if res.StatusCode() == http.StatusNotFound {
			return nil
		}

		// All other status codes > 200 NOT retried
		if res.StatusCode() > http.StatusOK || res.JSON200 == nil {
			return resource.NonRetryableError(fmt.Errorf("unexpected response fetching database: %s", string(res.Body)))
		}

		// Return when the database is in a TERMINATED state
		db := res.JSON200
		if db.Status == astra.StatusEnumTERMINATED {
			return nil
		}

		// Continue until one of the expected conditions above are met
		return resource.RetryableError(fmt.Errorf("expected database to be terminated but is %s", db.Status))
	}); err != nil {
		return diag.FromErr(err)
	}

	resourceData.SetId("")
	return nil
}

func resourceDatabaseUpdate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	databaseID := resourceData.Id()
	cloudProvider := resourceData.Get("cloud_provider").(string)

	if resourceData.HasChangeExcept("regions") {
		// only region changes supported at the moment
		return diag.Errorf("Update of Database resource only supported for fields: %s", "regions")
	}

	if resourceData.HasChange("regions") {
		// get regions to add and delete
		regionsToAdd, regionsToDelete := getRegionUpdates(resourceData.GetChange("regions"))
		if len(regionsToAdd) > 0 {
			// add any regions to add first
			if err := addRegionsToDatabase(ctx, resourceData, client, regionsToAdd, databaseID, cloudProvider); err != nil {
				return err
			}
		}
		if len(regionsToDelete) > 0 {
			// delete any regions that should be removed
			if err := deleteRegionsFromDatabase(ctx, resourceData, client, regionsToDelete, databaseID, cloudProvider); err != nil {
				return err
			}
		}
	}
	return nil
}

func getRegionUpdates(oldRegions interface{}, newRegions interface{}) ([]string, []string){
	mOld := map[string]bool{}
	mNew := map[string]bool{}
	var regionsToAdd []string
	var regionsToDelete []string
	// find any regions to add
	for _, v := range oldRegions.([]interface{}) {
		mOld[v.(string)] = true
	}
	for _, v := range newRegions.([]interface{}) {
		mNew[v.(string)] = true
	}
	for _, v := range oldRegions.([]interface{}) {
		if !mNew[v.(string)] {
			regionsToDelete = append(regionsToDelete, v.(string))
		}
	}
	for _, v := range newRegions.([]interface{}) {
		if !mOld[v.(string)] {
			regionsToAdd = append(regionsToAdd, v.(string))
		}
	}

	return regionsToAdd, regionsToDelete
}

func addRegionsToDatabase(ctx context.Context, resourceData *schema.ResourceData, client *astra.ClientWithResponses, regions []string, databaseID string, cloudProvider string) diag.Diagnostics {
	// make sure the regions are valid
	if err := ensureValidRegions(ctx, client, resourceData); err != nil {
		return err
	}
	// Currently, DevOps API only allows for adding 1 region at a time
	for _, region := range regions {
		datacenters := make([]astra.Datacenter, 1)
		datacenters[0] = astra.Datacenter {
			CloudProvider: astra.CloudProvider(cloudProvider),
			Region:        region,
			Tier:          "serverless",
		}
		resp, err := client.AddDatacentersWithResponse(ctx, astra.DatabaseIdParam(databaseID), datacenters)
		if err != nil {
			return diag.FromErr(err)
		}
		if resp.StatusCode() != http.StatusCreated {
			return diag.FromErr(fmt.Errorf("Unexpected response addinng Regions: %s", string(resp.Body)))
		}
		// Wait for the database to be ACTIVE then set resource data
		if err := waitForDatabaseAndUpdateResource(ctx, resourceData, client, databaseID); err != nil {
			return err
		}
	}
	return nil
}

func deleteRegionsFromDatabase(ctx context.Context, resourceData *schema.ResourceData, client *astra.ClientWithResponses, regions []string, databaseID string, cloudProvider string) diag.Diagnostics {
	// get all the datacenetrs for the Datbase ID
	dcListResp, err := client.ListDatacentersWithResponse(ctx, astra.DatabaseIdParam(databaseID), &astra.ListDatacentersParams{})
	if err != nil {
		return diag.FromErr(err)
	}
	if dcListResp.StatusCode() != http.StatusOK || dcListResp.JSON200 == nil {
		return diag.FromErr(fmt.Errorf("Unexpected response fetching Datacenters: %s", dcListResp.Body))
	}
	dcs := *dcListResp.JSON200
	// map regions to DCs
	regionDcMap := map[string]astra.Datacenter{}
	for _, v := range dcs {
		regionDcMap[v.Region] = v
	}
	// delete each region that exists
	for _, v := range regions {
		if dc := regionDcMap[v]; dc.Id != nil {
			termResp, err := client.TerminateDatacenterWithResponse(ctx, astra.DatabaseIdParam(databaseID), astra.DatacenterIdParam(*dc.Id))
			if err != nil {
				return diag.FromErr(err)
			}
			if termResp.StatusCode() != http.StatusAccepted {
				return diag.FromErr(fmt.Errorf("Error terminating datacenter for region \"%s\": %s", v, string(termResp.Body)))
			}
			// Wait for the database to be ACTIVE then set resource data
			if err := waitForDatabaseAndUpdateResource(ctx, resourceData, client, databaseID); err != nil {
				return err
			}
		}
	}
	return nil
}

func waitForDatabaseAndUpdateResource(ctx context.Context, resourceData *schema.ResourceData, client *astra.ClientWithResponses, databaseID string) diag.Diagnostics {
	if err := resource.RetryContext(ctx, resourceData.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		res, err := client.GetDatabaseWithResponse(ctx, astra.DatabaseIdParam(databaseID))
		// Errors sending request should be retried and are assumed to be transient
		if err != nil {
			return resource.RetryableError(err)
		}

		// Status code >=5xx are assumed to be transient
		if res.StatusCode() >= http.StatusInternalServerError {
			return resource.RetryableError(fmt.Errorf("error while fetching database: %s", string(res.Body)))
		}

		// Status code > 200 NOT retried
		if res.StatusCode() > http.StatusOK || res.JSON200 == nil {
			return resource.NonRetryableError(fmt.Errorf("unexpected response fetching database: %s", string(res.Body)))
		}

		// Success fetching database
		db := res.JSON200
		switch db.Status {
		case astra.StatusEnumERROR, astra.StatusEnumTERMINATED, astra.StatusEnumTERMINATING:
			// If the database reached a terminal state it will never become active
			return resource.NonRetryableError(fmt.Errorf("database failed to reach active status: status=%s", db.Status))
		case astra.StatusEnumACTIVE:
			if err := setDatabaseResourceData(resourceData, db); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		default:
			return resource.RetryableError(fmt.Errorf("expected database to be active but is %s", db.Status))
		}
	}); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func setDatabaseResourceData(resourceData *schema.ResourceData, db *astra.Database) error {
	resourceData.SetId(db.Id)
	flatDb := flattenDatabase(db)
	for k, v := range flatDb {
		if k == "id" {
			continue
		}
		if err := resourceData.Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func flattenDatabase(db *astra.Database) map[string]interface{} {
	flatDB := map[string]interface{}{
		"id":                   db.Id,
		"name":                 astra.StringValue(db.Info.Name),
		"organization_id":      db.OrgId,
		"owner_id":             db.OwnerId,
		"status":               string(db.Status),
		"grafana_url":          astra.StringValue(db.GrafanaUrl),
		"graphql_url":          astra.StringValue(db.GraphqlUrl),
		"data_endpoint_url":    astra.StringValue(db.DataEndpointUrl),
		"cqlsh_url":            astra.StringValue(db.CqlshUrl),
		"cloud_provider":       "",
		"regions":              []string{astra.StringValue(db.Info.Region)},
		"keyspace":             astra.StringValue(db.Info.Keyspace),
		"additional_keyspaces": astra.StringSlice(db.Info.AdditionalKeyspaces),
		"node_count":           db.Storage.NodeCount,
		"replication_factor":   db.Storage.ReplicationFactor,
		"total_storage":        db.Storage.TotalStorage,
	}

	if db.Info.CloudProvider != nil {
		cloudProvider := *db.Info.CloudProvider
		flatDB["cloud_provider"] = string(cloudProvider)
	}

	if db.Info.Datacenters != nil && len(*db.Info.Datacenters) > 1 {
		regions := make([]string, len(*db.Info.Datacenters))
		for index, dc := range *db.Info.Datacenters {
			regions[index] = dc.Region
		}
		flatDB["regions"] = regions
	}
	return flatDB
}

func ensureValidRegions(ctx context.Context, client *astra.ClientWithResponses, resourceData *schema.ResourceData) diag.Diagnostics {
	// get the list of serveless regions
	regionsResp, err := client.ListServerlessRegionsWithResponse(ctx)
	if err != nil {
		return diag.FromErr(err)
	} else if regionsResp.StatusCode() != http.StatusOK {
		return diag.Errorf("unexpected list available regions response: %s", string(regionsResp.Body))
	}
	// make sure all of the regions are valid
	cloudProvider := resourceData.Get("cloud_provider").(string)
	regions := resourceData.Get("regions").([] interface{})
	for _, r := range regions {
		region := r.(string)
		dbRegion := findMatchingRegion(cloudProvider, region, "serverless", *regionsResp.JSON200)
		if dbRegion == nil {
			return diag.Errorf("cloud provider and region combination not available: %s/%s", cloudProvider, region)
		}
	}
	return nil
}

func findMatchingRegion(provider, region, tier string, availableRegions []astra.ServerlessRegion) *astra.ServerlessRegion {
	for _, ar := range availableRegions {
		if strings.EqualFold(string(ar.CloudProvider), provider) &&
			strings.EqualFold(ar.Name, region) {
			return &ar
		}
	}

	return nil
}
