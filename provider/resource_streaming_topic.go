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

func resourceStreamingTopic() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_streaming_topic` creates an Astra Streaming topic.",
		CreateContext: resourceStreamingTopicCreate,
		ReadContext:   resourceStreamingTopicRead,
		DeleteContext: resourceStreamingTopicDelete,

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
			"namespace": {
				Description:  "Pulsar Namespace",
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
			},
		},
	}
}



func resourceStreamingTopicDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	namespace := resourceData.Get("namespace").(string)
	topic := resourceData.Get("topic").(string)
	tenant := resourceData.Get("tenant_name").(string)

	cloudProvider := resourceData.Get("cloud_provider").(string)
	rawRegion := resourceData.Get("region").(string)

	token := meta.(astraClients).token

	region := strings.ReplaceAll(rawRegion, "-", "")
	pulsarCluster := GetPulsarCluster(cloudProvider, region)
	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, err, streamingClient, tenant)

	deleteTopicParams := astrastreaming.DeleteTopicParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	deleteTopicResponse, err := streamingClientv3.DeleteTopic(ctx, tenant, namespace, topic, &deleteTopicParams)
	if err != nil{
		diag.FromErr(err)
	}

	if !strings.HasPrefix(deleteTopicResponse.Status, "2") {
		bodyBuffer, err = ioutil.ReadAll(deleteTopicResponse.Body)
		return diag.Errorf("Error deleting topic %s", bodyBuffer)
	}
	bodyBuffer, err = ioutil.ReadAll(deleteTopicResponse.Body)

	resourceData.SetId("")

	return nil
}

func resourceStreamingTopicRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	namespace := resourceData.Get("namespace").(string)
	topic := resourceData.Get("topic").(string)
	tenant := resourceData.Get("tenant_name").(string)

	cloudProvider := resourceData.Get("cloud_provider").(string)
	rawRegion := resourceData.Get("region").(string)

	token := meta.(astraClients).token

	region := strings.ReplaceAll(rawRegion, "-", "")
	pulsarCluster := GetPulsarCluster(cloudProvider, region)
	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, err, streamingClient, tenant)

	getTopicsParams := astrastreaming.GetTopicsParams{
		XDataStaxPulsarCluster: &pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	createTopicResponse, err := streamingClientv3.GetTopics(ctx, tenant, namespace, &getTopicsParams)
	if err != nil{
		diag.FromErr(err)
	}

	if !strings.HasPrefix(createTopicResponse.Status, "2") {
		bodyBuffer, err = ioutil.ReadAll(createTopicResponse.Body)
		return diag.Errorf("Error reading topic %s", bodyBuffer)
	}
	bodyBuffer, err = ioutil.ReadAll(createTopicResponse.Body)

	//TODO: validate that our topic is there

	setStreamingTopicData(resourceData, tenant, topic)

	return nil
}

func resourceStreamingTopicCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	streamingClient := meta.(astraClients).astraStreamingClient.(*astrastreaming.ClientWithResponses)
	client := meta.(astraClients).astraClient.(*astra.ClientWithResponses)
	streamingClientv3 := meta.(astraClients).astraStreamingClientv3

	namespace := resourceData.Get("namespace").(string)
	topic := resourceData.Get("topic").(string)
	tenant := resourceData.Get("tenant_name").(string)

	cloudProvider := resourceData.Get("cloud_provider").(string)
	rawRegion := resourceData.Get("region").(string)

	token := meta.(astraClients).token

	region := strings.ReplaceAll(rawRegion, "-", "")
	pulsarCluster := GetPulsarCluster(cloudProvider, region)
	orgBody, _ := client.GetCurrentOrganization(ctx)

	var org OrgId
	bodyBuffer, err := ioutil.ReadAll(orgBody.Body)

	err = json.Unmarshal(bodyBuffer, &org)
	if err != nil {
		fmt.Println("Can't deserislize", orgBody)
	}

	pulsarToken, err := getPulsarToken(ctx, pulsarCluster, token, org, err, streamingClient, tenant)

	createTopicParams := astrastreaming.CreateTopicParams{
		XDataStaxCurrentOrg: &org.ID,
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	createTopicResponse, err := streamingClientv3.CreateTopic(ctx, tenant, namespace, topic, &createTopicParams)
	if err != nil{
		diag.FromErr(err)
	}

	if !strings.HasPrefix(createTopicResponse.Status, "2") {
		bodyBuffer, err = ioutil.ReadAll(createTopicResponse.Body)
		return diag.Errorf("Error creating topic %s", bodyBuffer)
	}
	bodyBuffer, err = ioutil.ReadAll(createTopicResponse.Body)

	setStreamingTopicData(resourceData, tenant, topic)

    return nil
}

func setStreamingTopicData(d *schema.ResourceData, tenantName string, topic string) error {
	d.SetId(fmt.Sprintf("%s/%s", tenantName, topic))

	if err := d.Set("tenant_name", tenantName); err != nil {
		return err
	}
	if err := d.Set("topic", topic); err != nil {
		return err
	}


	return nil
}

func parseStreamingTopicID(id string) (string, string, error) {
	idParts := strings.Split(strings.ToLower(id), "/")
	if len(idParts) != 1 {
		return "",  "", errors.New("invalid role id format: expected tenant_name/topic")
	}
	return idParts[0], idParts[1],  nil
}
