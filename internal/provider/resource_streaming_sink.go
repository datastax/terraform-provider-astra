package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &StreamingSinkResource{}
	_ resource.ResourceWithConfigure = &StreamingSinkResource{}
	//_ resource.ResourceWithImportState = &StreamingTenantResource{}
)

// NewStreamingTenantResource is a helper function to simplify the provider implementation.
func NewStreamingSinkResource() resource.Resource {
	return &StreamingSinkResource{}
}

// StreamingTenantResource is the resource implementation.
type StreamingSinkResource struct {
	clients *astraClients2
}

type StreamingSinkResourceModel struct {
	ID                   types.String `tfsdk:"id"` // Unique ID in the form cluster_name/tenant_name
	Cluster              types.String `tfsdk:"cluster"`
	PulsarClusterName    types.String `tfsdk:"pulsar_cluster"`
	CloudProvider        types.String `tfsdk:"cloud_provider"`
	Region               types.String `tfsdk:"region"`
	TenantName           types.String `tfsdk:"tenant_name"`
	Namespace            types.String `tfsdk:"namespace"`
	SinkName             types.String `tfsdk:"sink_name"`
	Archive              types.String `tfsdk:"archive"`
	Topic                types.String `tfsdk:"topic"`
	RetainOrdering       types.Bool   `tfsdk:"retain_ordering"`
	ProcessingGuarantees types.String `tfsdk:"processing_guarantees"`
	Parallelism          types.Int32  `tfsdk:"parallelism"`
	SinkConfigs          types.String `tfsdk:"sink_configs"`
	AutoAck              types.Bool   `tfsdk:"auto_ack"`
	DeletionProtection   types.Bool   `tfsdk:"deletion_protection"`
}

// Metadata returns the data source type name.
func (r *StreamingSinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streaming_sink"
}

