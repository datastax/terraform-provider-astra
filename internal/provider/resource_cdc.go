package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceCDC() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_cdc` enables cdc for an Astra Serverless table.",
		CreateContext: resourceCDCCreate,
		ReadContext:   resourceCDCRead,
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
				Description: "Astra database name.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"topic_partitions": {
				Description: "Number of partitions in cdc topic.",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"tenant_name": {
				Description: "Streaming tenant name",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"connector_status": {
				Description: "Connector Status",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"data_topic": {
				Description: "Data topic name",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

func resourceCDCDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	token := meta.(astraClients).token

	id := resourceData.Id()

	databaseId, keyspace, table, tenantName, err := parseCDCID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	orgResp, err := client.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to request current organization: %w", err))
	}

	var org OrgId

	err = json.NewDecoder(orgResp.Body).Decode(&org)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read current organization: %w", err))
	}

	pulsarCluster, pulsarToken, err := prepCDC(ctx, client, databaseId, token, org, streamingClient, tenantName)
	if err != nil {
		diag.FromErr(err)
	}

	deleteCDCParams := astrastreaming.DeleteCDCParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          pulsarToken,
	}

	deleteRequestBody := astrastreaming.DeleteCDCJSONRequestBody{
		DatabaseId:      databaseId,
		DatabaseName:    resourceData.Get("database_name").(string),
		Keyspace:        keyspace,
		OrgId:           org.ID,
		TableName:       table,
		TopicPartitions: resourceData.Get("topic_partitions").(int),
	}
	getDeleteCDCResponse, err := streamingClientv3.DeleteCDC(ctx, tenantName, &deleteCDCParams, deleteRequestBody)

	if err != nil {
		diag.FromErr(err)
	}
	if getDeleteCDCResponse.StatusCode > 299 {
		body, _ := io.ReadAll(getDeleteCDCResponse.Body)
		return diag.Errorf("Error deleting cdc %s", body)
	}

	// Deleted. Remove from state.
	resourceData.SetId("")

	return nil

}

type CDCStatusResponse []CDCStatus

type CDCStatus struct {
	OrgID           string    `json:"orgId"`
	ClusterName     string    `json:"clusterName"`
	Tenant          string    `json:"tenant"`
	Namespace       string    `json:"namespace"`
	ConnectorName   string    `json:"connectorName"`
	ConfigType      string    `json:"configType"`
	DatabaseID      string    `json:"databaseId"`
	DatabaseName    string    `json:"databaseName"`
	Keyspace        string    `json:"keyspace"`
	DatabaseTable   string    `json:"databaseTable"`
	ConnectorStatus string    `json:"connectorStatus"`
	CdcStatus       string    `json:"cdcStatus"`
	CodStatus       string    `json:"codStatus"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	EventTopic      string    `json:"eventTopic"`
	DataTopic       string    `json:"dataTopic"`
	Instances       int       `json:"instances"`
	CPU             int       `json:"cpu"`
	Memory          int       `json:"memory"`
}

func resourceCDCRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	token := meta.(astraClients).token

	id := resourceData.Id()

	databaseId, keyspace, table, tenantName, err := parseCDCID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	orgResp, err := client.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to request organization: %w", err))
	}

	var org OrgId

	err = json.NewDecoder(orgResp.Body).Decode(&org)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read organization: %w", err))
	}

	pulsarCluster, pulsarToken, err := prepCDC(ctx, client, databaseId, token, org, streamingClient, tenantName)
	if err != nil {
		diag.FromErr(err)
	}

	getCDCParams := astrastreaming.GetCDCParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          pulsarToken,
	}
	getCDCResponse, err := streamingClientv3.GetCDC(ctx, tenantName, &getCDCParams)
	if err != nil {
		diag.FromErr(fmt.Errorf("failed get CDC request: %w", err))
	} else if getCDCResponse.StatusCode > 299 {
		body, _ := io.ReadAll(getCDCResponse.Body)
		return diag.Errorf("Error getting cdc config %s", body)
	}

	var cdcResult CDCStatusResponse
	err = json.NewDecoder(getCDCResponse.Body).Decode(&cdcResult)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read CDC status: %w", err))
	}

	for i := 0; i < len(cdcResult); i++ {
		if cdcResult[i].Keyspace == keyspace {
			if cdcResult[i].DatabaseTable == table {
				return nil
			}
		}
	}

	if err := resourceData.Set("connector_status", cdcResult[0].ConnectorStatus); err != nil {
		return diag.FromErr(err)
	}
	if err := resourceData.Set("data_topic", cdcResult[0].DataTopic); err != nil {
		return diag.FromErr(err)
	}

	// Not found. Remove from state.
	resourceData.SetId("")

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

