package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &streamingNamespaceResource{}
	_ resource.ResourceWithConfigure   = &streamingNamespaceResource{}
	_ resource.ResourceWithImportState = &streamingNamespaceResource{}
)

// NewStreamingNamespaceResource is a helper function to simplify the provider implementation.
func NewStreamingNamespaceResource() resource.Resource {
	return &streamingNamespaceResource{}
}

// streamingNamespaceResource is the resource implementation.
type streamingNamespaceResource struct {
	clients *astraClients2
}

// streamingNamespaceResourceModel maps the resource schema data.
type streamingNamespaceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Cluster   types.String `tfsdk:"cluster"`
	Tenant    types.String `tfsdk:"tenant"`
	Namespace types.String `tfsdk:"namespace"`
}

// Metadata returns the data source type name.
func (r *streamingNamespaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streaming_namespace"
}

// Schema defines the schema for the data source.
func (r *streamingNamespaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Pulsar Namespace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Full path to the namespace",
				Computed:    true,
			},
			"cluster": schema.StringAttribute{
				Description: "Cluster where the tenant is located.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tenant": schema.StringAttribute{
				Description: "Name of the tenant.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace": schema.StringAttribute{
				Description: "Name of the Pulsar namespace.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *streamingNamespaceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

// Create creates the resource and sets the initial Terraform state.
func (r *streamingNamespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan streamingNamespaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	streamingClient := r.clients.astraStreamingClient
	streamingV3Client := r.clients.astraStreamingClientv3

	orgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			"Could not get current organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, streamingClient, r.clients.token, orgID, plan.Cluster.ValueString(), plan.Tenant.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	pulsarRequestEditor := setPulsarClusterHeaders(pulsarToken, plan.Cluster.ValueString(), "")
	streamingNamespaceResp, err := streamingV3Client.CreateNamespaceWithResponse(ctx, plan.Tenant.ValueString(), plan.Namespace.ValueString(), pulsarRequestEditor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			"Could not create Pulsar namespace: "+err.Error(),
		)
		return
	} else if streamingNamespaceResp.StatusCode() >= 300 {
		resp.Diagnostics.AddError(
			"Error creating namespace",
			fmt.Sprintf("Could not create Pulsar namespace, status '%s', body: %s", streamingNamespaceResp.Status(), string(streamingNamespaceResp.Body)),
		)
		return
	}

	// Manually set the ID because this is computed
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Cluster, plan.Tenant, plan.Namespace))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *streamingNamespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state streamingNamespaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	streamingClient := r.clients.astraStreamingClient
	streamingV3Client := r.clients.astraStreamingClientv3

	orgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting namespace",
			"Could not get current organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, streamingClient, r.clients.token, orgID, state.Cluster.ValueString(), state.Tenant.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting namespace",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	pulsarRequestEditor := setPulsarClusterHeaders(pulsarToken, state.Cluster.ValueString(), orgID)
	streamingNamespaceResp, err := streamingV3Client.GetNamespaceWithResponse(ctx, state.Tenant.ValueString(), state.Namespace.ValueString(), pulsarRequestEditor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting namespace",
			"Could not get Pulsar namespace: "+err.Error(),
		)
		return
	} else if streamingNamespaceResp.StatusCode() >= 300 {
		resp.Diagnostics.AddError(
			"Error getting namespace",
			fmt.Sprintf("Could not get Pulsar namespace status: %v", streamingNamespaceResp.Status()),
		)
		return
	}
	var pulsarNamespacePolicies map[string]interface{}
	err = json.Unmarshal(streamingNamespaceResp.Body, &pulsarNamespacePolicies)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting namespace",
			"Could not unmarshal Pulsar namespace: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *streamingNamespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not implemented

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *streamingNamespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state streamingNamespaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	streamingClient := r.clients.astraStreamingClient
	streamingV3Client := r.clients.astraStreamingClientv3

	orgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting namespace",
			"Could not get current organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getLatestPulsarToken(ctx, streamingClient, r.clients.token, orgID, state.Cluster.ValueString(), state.Tenant.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting namespace",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	pulsarRequestEditor := setPulsarClusterHeaders(pulsarToken, state.Cluster.ValueString(), "")
	params := astrastreaming.DeleteNamespaceParams{}
	_, err = streamingV3Client.DeleteNamespace(ctx, state.Tenant.ValueString(), state.Namespace.ValueString(), &params, pulsarRequestEditor)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting namespace",
			"Could not create Pulsar namespace: "+err.Error(),
		)
		return
	}
}

func (r *streamingNamespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	namespaceID := strings.Split(req.ID, "/")
	if len(namespaceID) != 3 {
		resp.Diagnostics.AddError(
			"Error importing namespace",
			"ID must be in the format <cluster>/<tenant>/<namespace>",
		)
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster"), namespaceID[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tenant"), namespaceID[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("namespace"), namespaceID[2])...)
}
