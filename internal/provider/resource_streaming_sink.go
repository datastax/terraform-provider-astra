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

func resourceStreamingSink() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_cdc` enables cdc for an Astra Serverless table.",
		CreateContext: resourceStreamingSinkCreate,
		ReadContext:   resourceStreamingSinkRead,
		DeleteContext: resourceStreamingSinkDelete,

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
		},
	}
}



func resourceStreamingSinkDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	//TODO: call delete endpoint

	// Deleted. Remove from state.
	resourceData.SetId("")

	return nil
}

func resourceStreamingSinkRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	/*
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)


	id := resourceData.Id()

	tenantID, err := parseStreamingSinkID(id)
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
		return diag.Errorf("Error reading tenant %s", getTenantResponse.Body)
	}

	var streamingTenant StreamingTenant
	err = json.Unmarshal(getTenantResponse.Body, &streamingTenant)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	if streamingTenant.TenantName == tenantID {
		if err := setStreamingSinkData(resourceData, tenantID); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Not found. Remove from state.
	resourceData.SetId("")

	*/

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
		fmt.Printf("body %s", streamingClusters[i].ClusterName)
		if streamingClusters[i].CloudProvider == cloudProvider{
			if streamingClusters[i].CloudRegion == region{
				// TODO - validation
				fmt.Printf("body %s", streamingClusters[i].ClusterName)
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

	inputSpecs := astrastreaming.SinkConfig_InputSpecs{
		AdditionalProperties: map[string]astrastreaming.ConsumerConfig{
			topic: {
				ConsumerProperties: nil,
				CryptoConfig:       nil,
				PoolMessages:       nil,
				ReceiverQueueSize:  nil,
				RegexPattern:       nil,
				SchemaProperties:   nil,
				SchemaType:         nil,
				SerdeClassName:     nil,
		    },
		},
	}

	inputs := []string{topic}
	createSinkBody := astrastreaming.CreateSinkJSONJSONRequestBody{
		Archive:                      &archive,
		AutoAck:                      &autoAck,
		ClassName:                    nil,
		CleanupSubscription:          nil,
		Configs:                      &configs,
		CustomRuntimeOptions:         nil,
		DeadLetterTopic:              nil,
		InputSpecs:                   &inputSpecs,
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
		return diag.Errorf("Error creating tenant %s", bodyBuffer)
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

func parseStreamingSinkID(id string) (string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  errors.New("invalid role id format: expected tenantID/")
	}
	return idParts[0],  nil
}
