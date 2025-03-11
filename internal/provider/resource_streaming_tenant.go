package provider

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
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
			"tenant_name": {
				Description:  "Streaming tenant name.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])$"), "name must be atleast 2 characters and contain only alphanumeric characters"),
			},
			"topic": {
				Description:  "Streaming tenant topic. Please use the `astra_streaming_topic` resource instead.",
				Type:         schema.TypeString,
				Optional:     true,
				Deprecated:   "This field is deprecated and will be removed in a future release. Please use the `astra_streaming_topic` resource instead.",
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"cluster_name": {
				Description: "Pulsar cluster name.  Required if `cloud_provider` and `region` are not specified.",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
			},
			"cloud_provider": {
				Description:  "Cloud provider, one of `aws`, `gcp`, or `azure`.  Required if `cluster_name` is not set.",
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"region": {
				Description:      "Cloud provider region.  Required if `cluster_name` is not set.",
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				ForceNew:         true,
				RequiredWith:     []string{"cloud_provider"},
				ValidateFunc:     validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
				DiffSuppressFunc: streamingRegionSuppressDiff,
			},
			"user_email": {
				Description:  "User email for tenant.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"deletion_protection": {
				Description: "Whether or not to allow Terraform to destroy this tenant. Unless this field is set to false in Terraform state, a `terraform destroy` or `terraform apply` command that deletes the instance will fail. Defaults to `true`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
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
	UserMetricsUrl         string `json:"userMetricsUrl"`
	PulsarInstance         string `json:"pulsarInstance"`
	PulsarClusterDNS       string `json:"pulsarClusterDNS"`
	ClusterType            string `json:"clusterType"`
	AzType                 string `json:"azType"`
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

	id := resourceData.Id()

	tenantID, err := parseStreamingTenantID(id)
	if err != nil {
		return diag.FromErr(err)
	}

	params := astrastreaming.DeleteStreamingTenantParams{}
	cluster := resourceData.Get("cluster_name").(string)
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
	astraClient := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	tenantID, err := parseStreamingTenantID(resourceData.Id())
	if err != nil {
		return diag.Errorf("failed to parse tenannt ID: %v", err)
	}

	orgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		return diag.Errorf("failed to get current org ID: %v", err)
	}

	getTenantResponse, err := streamingClient.GetStreamingTenantWithResponse(ctx, orgID, tenantID)
	if err != nil {
		return diag.Errorf("failed to get streaming tenant: %v", err)
	}
	if getTenantResponse.HTTPResponse.StatusCode == 404 {
		// Tenant not found, remove it from the state
		resourceData.SetId("")
		return nil
	}
	if getTenantResponse.HTTPResponse.StatusCode != http.StatusOK {
		return diag.Errorf("invalid status code returned for tenant: %v", getTenantResponse.HTTPResponse.StatusCode)
	}

	if err := setStreamingTenantData(ctx, resourceData, *getTenantResponse.JSON200); err != nil {
		return diag.Errorf("failed to set streaming tenant data: %v", err)
	}
	return nil
}

func resourceStreamingTenantCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	clusterName := resourceData.Get("cluster_name").(string) // this can be used for dedicated plan that must specify a cluster name
	cloudProvider := resourceData.Get("cloud_provider").(string)
	region := resourceData.Get("region").(string)
	normalizedRegion := removeDashes(region)

	if clusterName == "" && (cloudProvider == "" || region == "") {
		return diag.Errorf("cluster_name or (cloud_provider and region) must be specified")
	}

	tenantName := resourceData.Get("tenant_name").(string)
	userEmail := resourceData.Get("user_email").(string)
	topic := resourceData.Get("topic").(string)

	astraClient := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	astraStreamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	orgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		return diag.FromErr(err)
	}

	tenantRequest := astrastreaming.IdOfCreateTenantEndpointJSONRequestBody{
		OrgID:      &orgID,
		OrgName:    &orgID,
		TenantName: &tenantName,
		UserEmail:  &userEmail,
	}
	if clusterName != "" {
		tenantRequest.ClusterName = &clusterName
	} else {
		tenantRequest.CloudProvider = &cloudProvider
		tenantRequest.CloudRegion = &normalizedRegion
	}

	params := astrastreaming.IdOfCreateTenantEndpointParams{
		Topic: &topic,
	}

	tenantCreationResponse, err := astraStreamingClient.IdOfCreateTenantEndpointWithResponse(ctx, &params, tenantRequest)
	if err != nil {
		return diag.Errorf("failed to create tenant: %v", err)
	}
	if tenantCreationResponse.StatusCode() != http.StatusOK {
		return diag.Errorf("failed to create tenant '%s' on cluster '%s'. Status Code: %d, Message: %s",
			tenantName, clusterName, tenantCreationResponse.StatusCode(), string(tenantCreationResponse.Body))
	}

	// Now let's fetch the tenant again so that it fills in the missing fields (like userMetricsUrl and tenant ID)
	streamingTenantResponse, err := astraStreamingClient.GetStreamingTenantWithResponse(ctx, orgID, tenantName)
	if err != nil {
		diag.FromErr(err)
	}
	if streamingTenantResponse.StatusCode() != http.StatusOK {
		return diag.Errorf("Unexpected response fetching tenant: %s. Response code: %d, message = %s", tenantName, streamingTenantResponse.StatusCode(), string(streamingTenantResponse.Body))
	}

	resourceData.SetId(tenantName)
	setStreamingTenantData(ctx, resourceData, *streamingTenantResponse.JSON200)

	return nil
}

func setStreamingTenantData(ctx context.Context, d *schema.ResourceData, tenantResponse astrastreaming.TenantClusterPlanResponse) error {
	if err := d.Set("cluster_name", *tenantResponse.ClusterName); err != nil {
		return err
	}
	if err := d.Set("cloud_provider", *tenantResponse.CloudProvider); err != nil {
		return err
	}
	if region, ok := d.Get("region").(string); !ok || region == "" {
		if err := d.Set("region", *tenantResponse.CloudProviderRegion); err != nil {
			return err
		}
	}
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

	if tenantResponse.UserMetricsURL != nil && *tenantResponse.UserMetricsURL != "" {
		if err := d.Set("user_metrics_url", *tenantResponse.UserMetricsURL); err != nil {
			return err
		}
	}
	if tenantResponse.Id != nil && *tenantResponse.Id != "" {
		if err := d.Set("tenant_id", *tenantResponse.Id); err != nil {
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

func streamingRegionSuppressDiff(k, oldValue, newValue string, d *schema.ResourceData) bool {
	return removeDashes(oldValue) == removeDashes(newValue)
}
