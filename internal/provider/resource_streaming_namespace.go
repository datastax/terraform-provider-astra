package provider

import (
	"context"
	"fmt"
	"strings"

	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/datastax/pulsar-admin-client-go/src/pulsaradmin"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &StreamingNamespaceResource{}
	_ resource.ResourceWithConfigure   = &StreamingNamespaceResource{}
	_ resource.ResourceWithImportState = &StreamingNamespaceResource{}
)

// NewStreamingNamespaceResource is a helper function to simplify the provider implementation.
func NewStreamingNamespaceResource() resource.Resource {
	return &StreamingNamespaceResource{}
}

// StreamingNamespaceResource is the resource implementation.
type StreamingNamespaceResource struct {
	clients *astraClients2
}

// StreamingNamespaceResourceModel maps the resource schema data.
type StreamingNamespaceResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Cluster   types.String `tfsdk:"cluster"`
	Tenant    types.String `tfsdk:"tenant"`
	Namespace types.String `tfsdk:"namespace"`
	Policies  types.Object `tfsdk:"policies"`
}

// Metadata returns the data source type name.
func (r *StreamingNamespaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streaming_namespace"
}

// Schema defines the schema for the data source.
func (r *StreamingNamespaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Pulsar Namespace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Full path to the namespace",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
			"policies": pulsarNamespacePoliciesSchema,
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *StreamingNamespaceResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

// Create the resource and sets the initial Terraform state.
func (r *StreamingNamespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := StreamingNamespaceResourceModel{}
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Manually set the ID because this is not directly managed by the user or the server when creating a new namespace
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Cluster.ValueString(), plan.Tenant.ValueString(), plan.Namespace.ValueString()))

	pulsarRequestEditor := setPulsarClusterHeaders(plan.Cluster.ValueString())

	// We have to create the namespace with an empty policy because the Astra Streaming control plane will override any
	// policy that we send.  Then later we adjust any policy fields that have been set by the user.
	pulsarResp, err := r.clients.pulsarAdminClient.NamespacesCreateNamespace(ctx, plan.Tenant.ValueString(), plan.Namespace.ValueString(),
		pulsaradmin.Policies{}, pulsarRequestEditor)
	resp.Diagnostics.Append(HTTPResponseDiagErr(pulsarResp, err, fmt.Sprintf("Error creating namespace %s", plan.Tenant.ValueString()+"/"+plan.Namespace.ValueString()))...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(setNamespacePolicies(ctx, r.clients.pulsarAdminClient, plan, pulsarRequestEditor)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// After creating the namespace, we have to get the policies from the server because some of them are set automatically to default values
	policiesFromServer, diags := getPulsarNamespacePolicies(ctx, r.clients.pulsarAdminClient, plan, pulsarRequestEditor)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mergedPolicies, diags := MergeTerraformObjects(plan.Policies, policiesFromServer, plan.Policies.AttributeTypes(context.Background()))
	resp.Diagnostics.Append(diags...)
	plan.Policies = mergedPolicies

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read the resource state from the remote resource and update the local Terraform state.
func (r *StreamingNamespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := StreamingNamespaceResourceModel{}
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	pulsarClient := r.clients.pulsarAdminClient

	pulsarRequestEditor := setPulsarClusterHeaders(state.Cluster.ValueString())
	policiesFromServer, diags := getPulsarNamespacePolicies(ctx, pulsarClient, state, pulsarRequestEditor)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Policies = policiesFromServer
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update the remote resource and sets the updated Terraform state on success.
func (r *StreamingNamespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StreamingNamespaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Manually set the ID because this is not directly managed by the user or the server when creating a new namespace
	plan.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", plan.Cluster.ValueString(), plan.Tenant.ValueString(), plan.Namespace.ValueString()))

	pulsarRequestEditor := setPulsarClusterHeaders(plan.Cluster.ValueString())
	resp.Diagnostics.Append(setNamespacePolicies(ctx, r.clients.pulsarAdminClient, plan, pulsarRequestEditor)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// After creating the namespace, we have to get the policies from the server because some of them are set automatically to default values
	policiesFromServer, diags := getPulsarNamespacePolicies(ctx, r.clients.pulsarAdminClient, plan, pulsarRequestEditor)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mergedPolicies, diags := MergeTerraformObjects(plan.Policies, policiesFromServer, plan.Policies.AttributeTypes(context.Background()))
	resp.Diagnostics.Append(diags...)
	plan.Policies = mergedPolicies

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *StreamingNamespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StreamingNamespaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	streamingClient := r.clients.astraStreamingClient

	pulsarRequestEditor := setPulsarClusterHeaders(state.Cluster.ValueString())
	params := astrastreaming.DeleteNamespaceParams{}
	_, err := streamingClient.DeleteNamespace(ctx, state.Tenant.ValueString(), state.Namespace.ValueString(), &params, pulsarRequestEditor)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting streaming namespace '%s/%s'", state.Tenant.ValueString(), state.Namespace.ValueString()),
			"Failed to delete streaming namespace: "+err.Error(),
		)
		return
	}
}

// ImportState just reads the ID from the CLI and then calls Read() to get the state of the object
func (r *StreamingNamespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
