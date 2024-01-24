package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/datastax/pulsar-admin-client-go/src/pulsaradmin"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &StreamingTopicResource{}
	_ resource.ResourceWithConfigure   = &StreamingTopicResource{}
	_ resource.ResourceWithImportState = &StreamingTopicResource{}
)

// NewStreamingTopicResource is a helper function to simplify the provider implementation.
func NewStreamingTopicResource() resource.Resource {
	return &StreamingTopicResource{}
}

// StreamingTopicResource is the resource implementation.
type StreamingTopicResource struct {
	clients *astraClients2
}

// StreamingTopicResourceModel maps the resource schema data.
type StreamingTopicResourceModel struct {
	ID                 types.String          `tfsdk:"id" json:"id,omitempty"`
	Cluster            types.String          `tfsdk:"cluster" json:"cluster,omitempty"`
	CloudProvider      types.String          `tfsdk:"cloud_provider" json:"cloud_provider,omitempty"`
	Region             types.String          `tfsdk:"region" json:"region,omitempty"`
	Tenant             types.String          `tfsdk:"tenant" json:"tenant,omitempty"`
	TenantName         types.String          `tfsdk:"tenant_name" json:"tenant_name,omitempty"`
	Namespace          types.String          `tfsdk:"namespace" json:"namespace,omitempty"`
	Topic              types.String          `tfsdk:"topic" json:"topic,omitempty"`
	Persistent         types.Bool            `tfsdk:"persistent" json:"persistent,omitempty"`
	Partitioned        types.Bool            `tfsdk:"partitioned" json:"partitioned,omitempty"`
	NumPartitions      types.Int64           `tfsdk:"num_partitions" json:"num_partitions,omitempty"`
	DeletionProtection types.Bool            `tfsdk:"deletion_protection" json:"deletion_protection,omitempty"`
	Schema             *StreamingTopicSchema `tfsdk:"schema" json:"schema,omitempty"`
	TopicFQN           types.String          `tfsdk:"topic_fqn" json:"topic_fqn,omitempty"`
}

type StreamingTopicSchema struct {
	Type       *string            `tfsdk:"type" json:"type"`
	Schema     *string            `tfsdk:"schema" json:"schema"`
	Properties *map[string]string `tfsdk:"properties" json:"properties"`
}

// Metadata returns the data source type name.
func (r *StreamingTopicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streaming_topic"
}

