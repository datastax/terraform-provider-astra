package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/datastax/astra-client-go/v2/astra"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"io/ioutil"
	"regexp"
	"strings"
)

func resourceCDC() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_cdc` enables cdc for an Astra Serverless table.",
		CreateContext: resourceCDCCreate,
		ReadContext: resourceCDCRead,
		DeleteContext: resourceCDCDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"table": {
				Description:  "Astra database table.",
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
			"database_id": {
				Description:  "Astra database to create the keyspace.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"database_name": {
				Description:  "Astra database name.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
			"topic_partitions": {
				Description:  "Number of partitions in cdc topic.",
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
			},
			"tenant_name": {
				Description:  "Streaming tenant name",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
		},
	}
}

func resourceCDCDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {

	// TODO

	return nil

}

func resourceCDCRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO
	return nil
}

type ServerlessStreamingAvailableRegionsResult []struct {
	Tier            string `json:"tier"`
	Description     string `json:"description"`
	CloudProvider   string `json:"cloudProvider"`
	Region          string `json:"region"`
	RegionDisplay   string `json:"regionDisplay"`
	RegionContinent string `json:"regionContinent"`
	Cost            struct {
		CostPerMinCents         int `json:"costPerMinCents"`
		CostPerHourCents        int `json:"costPerHourCents"`
		CostPerDayCents         int `json:"costPerDayCents"`
		CostPerMonthCents       int `json:"costPerMonthCents"`
		CostPerMinMRCents       int `json:"costPerMinMRCents"`
		CostPerHourMRCents      int `json:"costPerHourMRCents"`
		CostPerDayMRCents       int `json:"costPerDayMRCents"`
		CostPerMonthMRCents     int `json:"costPerMonthMRCents"`
		CostPerMinParkedCents   int `json:"costPerMinParkedCents"`
		CostPerHourParkedCents  int `json:"costPerHourParkedCents"`
		CostPerDayParkedCents   int `json:"costPerDayParkedCents"`
		CostPerMonthParkedCents int `json:"costPerMonthParkedCents"`
		CostPerNetworkGbCents   int `json:"costPerNetworkGbCents"`
		CostPerWrittenGbCents   int `json:"costPerWrittenGbCents"`
		CostPerReadGbCents      int `json:"costPerReadGbCents"`
	} `json:"cost"`
	DatabaseCountUsed               int `json:"databaseCountUsed"`
	DatabaseCountLimit              int `json:"databaseCountLimit"`
	CapacityUnitsUsed               int `json:"capacityUnitsUsed"`
	CapacityUnitsLimit              int `json:"capacityUnitsLimit"`
	DefaultStoragePerCapacityUnitGb int `json:"defaultStoragePerCapacityUnitGb"`
}

func resourceCDCCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	table := resourceData.Get("table").(string)
	keyspace := resourceData.Get("keyspace").(string)
	databaseId := resourceData.Get("database_id").(string)
	databaseName := resourceData.Get("database_name").(string)
	topicPartitions := resourceData.Get("topic_partitions").(int)
	tenantName := resourceData.Get("tenant_name").(string)

	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserialize", orgBody)
	}

	databaseResourceData := schema.ResourceData{}
	db, diagnostics, done := getDatabase(ctx, &databaseResourceData, client, databaseId)
	if done {
		return diagnostics
	}

	regions := db.Info.Region
	cloudProvider:= db.Info.CloudProvider
	fmt.Printf("%s", *regions)
	fmt.Printf("%s", *cloudProvider)


	response, err := streamingClient.ListAvailableRegionsWithResponse(ctx)
	if err != nil{
		return  diag.FromErr(err)
	}

	body := response.Body

	var result ServerlessStreamingAvailableRegionsResult
	err = json.Unmarshal(body, &result)
	if err != nil {
		fmt.Println("Can't deserialize", body)
	}

	// TODO: validate region
	if checkRegion(result, cloudProvider, regions){
		return setupCDC(ctx, databaseId, databaseName, keyspace, org, table, topicPartitions, err, streamingClient, tenantName, bodyBuffer)
	}
	return diag.Errorf("CDC not available for region: %s", *regions)



	/*
		var result ServerlessStreamingAvailableRegionsResult
		regionsBodyBuffer := new(bytes.Buffer)
		err = json.Unmarshal(regionsBodyBuffer.Bytes(), &result)
		if err != nil {
			fmt.Println("Can't deserislize", body)
		}
	*/

	// TODO: check that db is single region

	// Step 0
	/*
		streamingClustersResponse, _ := streamingClient.GetStreamingTenantWithResponse(ctx)

		b, err := io.ReadAll(streamingClustersResponse.Body)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("body %s", b)
	*/

	//streamingClient.ListTenant

	// HEAD to see if the tenant exists
	// Step 1: create tenant
	//streamingClient.IdOfCreateTenantEndpointWithBodyWithResponse()

	// Step 2: create cdc

	// Step 3: create sink https://pulsar.apache.org/sink-rest-api/?version=2.8.0&apiversion=v3#operation/registerSink

	return nil
}

func setupCDC(ctx context.Context, databaseId string, databaseName string, keyspace string, org OrgId, table string, topicPartitions int, err error, streamingClient *astrastreaming.ClientWithResponses, tenantName string, bodyBuffer []byte) (diag.Diagnostics) {
	cdcRequestJSON := astrastreaming.EnableCDCJSONRequestBody{
		DatabaseId:      databaseId,
		DatabaseName:    databaseName,
		Keyspace:        keyspace,
		OrgId:           org.ID,
		TableName:       table,
		TopicPartitions: topicPartitions,
	}

	enableClientResult, err := streamingClient.EnableCDC(ctx, tenantName, cdcRequestJSON)

	if err != nil {
		return diag.FromErr(err)
	}

    if !strings.HasPrefix(enableClientResult.Status, "2") {
		bodyBuffer, err = ioutil.ReadAll(enableClientResult.Body)
		return diag.Errorf("Error creating tenant %s", enableClientResult.Body)
	}
	bodyBuffer, err = ioutil.ReadAll(enableClientResult.Body)

	fmt.Printf("success enabling cdc: %s", bodyBuffer)

	return nil
}

func checkRegion(streamingRegions ServerlessStreamingAvailableRegionsResult, provider *astra.CloudProvider, regions *string) bool {
	for i:=0; i< len(streamingRegions); i++ {
		if *provider == astra.CloudProvider(streamingRegions[i].CloudProvider){
			if *regions == streamingRegions[i].Region{
				return true
			}
		}
	}
	return false
}


func setCDCData(d *schema.ResourceData, tenantName string) error {
	d.SetId(fmt.Sprintf("%s", tenantName))

	if err := d.Set("tenant_name", tenantName); err != nil {
		return err
	}

	return nil
}

func parseCDCID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  errors.New("invalid role id format: expected tenantID/")
	}
	return idParts[0],  nil
}

