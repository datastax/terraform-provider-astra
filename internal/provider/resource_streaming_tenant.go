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

func resourceStreamingTenant() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_cdc` enables cdc for an Astra Serverless table.",
		CreateContext: resourceStreamingTenantCreate,
		ReadContext:   resourceStreamingTenantRead,
		DeleteContext: resourceStreamingTenantDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			"tenant_name": {
				Description:  "Streaming tenant name.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"topic": {
				Description:  "Streaming tenant topic.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"region": {
				Description:  "cloud region",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"cloud_provider": {
				Description:  "Cloud provider",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"user_email": {
				Description:  "User email for tenant.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
		},
	}
}




type OrgId struct {
	ID string `json:"id"`
}

type StreamingClusters []struct {
	ID                     string `json:"id"`
	TenantName             string `json:"tenantName"`
	ClusterName            string `json:"clusterName"`
	WebServiceURL          string `json:"webServiceUrl"`
	BrokerServiceURL       string `json:"brokerServiceUrl"`
	WebsocketURL           string `json:"websocketUrl"`
	WebsocketQueryParamURL string `json:"websocketQueryParamUrl"`
	PulsarToken            string `json:"pulsarToken"`
	Plan                   string `json:"plan"`
	PlanCode               string `json:"planCode"`
	AstraOrgGUID           string `json:"astraOrgGUID"`
	CloudProvider          string `json:"cloudProvider"`
	CloudProviderCode      string `json:"cloudProviderCode"`
	CloudRegion            string `json:"cloudRegion"`
	Status                 string `json:"status"`
	JvmVersion             string `json:"jvmVersion"`
	PulsarVersion          string `json:"pulsarVersion"`
	Email                  string `json:"Email"`
}


type StreamingTenant struct {
	Namespace              string `json:"namespace"`
	Topic                  string `json:"topic"`
	ID                     string `json:"id"`
	TenantName             string `json:"tenantName"`
	ClusterName            string `json:"clusterName"`
	WebServiceURL          string `json:"webServiceUrl"`
	BrokerServiceURL       string `json:"brokerServiceUrl"`
	WebsocketURL           string `json:"websocketUrl"`
	WebsocketQueryParamURL string `json:"websocketQueryParamUrl"`
	PulsarToken            string `json:"pulsarToken"`
	Plan                   string `json:"plan"`
	PlanCode               string `json:"planCode"`
	AstraOrgGUID           string `json:"astraOrgGUID"`
	CloudProvider          string `json:"cloudProvider"`
	CloudProviderCode      string `json:"cloudProviderCode"`
	CloudRegion            string `json:"cloudRegion"`
	Status                 string `json:"status"`
	JvmVersion             string `json:"jvmVersion"`
	PulsarVersion          string `json:"pulsarVersion"`
	Email                  string `json:"Email"`
}

func resourceStreamingTenantDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	region := resourceData.Get("region").(string)
	cloudProvider := resourceData.Get("cloud_provider").(string)

	id := resourceData.Id()

	tenantID, err := parseStreamingTenantID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	//tenantName:= resourceData.Get("tenant_name").(string)

	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserialize", orgBody)
	}

	params := astrastreaming.DeleteStreamingTenantParams{}
	cluster := fmt.Sprintf("pulsar-%s-%s", cloudProvider, region)
	deleteResponse, err := streamingClient.DeleteStreamingTenantWithResponse(ctx, tenantID, cluster, &params)
	if err != nil{
		return diag.FromErr(err)
	}
	if !strings.HasPrefix(deleteResponse.HTTPResponse.Status, "2") {
		return diag.Errorf("Error creating tenant %s", deleteResponse.Body)
	}

	// Deleted. Remove from state.
	resourceData.SetId("")

	return nil
}

func resourceStreamingTenantRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	id := resourceData.Id()

	tenantID, err := parseStreamingTenantID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	//tenantName:= resourceData.Get("tenant_name").(string)

	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserialize", orgBody)
	}

	getTenantResponse, err := streamingClient.GetStreamingTenantWithResponse(ctx, org.ID, tenantID)
	if err != nil{
		diag.FromErr(err)
	}
	if !strings.HasPrefix(getTenantResponse.HTTPResponse.Status, "2") {
		return diag.Errorf("Error creating tenant %s", getTenantResponse.Body)
	}

	var streamingTenant StreamingTenant
	err = json.Unmarshal(getTenantResponse.Body, &streamingTenant)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	if streamingTenant.TenantName == tenantID {
		if err := setStreamingTenantData(resourceData, tenantID); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Not found. Remove from state.
	resourceData.SetId("")

	return nil
}

func resourceStreamingTenantCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	//name := resourceData.Get("name").(string)
	topic := resourceData.Get("topic").(string)
	region := resourceData.Get("region").(string)
	cloudProvider := resourceData.Get("cloud_provider").(string)
	tenantName:= resourceData.Get("tenant_name").(string)
	userEmail:= resourceData.Get("user_email").(string)


	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	// Step 0
	streamingClustersResponse, _ := streamingClient.GetPulsarClustersWithResponse(ctx, org.ID)

	var streamingClusters StreamingClusters
	//bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(streamingClustersResponse.Body, &streamingClusters)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	for i :=0 ; i < len(streamingClusters) ; i++{
		fmt.Printf("body %s", streamingClusters[i].ClusterName)
		if streamingClusters[i].CloudProvider == cloudProvider{
			if streamingClusters[i].CloudRegion == region{
				// TODO - validation
				fmt.Printf("body %s", streamingClusters[i].ClusterName)
			}
		}
	}


	params := astrastreaming.IdOfCreateTenantEndpointParams{
		Topic: &topic,
	}

	tenantRequest := astrastreaming.IdOfCreateTenantEndpointJSONRequestBody{
		CloudProvider: &cloudProvider,
		CloudRegion:   &region,
		OrgID:         &org.ID,
		OrgName:       &org.ID,
		TenantName:    &tenantName,
		UserEmail:     &userEmail,
	}

	tenantCreationResponse, err := streamingClient.IdOfCreateTenantEndpoint(ctx, &params, tenantRequest)
	if err != nil{
		diag.FromErr(err)
	}
	if !strings.HasPrefix(tenantCreationResponse.Status, "2") {
		bodyBuffer, err = ioutil.ReadAll(tenantCreationResponse.Body)
		return diag.Errorf("Error creating tenant %s", tenantCreationResponse.Body)
	}
	bodyBuffer, err = ioutil.ReadAll(tenantCreationResponse.Body)

	var streamingTenant StreamingTenant
	err = json.Unmarshal(bodyBuffer, &streamingTenant)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	setStreamingTenantData(resourceData, streamingTenant.TenantName)


    return nil
}

func setStreamingTenantData(d *schema.ResourceData, tenantName string) error {
	d.SetId(fmt.Sprintf("%s", tenantName))

	if err := d.Set("tenant_name", tenantName); err != nil {
		return err
	}

	return nil
}

func parseStreamingTenantID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  errors.New("invalid role id format: expected tenantID/")
	}
	return idParts[0],  nil
}


