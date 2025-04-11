package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/datastax/astra-client-go/v2/astra"
	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceStreamingSink() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_streaming_sink` creates a streaming sink which sends data from a topic to a target system.",
		CreateContext: resourceStreamingSinkCreate,
		ReadContext:   resourceStreamingSinkRead,
		DeleteContext: resourceStreamingSinkDelete,
		UpdateContext: resourceStreamingSinkUpdate,

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
			"pulsar_cluster": {
				Description: "Name of the pulsar cluster in which to create the sink.  If left blank, the name will be inferred from the" +
					"cloud provider and region",
				Type:         schema.TypeString,
				Optional:     true,
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
			"sink_name": {
				Description:  "Name of the sink.",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"archive": {
				Description:  "Name of the sink archive type to use. Defaults to the value of sink_name.  Must be formatted as a URL, e.g. 'builtin://jdbc-clickhouse",
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile("^.{2,}"), "name must be atleast 2 characters"),
			},
			"retain_ordering": {
				Description: "Retain ordering.",
				Type:        schema.TypeBool,
				Required:    true,
				ForceNew:    true,
			},
			"processing_guarantees": {
				Description: "\"ATLEAST_ONCE\"\"ATMOST_ONCE\"\"EFFECTIVELY_ONCE\".",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"parallelism": {
				Description: "Parallelism for Pulsar sink",
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
			},
			"namespace": {
				Description: "Pulsar Namespace",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"sink_configs": {
				Description: "Sink Configs",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"auto_ack": {
				Description: "auto ack",
				Type:        schema.TypeBool,
				Required:    true,
				ForceNew:    true,
			},
			// Optional
			"deletion_protection": {
				Description: "Whether or not to allow Terraform to destroy this streaming sink. Unless this field is set to false in Terraform state, a `terraform destroy` or `terraform apply` command that deletes the instance will fail. Defaults to `true`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
		},
	}
}

func resourceStreamingSinkUpdate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// In-place update not supported. This is only here to support deletion_protection
	return nil
}

func resourceStreamingSinkDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if protectedFromDelete(resourceData) {
		return diag.Errorf("\"deletion_protection\" must be explicitly set to \"false\" in order to destroy astra_streaming_sink")
	}

	astraClient := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	tenantName := resourceData.Get("tenant_name").(string)
	sinkName := resourceData.Get("sink_name").(string)
	namespace := resourceData.Get("namespace").(string)

	pulsarClusterName := resourceData.Get("pulsar_cluster").(string)
	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)

	pulsarCluster := getPulsarCluster(pulsarClusterName, cloudProvider, region, "")

	orgResp, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.Errorf("failed to get current organization ID: %v", err)
	}

	var org OrgId
	if err := json.NewDecoder(orgResp.Body).Decode(&org); err != nil {
		return diag.Errorf("failed to decode current organization ID: %v", err)
	}

	deleteSinkParams := astrastreaming.DeleteSinkParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          meta.(astraClients).token,
	}

	deleteSinkResponse, err := streamingClientv3.DeleteSinkWithResponse(ctx, tenantName, namespace, sinkName, &deleteSinkParams)
	if err != nil {
		diag.FromErr(err)
	}
	if !strings.HasPrefix(deleteSinkResponse.Status(), "2") {
		return diag.Errorf("Error deleting sink %s", deleteSinkResponse.Body)
	}

	// Not found. Remove from state.
	resourceData.SetId("")

	return nil
}

type SinkResponse struct {
	Tenant                       string                 `json:"tenant"`
	Namespace                    string                 `json:"namespace"`
	Name                         string                 `json:"name"`
	ClassName                    string                 `json:"className"`
	SourceSubscriptionName       interface{}            `json:"sourceSubscriptionName"`
	SourceSubscriptionPosition   string                 `json:"sourceSubscriptionPosition"`
	Inputs                       interface{}            `json:"inputs"`
	TopicToSerdeClassName        interface{}            `json:"topicToSerdeClassName"`
	TopicsPattern                interface{}            `json:"topicsPattern"`
	TopicToSchemaType            interface{}            `json:"topicToSchemaType"`
	TopicToSchemaProperties      interface{}            `json:"topicToSchemaProperties"`
	MaxMessageRetries            interface{}            `json:"maxMessageRetries"`
	DeadLetterTopic              interface{}            `json:"deadLetterTopic"`
	Configs                      map[string]interface{} `json:"configs"`
	Secrets                      interface{}            `json:"secrets"`
	Parallelism                  int                    `json:"parallelism"`
	ProcessingGuarantees         string                 `json:"processingGuarantees"`
	RetainOrdering               bool                   `json:"retainOrdering"`
	RetainKeyOrdering            bool                   `json:"retainKeyOrdering"`
	Resources                    interface{}            `json:"resources"`
	AutoAck                      bool                   `json:"autoAck"`
	TimeoutMs                    interface{}            `json:"timeoutMs"`
	NegativeAckRedeliveryDelayMs interface{}            `json:"negativeAckRedeliveryDelayMs"`
	Archive                      string                 `json:"archive"`
	CleanupSubscription          interface{}            `json:"cleanupSubscription"`
	RuntimeFlags                 interface{}            `json:"runtimeFlags"`
	CustomRuntimeOptions         interface{}            `json:"customRuntimeOptions"`
}

func resourceStreamingSinkRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	astraClient := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	tenantName := resourceData.Get("tenant_name").(string)
	sinkName := resourceData.Get("sink_name").(string)
	topic := resourceData.Get("topic").(string)
	namespace := resourceData.Get("namespace").(string)

	pulsarClusterName := resourceData.Get("pulsar_cluster").(string)
	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)

	pulsarCluster := getPulsarCluster(pulsarClusterName, cloudProvider, region, "")

	orgBody, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.Errorf("failed to get current organization ID: %v", err)
	}

	var org OrgId
	if err = json.NewDecoder(orgBody.Body).Decode(&org); err != nil {
		return diag.Errorf("failed to decode current organization ID: %v", err)
	}

	getSinksParams := astrastreaming.GetSinksParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          meta.(astraClients).token,
	}

	getSinkResponse, err := streamingClientv3.GetSinksWithResponse(ctx, tenantName, namespace, sinkName, &getSinksParams)
	if err != nil {
		diag.FromErr(err)
	} else if getSinkResponse.StatusCode() == 404 {
		// sink not found, remove it from the state
		resourceData.SetId("")
		return nil
	} else if getSinkResponse.StatusCode() > 299 {
		return diag.Errorf("failed to get sink, status code %d, message: %s", getSinkResponse.StatusCode(), getSinkResponse.Body)
	}

	var sinkResponse SinkResponse
	if err := json.Unmarshal(getSinkResponse.Body, &sinkResponse); err != nil {
		return diag.Errorf("failed to read sink response: %v", err)
	}

	setStreamingSinkData(resourceData, tenantName, topic)

	return nil
}

func resourceStreamingSinkCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	astraClient := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)

	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)
	tenantName := resourceData.Get("tenant_name").(string)
	pulsarClusterName := resourceData.Get("pulsar_cluster").(string)

	sinkName := resourceData.Get("sink_name").(string)
	archive := resourceData.Get("archive").(string)
	retainOrdering := resourceData.Get("retain_ordering").(bool)
	processingGuarantees := resourceData.Get("processing_guarantees").(string)
	parallelism := int32(resourceData.Get("parallelism").(int))
	namespace := resourceData.Get("namespace").(string)
	rawConfigs := resourceData.Get("sink_configs").(string)
	topic := resourceData.Get("topic").(string)
	autoAck := resourceData.Get("auto_ack").(bool)

	if archive == "" {
		archive = fmt.Sprintf("builtin://%s", sinkName)
	}

	orgResp, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.Errorf("failed to get current organization ID: %v", err)
	}

	var org OrgId
	if err := json.NewDecoder(orgResp.Body).Decode(&org); err != nil {
		return diag.Errorf("failed to decode current organization ID: %v", err)
	}

	streamingClustersResponse, err := streamingClientv3.GetPulsarClustersWithResponse(ctx, org.ID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to request pulsar clusters: %w", err))
	}

	if streamingClustersResponse.StatusCode() != http.StatusOK {
		return diag.FromErr(fmt.Errorf("failed to read pulsar clusters. Status code: %s, msg:\n%s", streamingClustersResponse.Status(), string(streamingClustersResponse.Body)))
	}

	pulsarCluster := getPulsarCluster(pulsarClusterName, cloudProvider, region, "")

	var configs map[string]interface{}
	if err := json.Unmarshal([]byte(rawConfigs), &configs); err != nil {
		return diag.Errorf("failed to unmarshal sink config: %v", err)
	}

	createSinkParams := astrastreaming.CreateSinkJSONParams{
		XDataStaxPulsarCluster: pulsarCluster,
		XDataStaxCurrentOrg:    "",
		Authorization:          meta.(astraClients).token,
	}

	sinkInputs := []string{topic}
	createSinkBody := astrastreaming.CreateSinkJSONJSONRequestBody{
		Archive:                      &archive,
		AutoAck:                      &autoAck,
		ClassName:                    nil,
		CleanupSubscription:          nil,
		Configs:                      &configs,
		CustomRuntimeOptions:         nil,
		DeadLetterTopic:              nil,
		InputSpecs:                   nil,
		Inputs:                       &sinkInputs,
		MaxMessageRetries:            nil,
		Name:                         &sinkName,
		Namespace:                    &namespace,
		NegativeAckRedeliveryDelayMs: nil,
		Parallelism:                  &parallelism,
		ProcessingGuarantees:         (*astrastreaming.SinkConfigProcessingGuarantees)(&processingGuarantees),
		Resources:                    nil,
		RetainKeyOrdering:            nil,
		RetainOrdering:               &retainOrdering,
		RuntimeFlags:                 nil,
		Secrets:                      nil,
		SinkType:                     nil,
		SourceSubscriptionName:       nil,
		SourceSubscriptionPosition:   nil,
		Tenant:                       &tenantName,
		TimeoutMs:                    nil,
		TopicToSchemaProperties:      nil,
		TopicToSchemaType:            nil,
		TopicToSerdeClassName:        nil,
		TopicsPattern:                nil,
	}

	sinkCreationResponse, err := streamingClientv3.CreateSinkJSON(ctx, tenantName, namespace, sinkName, &createSinkParams, createSinkBody)
	if err != nil {
		diag.FromErr(err)
	}
	if sinkCreationResponse.StatusCode > 299 {
		bodyBuffer, _ := io.ReadAll(sinkCreationResponse.Body)
		return diag.Errorf("failed to  create sink, status code: %d, message %s", sinkCreationResponse.StatusCode, bodyBuffer)
	}

	setStreamingSinkData(resourceData, tenantName, topic)

	return nil
}

func setStreamingSinkData(d *schema.ResourceData, tenantName string, topic string) error {
	d.SetId(fmt.Sprintf("%s/%s", tenantName, topic))

	if err := d.Set("tenant_name", tenantName); err != nil {
		return err
	}
	if err := d.Set("topic", topic); err != nil {
		return err
	}

	return nil
}