func (r *StreamingSinkResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a streaming sink which sends data from a topic to a target system.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique ID in the form cluster_name/tenant_name/namespace/sink_name",
				Computed:    true,
			},
			"cluster": schema.StringAttribute{
				Description: "Name of the pulsar cluster in which to create the sink. If left blank, the name will be inferred from the cloud provider and region.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 32),
				},
			},
			"pulsar_cluster": schema.StringAttribute{
				DeprecationMessage: "`cluster` should be used instead of `pulsar_cluster`.",
				Description:        "Name of the pulsar cluster in which to create the sink. If left blank, the name will be inferred from the cloud provider and region.",
				Optional:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 32),
				},
			},
			"cloud_provider": schema.StringAttribute{
				DeprecationMessage: "`cluster` should be used instead of 'cloud_provider' and 'region'.",
				Description:        "Cloud provider (deprecated, use `cluster` instead)",
				Optional:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 32),
				},
			},
			"region": schema.StringAttribute{
				DeprecationMessage: "`cluster` should be used instead of 'cloud_provider' and 'region'.",
				Description:        "cloud region (deprecated, use `cluster` instead)",
				Optional:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 32),
				},
			},
			"tenant_name": schema.StringAttribute{
				Description: "Streaming tenant name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 64),
				},
			},
			"namespace": schema.StringAttribute{
				Description: "Pulsar Namespace",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 64),
				},
			},
			"sink_name": schema.StringAttribute{
				Description: "Name of the sink. Note that the combination of tenant, namespace, and sink name must not exceed 47 characters.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"archive": schema.StringAttribute{
				Description: "Name of the sink archive type to use. Defaults to the value of sink_name. Must be formatted as a URL, e.g. 'builtin://jdbc-clickhouse'",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"topic": schema.StringAttribute{
				Description: "Streaming tenant topic.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"retain_ordering": schema.BoolAttribute{
				Description: "Retain ordering.",
				Required:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"processing_guarantees": schema.StringAttribute{
				Description: "\"ATLEAST_ONCE\" \"ATMOST_ONCE\" \"EFFECTIVELY_ONCE\".",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parallelism": schema.Int32Attribute{
				Description: "Parallelism for Pulsar sink",
				Required:    true,
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
			},
			"sink_configs": schema.StringAttribute{
				Description: "Sink Configs",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"auto_ack": schema.BoolAttribute{
				Description: "auto ack",
				Required:    true,
				// PlanModifiers: []planmodifier.Bool{
				// 	boolplanmodifier.RequiresReplace(),
				// },
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Whether or not to allow Terraform to destroy this streaming sink. Unless this field is set to false in Terraform state, a `terraform destroy` or `terraform apply` command that deletes the instance will fail. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *StreamingSinkResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

func (r *StreamingSinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &StreamingSinkResourceModel{}
	diags := req.Plan.Get(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	astraStreamingClient := r.clients.astraStreamingClient

	orgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to get current OrgID",
			err.Error())
		return
	}

	normalizedRegion := removeDashes(plan.Region.ValueString())
	streamingClusterName := plan.Cluster.ValueString()
	// handle deprecated cluster_name field
	if streamingClusterName == "" {
		streamingClusterName = plan.PulsarClusterName.ValueString()
	}
	if streamingClusterName == "" && (plan.CloudProvider.ValueString() == "" || normalizedRegion == "") {
		resp.Diagnostics.AddError(
			"missing required configuration",
			"cluster_name or (cloud_provider and region) must be specified")
		return
	}
	streamingClusterName = getPulsarCluster(streamingClusterName, plan.CloudProvider.ValueString(), normalizedRegion, "")

	tenantName := plan.TenantName.ValueString()
	namespace := plan.Namespace.ValueString()
	sinkName := plan.SinkName.ValueString()
	archive := plan.Archive.ValueString()
	retainOrdering := plan.RetainOrdering.ValueBool()
	processingGuarantees := plan.ProcessingGuarantees.ValueString()
	parallelism := plan.Parallelism.ValueInt32()
	topic := plan.Topic.ValueString()
	autoAck := plan.AutoAck.ValueBool()

	var configs map[string]interface{}
	if err := json.Unmarshal([]byte(plan.SinkConfigs.ValueString()), &configs); err != nil {
		resp.Diagnostics.AddError(
			"invalid sink config",
			err.Error())
		return
	}

	createSinkParams := astrastreaming.CreateSinkJSONParams{
		XDataStaxPulsarCluster: streamingClusterName,
		XDataStaxCurrentOrg:    orgID,
		Authorization:          r.clients.token,
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

	sinkCreationResponse, err := astraStreamingClient.CreateSinkJSON(ctx, tenantName, namespace, sinkName, &createSinkParams, createSinkBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create sink",
			err.Error())
		return
	} else if sinkCreationResponse.StatusCode > 299 {
		body, _ := io.ReadAll(sinkCreationResponse.Body)
		errMsg := fmt.Sprintf("failed to create sink, status code: %d, message: %s", sinkCreationResponse.StatusCode, string(body))
		resp.Diagnostics.AddError(
			"failed to create sink",
			errMsg)
		return
	}

	plan.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *StreamingSinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	state := &StreamingSinkResourceModel{}
	diags := req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// astraClient := r.clients.astraClient
	astraStreamingClient := r.clients.astraStreamingClient

	streamingClusterName := state.getClusterName()
	tenantName := state.TenantName.ValueString()
	namespace := state.Namespace.ValueString()
	sinkName := state.SinkName.ValueString()

	getSinksParams := astrastreaming.GetSinksParams{
		XDataStaxPulsarCluster: streamingClusterName,
		Authorization:          r.clients.token,
	}

	getSinkResponse, err := astraStreamingClient.GetSinksWithResponse(ctx, tenantName, namespace, sinkName, &getSinksParams)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get sink, error: %v", err)
		resp.Diagnostics.AddError(
			"failed to get sink",
			errMsg)
		return
	} else if getSinkResponse.StatusCode() == 404 {
		// sink not found, remove it from the state
		resp.State.RemoveResource(ctx)
		return
	} else if getSinkResponse.StatusCode() > 299 {
		errMsg := fmt.Sprintf("failed to get sink, status code: %d, message: %s", getSinkResponse.StatusCode(), getSinkResponse.Body)
		resp.Diagnostics.AddError(
			"failed to get sink",
			errMsg)
		return
	}

	var sinkResponseData SinkResponse
	if err := json.Unmarshal(getSinkResponse.Body, &sinkResponseData); err != nil {
		resp.Diagnostics.AddError(
			"failed to unmarshal sink response",
			err.Error())
		return
	}

	setStreamingSinkData(sinkResponseData, state)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)

}

// Update currently only handles updating the deletion protection field.  TODO: add support for updating sink config.
func (r *StreamingSinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := &StreamingSinkResourceModel{}
	diags := req.Plan.Get(ctx, plan)
	resp.Diagnostics.Append(diags...)

	state := &StreamingSinkResourceModel{}
	diags = req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DeletionProtection = plan.DeletionProtection
	state.setID()
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *StreamingSinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StreamingSinkResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraStreamingClient := r.clients.astraStreamingClient

	streamingClusterName := state.getClusterName()
	if streamingClusterName == "" {
		streamingClusterName = state.Cluster.ValueString()
	}
	tenantName := state.TenantName.ValueString()
	namespace := state.Namespace.ValueString()
	sinkName := state.SinkName.ValueString()

	deleteSinkParams := astrastreaming.DeleteSinkParams{
		XDataStaxPulsarCluster: streamingClusterName,
		Authorization:          r.clients.token,
	}

	deleteSinkResponse, err := astraStreamingClient.DeleteSinkWithResponse(ctx, tenantName, namespace, sinkName, &deleteSinkParams)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get sink, error: %v", err)
		resp.Diagnostics.AddError(
			"failed to get sink",
			errMsg)
		return
	} else if deleteSinkResponse.StatusCode() > 299 {
		errMsg := fmt.Sprintf("failed to delete sink, status code: %d, message: %s", deleteSinkResponse.StatusCode(), deleteSinkResponse.Body)
		resp.Diagnostics.AddError(
			"failed to delete sink",
			errMsg)
		return
	}

	// Remove the resource from state
	resp.State.RemoveResource(ctx)
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

// setStreamingSinkData copies the data from the REST API endpoint response to the Terraform resource model.
func setStreamingSinkData(sink SinkResponse, data *StreamingSinkResourceModel) {
	data.TenantName = types.StringValue(sink.Tenant)
	data.Namespace = types.StringValue(sink.Namespace)
	data.SinkName = types.StringValue(sink.Name)
	data.Archive = types.StringValue(sink.Archive)
	data.RetainOrdering = types.BoolValue(sink.RetainOrdering)
	data.ProcessingGuarantees = types.StringValue(string(sink.ProcessingGuarantees))
	data.Parallelism = types.Int32Value(int32(sink.Parallelism))
	jsonSinkConfig, err := json.Marshal(sink.Configs)
	if err != nil {
		data.SinkConfigs = types.StringNull()

	}
	data.SinkConfigs = types.StringValue(string(jsonSinkConfig))
	data.AutoAck = types.BoolValue(sink.AutoAck)

	data.setID()
}

func (m *StreamingSinkResourceModel) setID() {
	clusterName := m.getClusterName()
	m.ID = types.StringValue(clusterName + "/" + m.TenantName.ValueString() + "/" + m.Namespace.ValueString() + "/" + m.SinkName.ValueString())
}

const (
	// sinkIDRegexPattern matches a sink ID in the format "cluster-name/tenant-name/namespace/sink-name".
	sinkIDRegexPattern = `^[A-Za-z][\w-.]+\/[A-Za-z][\w-.]+\/[A-Za-z][\w-.]+\/[A-Za-z][\w-.]+$`
	// oldSinkIDRegexPattern matches an old sink ID in the format "pgier-terraformtest-2/persistent://pgier-terraformtest-2/my-namespace/my-topic".
	oldSinkIDRegexPattern = `^[A-Za-z][\w-.]+\/(non-)?persistent:\/\/[A-Za-z][\w-.]+\/[A-Za-z][\w-.]+\/[A-Za-z][\w-.]+$`
)

var (
	sinkIDRegex    = regexp.MustCompile(sinkIDRegexPattern)
	oldSinkIDRegex = regexp.MustCompile(oldSinkIDRegexPattern)
)

// getClusterNameFromSinkID Try to determine the cluster name based on
// the fields in the model.
func (m *StreamingSinkResourceModel) getClusterName() string {
	if m.Cluster.ValueString() != "" {
		return m.Cluster.ValueString()
	}
	if m.PulsarClusterName.ValueString() != "" {
		return m.PulsarClusterName.ValueString()
	}
	if m.ID.ValueString() != "" {
		if sinkIDRegex.MatchString(m.ID.ValueString()) {
			// New format: cluster_name/tenant_name/namespace/sink_name
			parts := strings.SplitN(m.ID.ValueString(), "/", 2)
			if len(parts) > 0 {
				return parts[0]
			}
		}
	}
	return getPulsarCluster("", m.CloudProvider.ValueString(), removeDashes(m.Region.ValueString()), "")
}
