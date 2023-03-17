package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceStreamingTenant() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_streaming_tenant` creates an Astra Streaming tenant.",
		CreateContext: resourceStreamingTenantCreate,
		ReadContext:   resourceStreamingTenantRead,
		DeleteContext: resourceStreamingTenantDelete,
		UpdateContext: resourceStreamingTenantUpdate,

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
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])$"), "name must be atleast 2 characters and contain only alphanumeric characters"),
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
			// Optional
			"deletion_protection": {
				Description: "Whether or not to allow Terraform to destroy this tenant. Unless this field is set to false in Terraform state, a `terraform destroy` or `terraform apply` command that deletes the instance will fail. Defaults to `true`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
			// Computed
			"broker_service_url": {
				Description: "The Pulsar Binary Protocol URL used for production and consumption of messages.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"web_service_url": {
				Description: "URL used for administrative operations.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"web_socket_url": {
				Description: "URL used for web socket operations.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"web_socket_query_param_url": {
				Description: "URL used for web socket query parameter operations.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"cluster_name": {
				Description: "Pulsar cluster name.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			// The fields below are only filled in after creating the tenant and then retrieving the tenant info from the DevOps API
			"user_metrics_url": {
				Description: "URL for metrics.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"tenant_id": {
				Description: "UUID for the tenant.",
				Type:        schema.TypeString,
				Computed:    true,
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

func resourceStreamingTenantUpdate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// In-place update not supported. This is only here to support deletion_protection
	return nil
}

func resourceStreamingTenantDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if protectedFromDelete(resourceData) {
		return diag.Errorf("\"deletion_protection\" must be explicitly set to \"false\" in order to destroy astra_streaming_tenant")
	}
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)

	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")

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
	if err != nil {
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
	if err != nil {
		diag.FromErr(err)
	}
	if !strings.HasPrefix(getTenantResponse.HTTPResponse.Status, "2") {
		tflog.Info(ctx, fmt.Sprintf("Tenant not found with Name: %s", tenantID))
		resourceData.SetId("")
		return nil
	}

	streamingTenant := *getTenantResponse.JSON200

	if *streamingTenant.TenantName == tenantID {
		if err := setStreamingTenantData(ctx, resourceData, streamingTenant); err != nil {
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
	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)
	tenantName := resourceData.Get("tenant_name").(string)
	userEmail := resourceData.Get("user_email").(string)
	// this can be used for dedicated plan that must specify a cluster name
	clusterName := resourceData.Get("cluster_name").(string)

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

	for i := 0; i < len(streamingClusters); i++ {
		fmt.Printf("body %s", streamingClusters[i].ClusterName)
		if streamingClusters[i].CloudProvider == cloudProvider {
			if streamingClusters[i].CloudRegion == region {
				// TODO - validation
				//fmt.Printf("body %s", streamingClusters[i].ClusterName)
			}
		}
	}

	params := astrastreaming.IdOfCreateTenantEndpointParams{
		Topic: &topic,
	}

	tenantRequest := astrastreaming.IdOfCreateTenantEndpointJSONRequestBody{
		OrgID:      &org.ID,
		OrgName:    &org.ID,
		TenantName: &tenantName,
		UserEmail:  &userEmail,
	}

	if clusterName != "" {
		tenantRequest.ClusterName = &clusterName
	} else {
		tenantRequest.CloudProvider = &cloudProvider
		tenantRequest.CloudRegion = &region
	}

	tenantCreationResponse, err := streamingClient.IdOfCreateTenantEndpointWithResponse(ctx, &params, tenantRequest)
	if err != nil {
		return diag.FromErr(err)
	}
	if tenantCreationResponse.StatusCode() != http.StatusOK {
		return diag.Errorf("Error creating tenant. Status Code: %d, Message: %s", tenantCreationResponse.StatusCode(), string(tenantCreationResponse.Body))
	}

	// Now let's fetch the tenant again so that it fills in the missing fields (like userMetricsUrl and tenant ID)
	streamingTenantResponse, err := streamingClient.GetStreamingTenantWithResponse(ctx, org.ID, tenantName)
	if err != nil {
		diag.FromErr(err)
	}
	if streamingTenantResponse.StatusCode() != http.StatusOK {
		return diag.Errorf("Unexpected response fetching tenant: %s. Response code: %d, message = %s", tenantName, streamingTenantResponse.StatusCode(), string(streamingTenantResponse.Body))
	}

	setStreamingTenantData(ctx, resourceData, *streamingTenantResponse.JSON200)

	return nil
}

func setStreamingTenantData(ctx context.Context, d *schema.ResourceData, tenantResponse astrastreaming.TenantClusterPlanResponse) error {
	d.SetId(*tenantResponse.TenantName)

	if err := d.Set("tenant_name", *tenantResponse.TenantName); err != nil {
		return err
	}
	if err := d.Set("broker_service_url", *tenantResponse.PulsarURL); err != nil {
		return err
	}
	if err := d.Set("web_service_url", *tenantResponse.AdminURL); err != nil {
		return err
	}
	if err := d.Set("cluster_name", *tenantResponse.ClusterName); err != nil {
		return err
	}
	if err := d.Set("web_socket_url", *tenantResponse.WebsocketURL); err != nil {
		return err
	}
	if err := d.Set("web_socket_query_param_url", *tenantResponse.WebsocketQueryParamURL); err != nil {
		return err
	}
	// only set these if they aren't nil or empty
	metricsUrl := *tenantResponse.UserMetricsURL
	if len(metricsUrl) > 0 {
		if err := d.Set("user_metrics_url", metricsUrl); err != nil {
			return err
		}
	}
	tennantId := *&tenantResponse.Id
	if len(*tennantId) > 0 {
		if err := d.Set("tenant_id", tennantId); err != nil {
			return err
		}
	}
	return nil
}

func parseStreamingTenantID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "", errors.New("invalid tenant id format: expected tenantID/")
	}
	return idParts[0], nil
}
