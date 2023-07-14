package provider

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	adminPulsarTokenType = "admin"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &StreamingPulsarTokenResource{}
	_ resource.ResourceWithConfigure   = &StreamingPulsarTokenResource{}
	_ resource.ResourceWithImportState = &StreamingPulsarTokenResource{}
)

// NewStreamingPulsarTokenResource is a helper function to simplify the provider implementation.
func NewStreamingPulsarTokenResource() resource.Resource {
	return &StreamingPulsarTokenResource{}
}

// StreamingPulsarTokenResource is the resource implementation.
type StreamingPulsarTokenResource struct {
	clients *astraClients2
}

// StreamingPulsarTokenResourceModel maps the resource schema data.
type StreamingPulsarTokenResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Cluster    types.String `tfsdk:"cluster"`
	Tenant     types.String `tfsdk:"tenant"`
	TimeToLive types.String `tfsdk:"time_to_live"`
	Token      types.String `tfsdk:"token"`
}

// Metadata returns the data source type name.
func (r *StreamingPulsarTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streaming_pulsar_token"
}

// Schema defines the schema for the data source.
func (r *StreamingPulsarTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Pulsar Token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Full path to the namespace",
				Computed:    true,
			},
			"cluster": schema.StringAttribute{
				Description: "Cluster where the Pulsar tenant is located.",
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
			"time_to_live": schema.StringAttribute{
				Description: "The relative time until the token expires.  For example 1h, 1d, 1y, etc.",
				Optional:    true,
				// Default:     stringdefault.StaticString("1y"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Description: "String values of the token",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *StreamingPulsarTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

// Create creates the resource and sets the initial Terraform state.
func (r *StreamingPulsarTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var tokenPlan StreamingPulsarTokenResourceModel
	diags := req.Plan.Get(ctx, &tokenPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	streamingClient := r.clients.astraStreamingClient

	astraOrgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Pulsar token",
			"Could not get current Astra organization: "+err.Error(),
		)
		return
	}

	tokenRequestParams := &astrastreaming.CreateTenantTokenHandlerV3Params{
		XDataStaxCurrentOrg:    astraOrgID,
		XDataStaxPulsarCluster: tokenPlan.Cluster.ValueString(),
	}
	tokenRequestBody := astrastreaming.CreateTenantTokenV3Request{
		Type: &adminPulsarTokenType,
		Exp:  tokenPlan.TimeToLive.ValueStringPointer(),
	}
	tokenHTTPResp, err := streamingClient.CreateTenantTokenHandlerV3(ctx,
		tokenPlan.Tenant.ValueString(), tokenRequestParams, tokenRequestBody)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Pulsar token",
			"Could not get Pulsar token: "+err.Error(),
		)
		return
	} else if tokenHTTPResp.StatusCode >= 300 {
		errorMsg, err := ioutil.ReadAll(tokenHTTPResp.Body)
		if err != nil {
			errorMsg = []byte(err.Error())
		}
		resp.Diagnostics.AddError(
			"Error creating Pulsar token",
			fmt.Sprintf("Received unexpected status code, status '%s', body: %s", tokenHTTPResp.Status, string(errorMsg)),
		)
		return
	}

	pulsarTokenResp, err := astrastreaming.ParseCreateTenantTokenHandlerV3Response(tokenHTTPResp)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Pulsar token",
			fmt.Sprintf("Failed to parse token response, status '%s', body: %s", tokenHTTPResp.Status, err.Error()),
		)
		return
	}

	// Manually set the ID because this is computed
	tokenPlan.ID = types.StringValue(*pulsarTokenResp.JSON201.ID)
	tokenPlan.Token = types.StringValue(*pulsarTokenResp.JSON201.Token)

	resp.Diagnostics.Append(resp.State.Set(ctx, &tokenPlan)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *StreamingPulsarTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StreamingPulsarTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	streamingClient := r.clients.astraStreamingClient

	astraOrgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting pulsar token",
			"Could not get current organization: "+err.Error(),
		)
		return
	}

	pulsarToken, err := getPulsarTokenByID(ctx, streamingClient, astraOrgID, state.Cluster.ValueString(), state.Tenant.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting pulsar token",
			"Could not get pulsar token: "+err.Error(),
		)
		return
	}

	state.Token = types.StringValue(pulsarToken)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *StreamingPulsarTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not implemented

}

// Delete deletes the resource and removes the Terraform state on success.
func (r *StreamingPulsarTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state StreamingPulsarTokenResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraClient := r.clients.astraClient
	streamingClient := r.clients.astraStreamingClient

	astraOrgID, err := getCurrentOrgID(ctx, astraClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting pulsar token",
			"Could not get current organization: "+err.Error(),
		)
		return
	}

	params := astrastreaming.DeletePulsarTokenByIDParams{
		XDataStaxCurrentOrg:    astraOrgID,
		XDataStaxPulsarCluster: state.Cluster.ValueString(),
	}
	httpResp, err := streamingClient.DeletePulsarTokenByID(ctx, state.Tenant.ValueString(), state.ID.ValueString(), &params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting token",
			"Could not create Pulsar token: "+err.Error(),
		)
	} else if httpResp.StatusCode > 300 {
		resp.Diagnostics.AddError(
			"Error deleting token",
			fmt.Sprintf("Unexpected status code: %v", httpResp.StatusCode),
		)
	}
}

func (r *StreamingPulsarTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