type StreamingTokens []struct {
	Iat     int    `json:"iat"`
	Iss     string `json:"iss"`
	Sub     string `json:"sub"`
	Tokenid string `json:"tokenid"`
}

// cdcEnablementMutex forces only a one CDC enablement at a time to prevent most concurrency issues
var cdcEnablementMutex sync.Mutex

func resourceCDCCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	token := meta.(astraClients).token

	table := resourceData.Get("table").(string)
	keyspace := resourceData.Get("keyspace").(string)
	databaseId := resourceData.Get("database_id").(string)
	databaseName := resourceData.Get("database_name").(string)
	topicPartitions := resourceData.Get("topic_partitions").(int)
	tenantName := resourceData.Get("tenant_name").(string)

	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	if err := json.NewDecoder(orgBody.Body).Decode(&org); err != nil {
		return diag.FromErr(fmt.Errorf("failed to ready organization: %w", err))
	}

	cdcRequestJSON := astrastreaming.EnableCDCJSONRequestBody{
		DatabaseId:      databaseId,
		DatabaseName:    databaseName,
		Keyspace:        keyspace,
		OrgId:           org.ID,
		TableName:       table,
		TopicPartitions: topicPartitions,
	}

	pulsarCluster, pulsarToken, err := prepCDC(ctx, client, databaseId, token, org, streamingClient, tenantName)
	if err != nil {
		return diag.FromErr(err)
	}

	enableCDCParams := astrastreaming.EnableCDCParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}
	getCDCParams := astrastreaming.GetCDCParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	const maxRetries = 1
	cdcEnablementMutex.Lock()
	defer cdcEnablementMutex.Unlock()

	for i := 0; i <= maxRetries; i++ {
		if enableCDCResponse, err := streamingClientv3.EnableCDC(ctx, tenantName, &enableCDCParams, cdcRequestJSON); err != nil {
			return diag.FromErr(fmt.Errorf("failed to enable CDC: %w", err))
		} else if enableCDCResponse.StatusCode > 299 {
			bodyBuffer, _ := io.ReadAll(enableCDCResponse.Body)
			return diag.FromErr(fmt.Errorf("failed to enable CDC, status: %v, message: %s", enableCDCResponse.StatusCode, string(bodyBuffer)))
		}

		tflog.Info(ctx, fmt.Sprintf("waiting for CDC on keyspace: %s, table: %s", cdcRequestJSON.Keyspace, cdcRequestJSON.TableName))
		time.Sleep(time.Second * 3)

		if cdcStatus, err := waitCDCStatusReady(ctx, streamingClientv3, databaseId, keyspace, table, tenantName, getCDCParams); err != nil {
			return diag.FromErr(fmt.Errorf("failed to check CDC status %w", err))
		} else if cdcStatus != nil {
			if err := resourceData.Set("connector_status", cdcStatus.CodStatus); err != nil {
				return diag.FromErr(err)
			}
			if err := resourceData.Set("data_topic", cdcStatus.DataTopic); err != nil {
				return diag.FromErr(err)
			}
			setCDCData(resourceData, fmt.Sprintf("%s/%s/%s/%s", databaseId, keyspace, table, tenantName))
			return nil
		}

		tflog.Warn(ctx, fmt.Sprintf("CDC not ready after max wait time, remaining retries: %v", (maxRetries-i)))
	}

	return diag.FromErr(fmt.Errorf("failed to enable cdc with max retries for keyspace: %s, table: %s", keyspace, table))
}

