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

func resourceStreamingTopic() *schema.Resource {
	return &schema.Resource{
		Description:   "`astra_streaming_topic` creates an Astra Streaming topic.",
		CreateContext: resourceStreamingTopicCreate,
		ReadContext:   resourceStreamingTopicRead,
		DeleteContext: resourceStreamingTopicDelete,
		UpdateContext: resourceStreamingTopicUpdate,

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
				Description: "Pulsar Namespace",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			// Optional
			"deletion_protection": {
				Description: "Whether or not to allow Terraform to destroy this streaming topic. Unless this field is set to false in Terraform state, a `terraform destroy` or `terraform apply` command that deletes the instance will fail. Defaults to `true`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
			},
		},
	}
}

func resourceStreamingTopicUpdate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// In-place update not supported. This is only here to support deletion_protection
	return nil
}

func resourceStreamingTopicDelete(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if protectedFromDelete(resourceData) {
		return diag.Errorf("\"deletion_protection\" must be explicitly set to \"false\" in order to destroy astra_streaming_topic")
	}
	astraClients := meta.(astraClients)
	astraClient := astraClients.astraClient.(*astra.ClientWithResponses)

	namespace := resourceData.Get("namespace").(string)
	topic := resourceData.Get("topic").(string)
	tenant := resourceData.Get("tenant_name").(string)

	cloudProvider := resourceData.Get("cloud_provider").(string)
	rawRegion := resourceData.Get("region").(string)

	token := astraClients.token

	region := strings.ReplaceAll(rawRegion, "-", "")
	pulsarCluster := getPulsarCluster(cloudProvider, region, astraClients.streamingTestMode)
	orgResp, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.Errorf("Failed to read orgnization ID: %v", err)
	}
	var org OrgId
	err = json.NewDecoder(orgResp.Body).Decode(&org)
	if err != nil {
		return diag.Errorf("Failed to read orgnization ID: %v", err)
	}

	pulsarToken, err := getLatestPulsarToken(ctx, astraClients.astraStreamingClientv3, token, org.ID, pulsarCluster, tenant)
	if err != nil {
		return diag.FromErr(err)
	}

	deleteTopicParams := astrastreaming.DeleteTopicParams{
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	deleteTopicResponse, err := astraClients.astraStreamingClientv3.DeleteTopic(ctx, tenant, namespace, topic, &deleteTopicParams)
	if err != nil {
		return diag.FromErr(err)
	} else if deleteTopicResponse.StatusCode > 299 {
		bodyBuffer, _ := ioutil.ReadAll(deleteTopicResponse.Body)
		return diag.Errorf("Error deleting topic %s", bodyBuffer)
	}

	resourceData.SetId("")

	return nil
}

func resourceStreamingTopicRead(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	astraClients := meta.(astraClients)
	astraClient := astraClients.astraClient.(*astra.ClientWithResponses)

	namespace := resourceData.Get("namespace").(string)
	topic := resourceData.Get("topic").(string)
	tenant := resourceData.Get("tenant_name").(string)

	cloudProvider := resourceData.Get("cloud_provider").(string)
	rawRegion := resourceData.Get("region").(string)

	token := astraClients.token

	region := strings.ReplaceAll(rawRegion, "-", "")
	pulsarCluster := getPulsarCluster(cloudProvider, region, astraClients.streamingTestMode)
	orgBody, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.Errorf("Failed to get organization ID: %v", err)
	}
	var org OrgId
	err = json.NewDecoder(orgBody.Body).Decode(&org)
	if err != nil {
		return diag.Errorf("Failed to read organization ID: %v", err)
	}

	pulsarToken, err := getLatestPulsarToken(ctx, astraClients.astraStreamingClientv3, token, org.ID, pulsarCluster, tenant)
	if err != nil {
		return diag.Errorf("Failed to get pulsar token: %v", err)
	}

	getTopicsParams := astrastreaming.GetTopicsParams{
		XDataStaxPulsarCluster: &pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	readTopicResponse, err := astraClients.astraStreamingClientv3.GetTopics(ctx, tenant, namespace, &getTopicsParams)
	if err != nil {
		return diag.Errorf("Failed to get topic list: %v", err)
	} else if readTopicResponse.StatusCode > 299 {
		bodyBuffer, _ := ioutil.ReadAll(readTopicResponse.Body)
		return diag.Errorf("Error reading topic %s", bodyBuffer)
	}
	//TODO: validate that our topic is there

	setStreamingTopicData(resourceData, tenant, topic)

	return nil
}

func resourceStreamingTopicCreate(ctx context.Context, resourceData *schema.ResourceData, meta interface{}) diag.Diagnostics {
	astraClients := meta.(astraClients)
	astraClient := astraClients.astraClient.(*astra.ClientWithResponses)

	namespace := resourceData.Get("namespace").(string)
	topic := resourceData.Get("topic").(string)
	tenant := resourceData.Get("tenant_name").(string)

	cloudProvider := resourceData.Get("cloud_provider").(string)
	rawRegion := resourceData.Get("region").(string)

	token := astraClients.token

	region := strings.ReplaceAll(rawRegion, "-", "")
	pulsarCluster := getPulsarCluster(cloudProvider, region, astraClients.streamingTestMode)

	orgResp, err := astraClient.GetCurrentOrganization(ctx)
	if err != nil {
		return diag.Errorf("Failed to get current organization: %v", err)
	}
	org := OrgId{}
	err = json.NewDecoder(orgResp.Body).Decode(&org)
	if err != nil {
		return diag.Errorf("Failed to read organization: %v", err)
	}

	pulsarToken, err := getLatestPulsarToken(ctx, astraClients.astraStreamingClientv3, token, org.ID, pulsarCluster, tenant)
	if err != nil {
		return diag.FromErr(err)
	}

	createTopicParams := astrastreaming.CreateTopicParams{
		XDataStaxCurrentOrg:    &org.ID,
		XDataStaxPulsarCluster: pulsarCluster,
		Authorization:          fmt.Sprintf("Bearer %s", pulsarToken),
	}

	createTopicResponse, err := astraClients.astraStreamingClientv3.CreateTopic(ctx, tenant, namespace, topic, &createTopicParams, setContentTypeHeader("application/json"))
	if err != nil {
		return diag.Errorf("Error creating topic: %v", err)
	} else if createTopicResponse.StatusCode > 299 {
		bodyBuffer, _ := ioutil.ReadAll(createTopicResponse.Body)
		return diag.Errorf("Error creating topic %s", bodyBuffer)
	}

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
		return "", "", errors.New("invalid role id format: expected tenant_name/topic")
	}
	return idParts[0], idParts[1], nil
}