// Schema defines the schema for the data source.
func (r *StreamingTopicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Pulsar Topic.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Full path to the topic",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster": schema.StringAttribute{
				Description: "Cluster where the Astra Streaming tenant is located.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Description:        "**Deprecated** Cloud provider where the  Astra Streaming tenant is located.",
				DeprecationMessage: "Please use `cluster` instead.",
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				Description:        "**Deprecated** Region where the  Astra Streaming tenant is located.",
				Optional:           true,
				Computed:           true,
				DeprecationMessage: "Please use `cluster` instead.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
					// planModifierRemoveDashes(),
				},
			},
			"tenant": schema.StringAttribute{
				Description: "Name of the streaming tenant.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(planModifierStringValueChanged(),
						"If the value of this string changes, Terraform will destroy and recreate the resource.",
						"If the value of this string changes, Terraform will destroy and recreate the resource.",
					),
				},
				Validators: []validator.String{
					// Validate only this attribute or tenant_name is configured.
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("tenant_name"),
					}...),
				},
			},
			"tenant_name": schema.StringAttribute{
				Description:        "**Deprecated** Name of the streaming tenant.",
				Optional:           true,
				DeprecationMessage: "Please use `tenant` instead.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIf(planModifierStringValueChanged(),
						"If the value of this string changes, Terraform will destroy and recreate the resource.",
						"If the value of this string changes, Terraform will destroy and recreate the resource.",
					),
				},
			},
			"namespace": schema.StringAttribute{
				Description: "Pulsar namespace of the topic.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"topic": schema.StringAttribute{
				Description: "Name of the topic",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"persistent": schema.BoolAttribute{
				Description: "Persistent or non-persistent topic",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"partitioned": schema.BoolAttribute{
				Description: "Partitioned or non-partitioned topic",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"num_partitions": schema.Int64Attribute{
				Description: "Number of partitions for a partitioned topic.  This field must not be set for a non-partitioned topic.",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Prevent this topic from being deleted via Terraform",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"schema": schema.SingleNestedAttribute{
				Description: "Pulsar topic schema.",
				Optional:    true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Description: "Type of the schema, e.g. JSON",
						Required:    true,
					},
					"schema": schema.StringAttribute{
						Description: "Schema definition",
						Required:    true,
					},
					"properties": schema.MapAttribute{
						Description: "Additional properties",
						Optional:    true,
						ElementType: types.StringType,
					},
				},
			},
			"topic_fqn": schema.StringAttribute{
				Description: "Fully qualified name of the topic, for example 'persistent://mytenant/namespace1/mytopic'",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *StreamingTopicResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

// Create creates the resource and sets the initial Terraform state.
func (r *StreamingTopicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StreamingTopicResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cluster := getPulsarCluster(plan.Cluster.ValueString(), plan.CloudProvider.ValueString(), plan.Region.ValueString(), r.clients.streamingClusterSuffix)
	plan.Cluster = types.StringValue(cluster)

	cloudProvider, region, err := getProviderRegionFromClusterName(cluster)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse cluster name",
			err.Error(),
		)
	}
	plan.CloudProvider = types.StringValue(cloudProvider)
	if plan.Region.IsNull() || plan.Region.IsUnknown() {
		plan.Region = types.StringValue(region)
	}

	tenant := plan.Tenant.ValueString()
	if tenant == "" {
		tenant = plan.TenantName.ValueString()
	}
	namespace := plan.Namespace.ValueString()
	topic := plan.Topic.ValueString()

	if plan.Partitioned.ValueBool() {
		if plan.NumPartitions.IsNull() || plan.NumPartitions.IsUnknown() || plan.NumPartitions.ValueInt64() < 1 {
			resp.Diagnostics.AddError(
				"Error creating topic",
				"Field num_partitions must be > 0 for partitioned topics",
			)
		}
	}

	pulsarClient := r.clients.pulsarAdminClient

	astraOrgID, err := getCurrentOrgID(ctx, r.clients.astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating topic",
			"Could not get current Astra organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, r.clients.astraStreamingClient, r.clients.token, astraOrgID, cluster, tenant)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating topic",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	streamingRequestHeaders := setPulsarClusterHeaders("", cluster, pulsarToken)

	if plan.Persistent.ValueBool() {
		if plan.Partitioned.ValueBool() {
			topicParams := pulsaradmin.PersistentTopicsCreatePartitionedTopicParams{}
			topicRequestBody := strings.NewReader(strconv.FormatInt(plan.NumPartitions.ValueInt64(), 10))
			respHTTP, err := pulsarClient.PersistentTopicsCreatePartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(respHTTP, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.PersistentTopicsCreateNonPartitionedTopicParams{}
			topicRequestBody := strings.NewReader("")
			respHTTP, err := pulsarClient.PersistentTopicsCreateNonPartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(respHTTP, err, "failed to create topic")...)
		}
	} else {
		if plan.Partitioned.ValueBool() {
			topicParams := pulsaradmin.NonPersistentTopicsCreatePartitionedTopicParams{}
			topicRequestBody := strings.NewReader(strconv.FormatInt(plan.NumPartitions.ValueInt64(), 10))
			respHTTP, err := pulsarClient.NonPersistentTopicsCreatePartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(respHTTP, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.NonPersistentTopicsCreateNonPartitionedTopicParams{}
			topicRequestBody := strings.NewReader("")
			respHTTP, err := pulsarClient.NonPersistentTopicsCreateNonPartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(respHTTP, err, "failed to create topic")...)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Schema != nil {
		params := pulsaradmin.SchemasResourcePostSchemaParams{}
		schemaBody := pulsaradmin.SchemasResourcePostSchemaJSONRequestBody{
			Type:       plan.Schema.Type,
			Schema:     plan.Schema.Schema,
			Properties: plan.Schema.Properties,
		}
		respHTTP, err := pulsarClient.SchemasResourcePostSchema(ctx, tenant, namespace, topic, &params, schemaBody, streamingRequestHeaders)
		resp.Diagnostics.Append(HTTPResponseDiagWarn(respHTTP, err, "Failed to update topic schema")...)
	}

	plan.TopicFQN = types.StringValue(plan.getTopicFQN())
	plan.ID = types.StringValue(plan.generateStreamingTopicID())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *StreamingTopicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StreamingTopicResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cluster := getPulsarCluster(state.Cluster.ValueString(), state.CloudProvider.ValueString(), state.Region.ValueString(), r.clients.streamingClusterSuffix)
	state.Cluster = types.StringValue(cluster)

	tenant := state.Tenant.ValueString()
	if tenant == "" {
		tenant = state.TenantName.ValueString()
	}
	namespace := state.Namespace.ValueString()
	topic := state.Topic.ValueString()

	pulsarClient := r.clients.pulsarAdminClient

	astraOrgID, err := getCurrentOrgID(ctx, r.clients.astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topic",
			"Could not get current Astra organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, r.clients.astraStreamingClient, r.clients.token, astraOrgID, cluster, tenant)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topic",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	// Default to persistent true and partitioned false for compatibility with older provider versions
	if state.Persistent.IsNull() {
		state.Persistent = types.BoolValue(true)
	}
	if state.Partitioned.IsNull() {
		state.Partitioned = types.BoolValue(false)
	}

	streamingRequestHeaders := setPulsarClusterHeaders("", cluster, pulsarToken)

	if state.Persistent.ValueBool() {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.PersistentTopicsGetPartitionedMetadataParams{}
			topicResp, err := pulsarClient.PersistentTopicsGetPartitionedMetadataWithResponse(ctx, tenant, namespace, topic, &topicParams, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(topicResp.HTTPResponse, err, "failed to read topic")...)
			if err == nil {
				state.NumPartitions = types.Int64PointerValue(int32ToInt64Pointer(topicResp.JSON200.Partitions))
			}
		} else {
			topicParams := pulsaradmin.PersistentTopicsGetStatsParams{}
			respHTTP, err := pulsarClient.PersistentTopicsGetStats(ctx, tenant, namespace, topic, &topicParams, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(respHTTP, err, "failed to read topic")...)
		}
	} else {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.NonPersistentTopicsGetPartitionedMetadataParams{}
			topicResp, err := pulsarClient.NonPersistentTopicsGetPartitionedMetadataWithResponse(ctx, tenant, namespace, topic, &topicParams, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(topicResp.HTTPResponse, err, "failed to read topic")...)
			if err != nil {
				// TODO: need to fix the pulsaradmin API to decode this response for non-partitioned topics
				partitionMeta := &pulsaradmin.PartitionedTopicMetadata{}
				err := json.NewDecoder(topicResp.HTTPResponse.Body).Decode(partitionMeta)
				if err != nil {
					resp.Diagnostics.AddError(
						"Failed to read partition metadata",
						err.Error(),
					)
				} else {
					state.NumPartitions = types.Int64PointerValue(int32ToInt64Pointer(partitionMeta.Partitions))
				}
			}
		} else {
			topicParams := pulsaradmin.NonPersistentTopicsGetStatsParams{}
			respHTTP, err := pulsarClient.NonPersistentTopicsGetStats(ctx, tenant, namespace, topic, &topicParams, streamingRequestHeaders)
			resp.Diagnostics.Append(HTTPResponseDiagErr(respHTTP, err, "failed to read topic")...)
		}
	}

	//schema := &StreamingTopicSchema{}
	params := pulsaradmin.SchemasResourceGetSchemaParams{}
	schemaResp, err := r.clients.pulsarAdminClient.SchemasResourceGetSchemaWithResponse(ctx, tenant, namespace, topic,
		&params, streamingRequestHeaders)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get topic schema",
			err.Error(),
		)
		return
	} else if schemaResp.StatusCode() > 299 && schemaResp.StatusCode() != 404 {
		resp.Diagnostics.Append(HTTPResponseDiagWarn(schemaResp.HTTPResponse, err, "Failed to get topic schema")...)
	} else if schemaResp.JSON200 != nil {
		state.Schema = &StreamingTopicSchema{
			Type:       (*string)(schemaResp.JSON200.Type),
			Schema:     schemaResp.JSON200.Data,
			Properties: schemaResp.JSON200.Properties,
		}
	}

	state.TopicFQN = types.StringValue(state.getTopicFQN())
	state.ID = types.StringValue(state.generateStreamingTopicID())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StreamingTopicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StreamingTopicResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if plan.Cluster.ValueString() == "" {
		cluster := getPulsarCluster(plan.Cluster.ValueString(), plan.CloudProvider.ValueString(), plan.Region.ValueString(), r.clients.streamingClusterSuffix)
		plan.Cluster = types.StringValue(cluster)
	}
	if plan.Tenant.ValueString() == "" {
		plan.Tenant = plan.TenantName
	}

	plan.TopicFQN = types.StringValue(plan.getTopicFQN())
	plan.ID = types.StringValue(plan.generateStreamingTopicID())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *StreamingTopicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StreamingTopicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Attempted to delete protected topic: "+state.ID.ValueString(),
			"Topic field 'deletion_protection' must be set to 'false' to allow this topic to be deleted",
		)
		return

	}
	cluster := getPulsarCluster(state.Cluster.ValueString(), state.CloudProvider.ValueString(), state.Region.ValueString(), r.clients.streamingClusterSuffix)
	tenant := state.Tenant.ValueString()
	if tenant == "" {
		tenant = state.TenantName.ValueString()
	}
	namespace := state.Namespace.ValueString()
	topic := state.Topic.ValueString()

	pulsarClient := r.clients.pulsarAdminClient

	astraOrgID, err := getCurrentOrgID(ctx, r.clients.astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting topic",
			"Could not get current Astra organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, r.clients.astraStreamingClient, r.clients.token, astraOrgID, cluster, tenant)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting topic",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	pulsarRequestEditor := setPulsarClusterHeaders("", cluster, pulsarToken)

	if state.Persistent.ValueBool() {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.PersistentTopicsDeletePartitionedTopicParams{}
			httpResp, err := pulsarClient.PersistentTopicsDeletePartitionedTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			resp.Diagnostics.Append(HTTPResponseDiagErr(httpResp, err, "failed to delete topic")...)
		} else {
			topicParams := pulsaradmin.PersistentTopicsDeleteTopicParams{}
			httpResp, err := pulsarClient.PersistentTopicsDeleteTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			resp.Diagnostics.Append(HTTPResponseDiagErr(httpResp, err, "failed to delete topic")...)
		}
	} else {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.NonPersistentTopicsDeletePartitionedTopicParams{}
			httpResp, err := pulsarClient.NonPersistentTopicsDeletePartitionedTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			resp.Diagnostics.Append(HTTPResponseDiagErr(httpResp, err, "failed to delete topic")...)
		} else {
			topicParams := pulsaradmin.NonPersistentTopicsDeleteTopicParams{}
			httpResp, err := pulsarClient.NonPersistentTopicsDeleteTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			resp.Diagnostics.Append(HTTPResponseDiagErr(httpResp, err, "failed to delete topic")...)
		}
	}

}