// waitCDCStatusReady tries to wait until CDC becomes ready
func waitCDCStatusReady(ctx context.Context, client *astrastreaming.ClientWithResponses,
	databaseId, keyspace, table, streamingTenant string, params astrastreaming.GetCDCParams) (*CDCStatus, error) {
	const CDCStatusActive = "Active"
	const maxRetries = 10
	const statusCheckInterval = time.Second * 6
	for i := 0; i <= maxRetries; i++ {
		if getCDCResponse, err := client.GetCDC(ctx, streamingTenant, &params); err != nil {
			return nil, fmt.Errorf("failed to get CDC status request: %w", err)
		} else if getCDCResponse.StatusCode > 299 {
			bodyBuffer, _ := io.ReadAll(getCDCResponse.Body)
			tflog.Warn(ctx, fmt.Sprintf("failed to read CDC status, code: %v, message: %s", getCDCResponse.StatusCode, string(bodyBuffer)))
		} else {
			var cdcStatusResponse CDCStatusResponse
			if err = json.NewDecoder(getCDCResponse.Body).Decode(&cdcStatusResponse); err != nil {
				return nil, (fmt.Errorf("failed to read CDC response %w", err))
			}
			if status := getTableCDCStatus(databaseId, keyspace, table, cdcStatusResponse); status != nil && status.CodStatus == CDCStatusActive {
				return status, nil
			}
		}
		time.Sleep(statusCheckInterval)
	}
	return nil, nil
}

// getTableCDCStatus get the CDC status of a specific table
func getTableCDCStatus(databaseID, keyspace, table string, cdcStatuses CDCStatusResponse) *CDCStatus {
	for _, cdcStatus := range cdcStatuses {
		if (databaseID == cdcStatus.DatabaseID) && (keyspace == cdcStatus.Keyspace) && (table == cdcStatus.DatabaseTable) {
			return &cdcStatus
		}
	}
	return nil
}

func prepCDC(ctx context.Context, client *astra.ClientWithResponses, databaseId string, token string, org OrgId, streamingClient *astrastreaming.ClientWithResponses, tenantName string) (string, string, error) {
	databaseResourceData := schema.ResourceData{}
	db, err := getDatabase(ctx, &databaseResourceData, client, databaseId)
	if err != nil {
		return "", "", err
	}

	// In most astra APIs there are dashes in region names depending on the cloud provider, this seems not to be the case for streaming
	cloudProvider := string(*db.Info.CloudProvider)
	fmt.Printf("%s", cloudProvider)

	pulsarCluster := getPulsarCluster("", cloudProvider, *db.Info.Region, "")
	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, streamingClient, tenantName)
	return pulsarCluster, pulsarToken, err
}

func getPulsarToken(ctx context.Context, pulsarCluster string, token string, org OrgId, streamingClient *astrastreaming.ClientWithResponses, tenantName string) (string, error) {

	tenantTokenParams := astrastreaming.GetPulsarTokensByTenantParams{
		Authorization:          fmt.Sprintf("Bearer %s", token),
		XDataStaxCurrentOrg:    org.ID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	pulsarTokenResponse, err := streamingClient.GetPulsarTokensByTenantWithResponse(ctx, tenantName, &tenantTokenParams)
	if err != nil {
		return "", fmt.Errorf("failed to get pulsar token: %w", err)
	} else if pulsarTokenResponse.StatusCode() > 299 {
		return "", fmt.Errorf("failed to get pulsar token, status code: %d, message: %s", pulsarTokenResponse.StatusCode(), string(pulsarTokenResponse.Body))
	}

	var streamingTokens StreamingTokens
	err = json.Unmarshal(pulsarTokenResponse.Body, &streamingTokens)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	tokenId := streamingTokens[0].Tokenid
	getTokenByIdParams := astrastreaming.GetPulsarTokenByIDParams{
		Authorization:          fmt.Sprintf("Bearer %s", token),
		XDataStaxCurrentOrg:    org.ID,
		XDataStaxPulsarCluster: pulsarCluster,
	}

	getTokenResponse, err := streamingClient.GetPulsarTokenByIDWithResponse(ctx, tenantName, tokenId, &getTokenByIdParams)

	if err != nil {
		fmt.Println("Can't get token", err)
		return "", err
	}

	pulsarToken := string(getTokenResponse.Body)
	return pulsarToken, err
}

func setCDCData(d *schema.ResourceData, id string) error {
	d.SetId(id)

	return nil
}

func parseCDCID(id string) (string, string, string, string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 4 {
		return "", "", "", "", errors.New("invalid role id format: expected databaseId/keyspace/table/tenantName")
	}
	return idParts[0], idParts[1], idParts[2], idParts[3], nil
}
