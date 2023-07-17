package provider

import (
	"context"
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
	ID                 types.String `tfsdk:"id" json:"id"`
	Cluster            types.String `tfsdk:"cluster" json:"cluster"`
	CloudProvider      types.String `tfsdk:"cloud_provider" json:"cloud_provider"`
	Region             types.String `tfsdk:"region" json:"region"`
	Tenant             types.String `tfsdk:"tenant" json:"tenant"`
	TenantName         types.String `tfsdk:"tenant_name" json:"tenant_name"`
	Namespace          types.String `tfsdk:"namespace" json:"namespace"`
	Topic              types.String `tfsdk:"topic" json:"topic"`
	Persistent         types.Bool   `tfsdk:"persistent" json:"persistent"`
	Partitioned        types.Bool   `tfsdk:"partitioned" json:"partitioned"`
	NumPartitions      types.Int64  `tfsdk:"num_partitions" json:"num_partitions"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection" json:"deletion_protection"`
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
				Description: "Full path to the namespace",
				Computed:    true,
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
				},
			},
			"tenant": schema.StringAttribute{
				Description: "Name of the streaming tenant.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
					stringplanmodifier.RequiresReplace(),
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
				// Default:     stringdefault.StaticString("1y"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"persistent": schema.BoolAttribute{
				Description: "Persistent or non-persistent topic",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"partitioned": schema.BoolAttribute{
				Description: "Partitioned or non-partitioned topic",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"num_partitions": schema.Int64Attribute{
				Description: "Number of partitions for a partitioned topic.  This field must not be set for a non-partitioned topic.",
				Optional:    true,
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Prevent this topic from being deleted via Terraform",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
	if plan.Cluster.IsNull() || plan.Cluster.IsUnknown() {
		cluster := getPulsarCluster(plan.CloudProvider.ValueString(), plan.Region.ValueString())
		plan.Cluster = types.StringValue(cluster)
	} else {
		provider, region, err := getProviderRegionFromClusterName(plan.Cluster.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Invalid cluster name: %s", plan.Cluster.ValueString()),
				err.Error(),
			)
			return
		}
		plan.CloudProvider = types.StringValue(provider)
		plan.Region = types.StringValue(region)
	}
	cluster := plan.Cluster.ValueString()

	tenant := plan.Tenant.ValueString()
	if plan.Tenant.IsNull() || plan.Tenant.IsUnknown() {
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

	pulsarRequestEditor := setPulsarClusterHeaders("", plan.Cluster.ValueString(), pulsarToken)

	if plan.Persistent.ValueBool() {
		if plan.Partitioned.ValueBool() {
			topicParams := pulsaradmin.PersistentTopicsCreatePartitionedTopicParams{}
			topicRequestBody := strings.NewReader(strconv.FormatInt(plan.NumPartitions.ValueInt64(), 10))
			resp, err := pulsarClient.PersistentTopicsCreatePartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.PersistentTopicsCreateNonPartitionedTopicParams{}
			topicRequestBody := strings.NewReader("")
			resp, err := pulsarClient.PersistentTopicsCreateNonPartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		}
	} else {
		if plan.Partitioned.ValueBool() {
			topicParams := pulsaradmin.NonPersistentTopicsCreatePartitionedTopicParams{}
			topicRequestBody := strings.NewReader(strconv.FormatInt(plan.NumPartitions.ValueInt64(), 10))
			resp, err := pulsarClient.NonPersistentTopicsCreatePartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.NonPersistentTopicsCreateNonPartitionedTopicParams{}
			topicRequestBody := strings.NewReader("")
			resp, err := pulsarClient.NonPersistentTopicsCreateNonPartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		}
	}

	persistence := "non-persistent"
	if plan.Persistent.ValueBool() {
		persistence = "persistent"
	}
	partitioned := ""
	if plan.Partitioned.ValueBool() {
		partitioned = "-partition"
	}

	// Manually set the ID because this is computed
	plan.ID = types.StringValue(fmt.Sprintf("%s:%s://%s/%s/%s%s", cluster, persistence, tenant, namespace, topic, partitioned))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *StreamingTopicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StreamingTopicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cluster := state.Cluster.ValueString()
	tenant := state.Tenant.ValueString()
	namespace := state.Namespace.ValueString()
	topic := state.Topic.ValueString()

	pulsarClient := r.clients.pulsarAdminClient

	astraOrgID, err := getCurrentOrgID(ctx, r.clients.astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Pulsar token",
			"Could not get current Astra organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, r.clients.astraStreamingClient, r.clients.token, astraOrgID, cluster, tenant)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	pulsarRequestEditor := setPulsarClusterHeaders("", state.Cluster.ValueString(), pulsarToken)

	if state.Persistent.ValueBool() {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.PersistentTopicsCreatePartitionedTopicParams{}
			topicRequestBody := strings.NewReader(strconv.FormatInt(state.NumPartitions.ValueInt64(), 10))
			resp, err := pulsarClient.PersistentTopicsCreatePartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.PersistentTopicsCreateNonPartitionedTopicParams{}
			topicRequestBody := strings.NewReader("")
			resp, err := pulsarClient.PersistentTopicsCreateNonPartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		}
	} else {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.NonPersistentTopicsCreatePartitionedTopicParams{}
			topicRequestBody := strings.NewReader(strconv.FormatInt(state.NumPartitions.ValueInt64(), 10))
			resp, err := pulsarClient.NonPersistentTopicsCreatePartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.NonPersistentTopicsCreateNonPartitionedTopicParams{}
			topicRequestBody := strings.NewReader("")
			resp, err := pulsarClient.NonPersistentTopicsCreateNonPartitionedTopicWithBody(ctx, tenant, namespace, topic, &topicParams, "application/json", topicRequestBody, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		}
	}

	persistence := "non-persistent"
	if state.Persistent.ValueBool() {
		persistence = "persistent"
	}
	partitioned := ""
	if state.Partitioned.ValueBool() {
		partitioned = "-partition"
	}

	// Manually set the ID because this is computed
	state.ID = types.StringValue(fmt.Sprintf("%s:%s://%s/%s/%s%s", cluster, persistence, tenant, namespace, topic, partitioned))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *StreamingTopicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not implemented
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *StreamingTopicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StreamingTopicResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cluster := state.Cluster.ValueString()
	tenant := state.Tenant.ValueString()
	namespace := state.Namespace.ValueString()
	topic := state.Topic.ValueString()

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

	pulsarRequestEditor := setPulsarClusterHeaders("", state.Cluster.ValueString(), pulsarToken)

	if state.Persistent.ValueBool() {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.PersistentTopicsDeletePartitionedTopicParams{}
			resp, err := pulsarClient.PersistentTopicsDeletePartitionedTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.PersistentTopicsDeleteTopicParams{}
			resp, err := pulsarClient.PersistentTopicsDeleteTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		}
	} else {
		if state.Partitioned.ValueBool() {
			topicParams := pulsaradmin.NonPersistentTopicsDeletePartitionedTopicParams{}
			resp, err := pulsarClient.NonPersistentTopicsDeletePartitionedTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		} else {
			topicParams := pulsaradmin.NonPersistentTopicsDeleteTopicParams{}
			resp, err := pulsarClient.NonPersistentTopicsDeleteTopic(ctx, tenant, namespace, topic, &topicParams, pulsarRequestEditor)
			diags.Append(HTTPResponseDiagErr(resp, err, "failed to create topic")...)
		}
	}

	// Manually set the ID because this is computed
	state.ID = types.StringNull()

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *StreamingTopicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tokenPath := strings.Split(req.ID, "/")
	if len(tokenPath) != 3 {
		resp.Diagnostics.AddError(
			"Error importing token",
			"ID must be in the format <cluster>/<tenant>/<tokenID>",
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster"), tokenPath[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tenant"), tokenPath[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), tokenPath[2])...)
}

var (
	streamingTopicIDRegexStr = `([a-z][a-z0-9-]*):(persistent|non-persistent)://` +
		`([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)/([a-z][a-z0-9-]*)(-partition(ed)?)?`
	streamingTopicIDRegex = regexp.MustCompile(streamingTopicIDRegexStr)
)

func parseStreamingTopicID(id string) (*StreamingTopicResourceModel, error) {
	model := &StreamingTopicResourceModel{}
	parts := streamingTopicIDRegex.FindStringSubmatch(id)
	if len(parts) < 7 || len(parts) > 9 {
		return nil, fmt.Errorf("failed to parse streaming topic ID")
	}
	model.Cluster = types.StringValue(parts[1])
	if parts[2] == "persistent" {
		model.Persistent = types.BoolValue(true)
	} else {
		model.Persistent = types.BoolValue(false)
	}
	model.Tenant = types.StringValue(parts[3])
	model.Namespace = types.StringValue(parts[4])
	model.Topic = types.StringValue(parts[5])
	if strings.HasPrefix(parts[6], "-partition") {
		model.Partitioned = types.BoolValue(true)
	} else {
		model.Partitioned = types.BoolValue(false)
	}
	return model, nil
}