func (r *StreamingTopicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	topic, err := parseStreamingTopicID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to import topic, ID must be in the format <cluster>:<persistence>://<tenant>/<namespace>/<topic>(-partition)",
			err.Error(),
		)
		return
	}
	provider, region, err := getProviderRegionFromClusterName(topic.Cluster.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Invalid cluster name: %s", topic.Cluster.ValueString()),
			err.Error(),
		)
		return
	}
	topic.CloudProvider = types.StringValue(provider)
	topic.Region = types.StringValue(region)
	topic.DeletionProtection = types.BoolValue(true)
	// var diags diag.Diagnostics
	// topic.Schema, diags = types.ObjectValue(StreamingTopicSchemaAttributeTypes, nil)
	// resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, topic)...)
}

// setStreamingTopicID formats the ID string based on the fields (cluster, tenant, etc) in the model
func (t *StreamingTopicResourceModel) getTopicFQN() string {
	persistence := "non-persistent"
	if t.Persistent.ValueBool() {
		persistence = "persistent"
	}
	tenant := t.Tenant.ValueString()
	if tenant == "" {
		tenant = t.TenantName.ValueString()
	}
	return fmt.Sprintf("%s://%s/%s/%s", persistence,
		tenant, t.Namespace.ValueString(), t.Topic.ValueString())
}

