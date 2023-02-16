package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
			"retain_ordering": {
				Description:  "Retain ordering.",
				Type:         schema.TypeBool,
				Required:     true,
				ForceNew:     true,
			},
			"processing_guarantees": {
				Description:  "\"ATLEAST_ONCE\"\"ATMOST_ONCE\"\"EFFECTIVELY_ONCE\".",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
			"parallelism": {
				Description:  "Parallelism for Pulsar sink",
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
			},
			"namespace": {
				Description:  "Pulsar Namespace",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
			"sink_configs": {
				Description:  "Sink Configs",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
			"auto_ack": {
				Description:  "auto ack",
				Type:         schema.TypeBool,
				Required:     true,
				ForceNew:     true,
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

	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3


	tenantName := resourceData.Get("tenant_name").(string)
	sinkName := resourceData.Get("sink_name").(string)
	namespace := resourceData.Get("namespace").(string)

	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)


	pulsarCluster := GetPulsarCluster(cloudProvider, region)

	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserialize", orgBody)
	}


	token := meta.(astraClients).token
	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, err, streamingClient, tenantName)
	if err != nil {
		diag.FromErr(err)
	}

	deleteSinkParams := astrastreaming.DeleteSinkParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	deleteSinkResponse, err := streamingClientv3.DeleteSinkWithResponse(ctx, tenantName, namespace, sinkName, &deleteSinkParams)
	if err != nil{
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
	Tenant                     string      `json:"tenant"`
	Namespace                  string      `json:"namespace"`
	Name                       string      `json:"name"`
	ClassName                  string      `json:"className"`
	SourceSubscriptionName     interface{} `json:"sourceSubscriptionName"`
	SourceSubscriptionPosition string      `json:"sourceSubscriptionPosition"`
	Inputs                     interface{} `json:"inputs"`
	TopicToSerdeClassName      interface{} `json:"topicToSerdeClassName"`
	TopicsPattern              interface{} `json:"topicsPattern"`
	TopicToSchemaType          interface{} `json:"topicToSchemaType"`
	TopicToSchemaProperties    interface{} `json:"topicToSchemaProperties"`
	MaxMessageRetries interface{} `json:"maxMessageRetries"`
	DeadLetterTopic   interface{} `json:"deadLetterTopic"`
	Configs           struct {
		Password  string `json:"password"`
		JdbcURL   string `json:"jdbcUrl"`
		UserName  string `json:"userName"`
		TableName string `json:"tableName"`
	} `json:"configs"`
	Secrets                      interface{} `json:"secrets"`
	Parallelism                  int         `json:"parallelism"`
	ProcessingGuarantees         string      `json:"processingGuarantees"`
	RetainOrdering               bool        `json:"retainOrdering"`
	RetainKeyOrdering            bool        `json:"retainKeyOrdering"`
	Resources                    interface{} `json:"resources"`
	AutoAck                      bool        `json:"autoAck"`
	TimeoutMs                    interface{} `json:"timeoutMs"`
	NegativeAckRedeliveryDelayMs interface{} `json:"negativeAckRedeliveryDelayMs"`
	Archive                      string      `json:"archive"`
	CleanupSubscription          interface{} `json:"cleanupSubscription"`
	RuntimeFlags                 interface{} `json:"runtimeFlags"`
	CustomRuntimeOptions         interface{} `json:"customRuntimeOptions"`
}

func resourceStreamingSinkRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3


	tenantName := resourceData.Get("tenant_name").(string)
	sinkName := resourceData.Get("sink_name").(string)
	topic := resourceData.Get("topic").(string)
	namespace := resourceData.Get("namespace").(string)

	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)


	pulsarCluster := GetPulsarCluster(cloudProvider, region)

	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}


	token := meta.(astraClients).token
	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, err, streamingClient, tenantName)
	if err != nil {
		diag.FromErr(err)
	}

	getSinksParams := astrastreaming.GetSinksParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	getSinkResponse, err := streamingClientv3.GetSinksWithResponse(ctx, tenantName, namespace, sinkName, &getSinksParams)
	if err != nil{
		diag.FromErr(err)
	}
	if !strings.HasPrefix(getSinkResponse.Status(), "2") {
		return diag.Errorf("Error getting sinks %s", getSinkResponse.Body)
	}

	var sinkResponse SinkResponse
	json.Unmarshal(getSinkResponse.Body, sinkResponse)

	setStreamingSinkData(resourceData, tenantName, topic)

	return nil
}

func resourceStreamingSinkCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	rawRegion := resourceData.Get("region").(string)
	region := strings.ReplaceAll(rawRegion, "-", "")
	cloudProvider := resourceData.Get("cloud_provider").(string)
	tenantName := resourceData.Get("tenant_name").(string)

	sinkName := resourceData.Get("sink_name").(string)
	retainOrdering := resourceData.Get("retain_ordering").(bool)
	processingGuarantees := resourceData.Get("processing_guarantees").(string)
	parallelism := int32(resourceData.Get("parallelism").(int))
	namespace := resourceData.Get("namespace").(string)
	rawConfigs := resourceData.Get("sink_configs").(string)
	topic := resourceData.Get("topic").(string)
	autoAck := resourceData.Get("auto_ack").(bool)



	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	streamingClustersResponse, _ := streamingClient.GetPulsarClustersWithResponse(ctx, org.ID)

	var streamingClusters StreamingClusters
	//bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(streamingClustersResponse.Body, &streamingClusters)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	for i :=0 ; i < len(streamingClusters) ; i++{
		//fmt.Printf("body %s", streamingClusters[i].ClusterName)
		if streamingClusters[i].CloudProvider == cloudProvider{
			if streamingClusters[i].CloudRegion == region{
				// TODO - validation
				//fmt.Printf("body %s", streamingClusters[i].ClusterName)
			}
		}
	}

	pulsarCluster := GetPulsarCluster(cloudProvider, region)

	token := meta.(astraClients).token
	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, err, streamingClient, tenantName)
	if err != nil {
		diag.FromErr(err)
	}

	createSinkParams := astrastreaming.CreateSinkJSONParams{
		XDataStaxPulsarCluster: pulsarCluster,
		//XDataStaxCurrentOrg:    org.ID,
		XDataStaxCurrentOrg:    "",
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	getBuiltinSinkParams := astrastreaming.GetBuiltInSinksParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          pulsarToken,
	}

	builtinSinksResponse, err := streamingClientv3.GetBuiltInSinks(ctx, &getBuiltinSinkParams)
	if err != nil{
		diag.FromErr(err)
	}


	type SinkConfig []struct {
		Name              string      `json:"name"`
		Description       string      `json:"description"`
		SourceClass       interface{} `json:"sourceClass"`
		SinkClass         string      `json:"sinkClass"`
		SourceConfigClass interface{} `json:"sourceConfigClass"`
		SinkConfigClass   interface{} `json:"sinkConfigClass"`
	}

	var builtinSinks []map[string]interface{}

	bodyBuffer, err = ioutil.ReadAll(builtinSinksResponse.Body)
	json.Unmarshal(bodyBuffer, &builtinSinks)

	var sinkConfig map[string]interface{}

	for index := range builtinSinks {
		for key, element := range builtinSinks[index] {
			if key == "name" {
				if element == sinkName {
					sinkConfig = builtinSinks[index]
				}
			}

		}
	}

	var configs map[string]interface{}
	json.Unmarshal([]byte(rawConfigs), &configs)

	if sinkConfig == nil{
		return diag.Errorf("Could not find sink name %s in prebuilt sinks", sinkName)
	}

	archive := fmt.Sprintf("builtin://%s", sinkName)


	inputs := []string{topic}
	createSinkBody := astrastreaming.CreateSinkJSONJSONRequestBody{
		Archive:                      &archive,
		AutoAck:                      &autoAck,
		ClassName:                    nil,
		CleanupSubscription:          nil,
		Configs:                      &configs,
		CustomRuntimeOptions:         nil,
		DeadLetterTopic:              nil,
		InputSpecs:                   nil,
		Inputs:                       &inputs,
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
	if err != nil{
		diag.FromErr(err)
	}
	if !strings.HasPrefix(sinkCreationResponse.Status, "2") {
		bodyBuffer, err = ioutil.ReadAll(sinkCreationResponse.Body)
		return diag.Errorf("Error creating sink %s", bodyBuffer)
	}
	bodyBuffer, err = ioutil.ReadAll(sinkCreationResponse.Body)

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

func parseStreamingSinkID(id string) (string, string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  "", errors.New("invalid role id format: expected tenant_name/topic")
	}
	return idParts[0], idParts[1],  nil
}