// setStreamingTopicID formats the ID string based on the fields (cluster, tenant, etc) in the model
func (t *StreamingTopicResourceModel) generateStreamingTopicID() string {
	partitioned := ""
	if t.Partitioned.ValueBool() {
		partitioned = "-partition"
	}
	topicFqn := t.getTopicFQN()
	return fmt.Sprintf("%s:%s%s", t.Cluster.ValueString(), topicFqn, partitioned)
}

var (
	streamingTopicIDRegexStr = `([a-z][a-z0-9-]*):(persistent|non-persistent)://` +
		`([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)`
	streamingTopicIDRegex = regexp.MustCompile(streamingTopicIDRegexStr)
)

func parseStreamingTopicID(id string) (*StreamingTopicResourceModel, error) {
	model := &StreamingTopicResourceModel{}
	parts := streamingTopicIDRegex.FindStringSubmatch(id)
	if len(parts) != 6 {
		return nil, fmt.Errorf("failed to parse streaming topic ID, does not match expected pattern")
	}
	model.Cluster = types.StringValue(parts[1])
	if parts[2] == "persistent" {
		model.Persistent = types.BoolValue(true)
	} else {
		model.Persistent = types.BoolValue(false)
	}
	model.Tenant = types.StringValue(parts[3])
	model.Namespace = types.StringValue(parts[4])
	topicName := parts[5]
	if strings.HasSuffix(topicName, "-partition") {
		topicName = strings.TrimSuffix(topicName, "-partition")
		model.Partitioned = types.BoolValue(true)
	} else {
		model.Partitioned = types.BoolValue(false)
	}
	model.Topic = types.StringValue(topicName)

	return model, nil
}
