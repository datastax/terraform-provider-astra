package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	astrastreaming "github.com/datastax/astra-client-go/v2/astra-streaming"
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
	_ resource.Resource                = &StreamingTenantResource{}
	_ resource.ResourceWithConfigure   = &StreamingTenantResource{}
	_ resource.ResourceWithImportState = &StreamingTenantResource{}
)

// NewStreamingTenantResource is a helper function to simplify the provider implementation.
func NewStreamingTenantResource() resource.Resource {
	return &StreamingTenantResource{}
}

// StreamingTenantResource is the resource implementation.
type StreamingTenantResource struct {
	clients *astraClients2
}

// StreamingTenantResourceModel maps the resource schema data.
type StreamingTenantResourceModel struct {
	ID                     types.String `tfsdk:"id"` // Unique ID in the form cluster_name/tenant_name
	ClusterName            types.String `tfsdk:"cluster_name"`
	CloudProvider          types.String `tfsdk:"cloud_provider"`
	Region                 types.String `tfsdk:"region"`
	TenantName             types.String `tfsdk:"tenant_name"`
	UserEmail              types.String `tfsdk:"user_email"`
	DeletionProtection     types.Bool   `tfsdk:"deletion_protection"`
	BrokerServiceURL       types.String `tfsdk:"broker_service_url"`
	WebServiceURL          types.String `tfsdk:"web_service_url"`
	WebSocketURL           types.String `tfsdk:"web_socket_url"`
	WebsocketQueryParamURL types.String `tfsdk:"web_socket_query_param_url"`
	UserMetricsURL         types.String `tfsdk:"user_metrics_url"`
	TenantID               types.String `tfsdk:"tenant_id"` // GUID assigned by the Astra backend
	Topic                  types.String `tfsdk:"topic"`
}

// Metadata returns the data source type name.
func (r *StreamingTenantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streaming_tenant"
}

// Schema defines the schema for the data source.
func (r *StreamingTenantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Astra Streaming Tenant",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "ID used by Terraform to identify the tenant.  In the form <cluster_name>/<tenant_name>",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Description: "Pulsar cluster name. Required if `cloud_provider` and `region` are not specified.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					requiresReplaceIfClusterChange(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				DeprecationMessage: "'cluster_name' should be used instead of 'cloud_provider' and 'region'.",
				Description:        "Cloud provider, one of `aws`, `gcp`, or `azure`. Required if `cluster_name` is not set.",
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"region": schema.StringAttribute{
				DeprecationMessage: "'cluster_name' should be used instead of 'cloud_provider' and 'region'.",
				Description:        "Cloud provider region. Required if `cluster_name` is not set.",
				Optional:           true,
				Computed:           true,
				PlanModifiers: []planmodifier.String{
					//removeDashesModifier{},
					//suppressDashesDiffModifier{},
					requiresReplaceIfRegionChange(),
				},
			},
			"tenant_name": schema.StringAttribute{
				Description: "Name of the Astra Streaming tenant.  Similar to a Pulsar tenant.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])$"),
					"name must be atleast 2 characters and contain only alphanumeric characters")},
			},
			"user_email": schema.StringAttribute{
				Description: "Email address of the owner of the tenant.",
				Required:    true,
			},
			"topic": schema.StringAttribute{
				Description:        "Streaming tenant topic. Use the `astra_streaming_topic` resource instead.",
				Optional:           true,
				DeprecationMessage: "This field is deprecated and will be removed in a future release. Please use the `astra_streaming_topic` resource instead.",
				Validators: []validator.String{stringvalidator.RegexMatches(regexp.MustCompile("^.{2,}"),
					"name must be atleast 2 characters")},
			},
			"deletion_protection": schema.BoolAttribute{
				Description: "Whether or not to allow Terraform to destroy this tenant. Unless this field is set to false in Terraform state, a `terraform destroy` or `terraform apply` command that deletes the instance will fail. Defaults to `true`.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			// The fields below are only filled in after creating the tenant and retrieving the tenant info from the DevOps API
			"broker_service_url": schema.StringAttribute{
				Description: "The Pulsar Binary Protocol URL used for production and consumption of messages.",
				Computed:    true,
			},
			"web_service_url": schema.StringAttribute{
				Description: "URL used for administrative operations.",
				Computed:    true,
			},
			"web_socket_url": schema.StringAttribute{
				Description: "URL used for web socket operations.",
				Computed:    true,
			},
			"web_socket_query_param_url": schema.StringAttribute{
				Description: "URL used for web socket query parameter operations.",
				Computed:    true,
			},
			"user_metrics_url": schema.StringAttribute{
				Description: "URL for metrics.",
				Computed:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "UUID for the tenant.",
				Computed:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (r *StreamingTenantResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.clients = req.ProviderData.(*astraClients2)
}

type StreamingClusters []struct {
	ID                     string `json:"id"`
	TenantName             string `json:"tenantName"`
	ClusterName            string `json:"clusterName"`
	WebServiceURL          string `json:"webServiceUrl"`
	BrokerServiceURL       string `json:"brokerServiceUrl"`
	WebsocketURL           string `json:"websocketUrl"`
	WebsocketQueryParamURL string `json:"websocketQueryParamUrl"`
	PulsarToken            string `json:"pulsarToken"`
	Plan                   string `json:"plan"`
	PlanCode               string `json:"planCode"`
	AstraOrgGUID           string `json:"astraOrgGUID"`
	CloudProvider          string `json:"cloudProvider"`
	CloudProviderCode      string `json:"cloudProviderCode"`
	CloudRegion            string `json:"cloudRegion"`
	Status                 string `json:"status"`
	JvmVersion             string `json:"jvmVersion"`
	PulsarVersion          string `json:"pulsarVersion"`
	Email                  string `json:"Email"`
	UserMetricsUrl         string `json:"userMetricsUrl"`
	PulsarInstance         string `json:"pulsarInstance"`
	PulsarClusterDNS       string `json:"pulsarClusterDNS"`
	ClusterType            string `json:"clusterType"`
	AzType                 string `json:"azType"`
}

func (r *StreamingTenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := &StreamingTenantResourceModel{}
	diags := req.Plan.Get(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	normalizedRegion := removeDashes(plan.Region.ValueString())

	if plan.ClusterName.ValueString() == "" && (plan.CloudProvider.ValueString() == "" || plan.Region.ValueString() == "") {
		resp.Diagnostics.AddError(
			"missing required configuration",
			"cluster_name or (cloud_provider and region) must be specified")
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

	tenantRequest := astrastreaming.IdOfCreateTenantEndpointJSONRequestBody{
		OrgID:         &orgID,
		OrgName:       &orgID,
		ClusterName:   plan.ClusterName.ValueStringPointer(),
		CloudProvider: plan.CloudProvider.ValueStringPointer(),
		CloudRegion:   &normalizedRegion,
		TenantName:    plan.TenantName.ValueStringPointer(),
		UserEmail:     plan.UserEmail.ValueStringPointer(),
	}

	postParams := astrastreaming.IdOfCreateTenantEndpointParams{
		Topic: plan.Topic.ValueStringPointer(),
	}

	tenantCreateResponse, err := astraStreamingClient.IdOfCreateTenantEndpointWithResponse(ctx, &postParams, tenantRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create tenant",
			err.Error())
		return
	} else if tenantCreateResponse.StatusCode() != http.StatusOK {
		errString := fmt.Sprintf("failed to create tenant '%s' with status code '%v', message: '%s'",
			plan.TenantName.ValueString(), tenantCreateResponse.StatusCode(), string(tenantCreateResponse.Body))
		resp.Diagnostics.AddError("failed to create tenant", errString)
		return
	}

	if plan.ClusterName.IsNull() || plan.ClusterName.IsUnknown() {
		plan.ClusterName = types.StringPointerValue(tenantCreateResponse.JSON200.ClusterName)
	}
	getParams := &astrastreaming.GetStreamingTenantParams{
		XDataStaxPulsarCluster: plan.ClusterName.ValueString(),
	}
	// Now fetch the tenant again so that it fills in the missing fields (like userMetricsUrl and tenant ID)
	tenantGetResponse, err := astraStreamingClient.GetStreamingTenantWithResponse(ctx, plan.TenantName.ValueString(), getParams)
	if err != nil {
		resp.Diagnostics.AddError("failed to get data for tenant "+plan.TenantName.ValueString(), err.Error())
		return
	} else if tenantGetResponse.StatusCode() != http.StatusOK {
		errDetail := fmt.Sprintf("failed to get tenant data for tenant '%s', received response code '%v' with error message: %v",
			plan.TenantName.ValueString(), tenantGetResponse.StatusCode(), string(tenantGetResponse.Body))
		resp.Diagnostics.AddError("failed to get data for tenant", errDetail)
		return
	}

	setStreamingTenantData(plan, tenantGetResponse.JSON200)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)

}

func (r *StreamingTenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {

	state := &StreamingTenantResourceModel{}
	diags := req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	astraStreamingClient := r.clients.astraStreamingClient

	params := &astrastreaming.GetStreamingTenantParams{
		XDataStaxPulsarCluster: state.ClusterName.ValueString(),
	}
	getTenantResponse, err := astraStreamingClient.GetStreamingTenantWithResponse(ctx, state.TenantName.ValueString(), params)
	if err != nil {
		resp.Diagnostics.AddError("failed to get tenant org ID", err.Error())
		return
	} else if getTenantResponse.HTTPResponse.StatusCode == 404 {
		// Tenant not found, remove it from the state
		resp.State.RemoveResource(ctx)
		return
	} else if getTenantResponse.HTTPResponse.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("received invalid status code from tenant request '%v' with error message: %v",
			getTenantResponse.HTTPResponse.StatusCode, string(getTenantResponse.Body))
		resp.Diagnostics.AddError("failed to get tenant data", errMsg)
		return
	}

	setStreamingTenantData(state, getTenantResponse.JSON200)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)

}

func (r *StreamingTenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	plan := &StreamingTenantResourceModel{}
	diags := req.Plan.Get(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := &StreamingTenantResourceModel{}
	diags = req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DeletionProtection = plan.DeletionProtection
	state.UserEmail = plan.UserEmail
	state.Topic = plan.Topic
	if !plan.Region.IsNull() && !plan.Region.IsUnknown() {
		state.Region = plan.Region
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)

}

func (r *StreamingTenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := &StreamingTenantResourceModel{}
	diags := req.State.Get(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("failed to delete streaming tenant", "'deletion_protection' must be explicitly set to 'false' in order to destroy astra_streaming_tenant")
		return
	}

	astraStreamingClient := r.clients.astraStreamingClient

	clusterName := state.ClusterName.ValueString()
	tenantName := state.TenantName.ValueString()
	params := astrastreaming.DeleteStreamingTenantParams{}

	deleteResponse, err := astraStreamingClient.DeleteStreamingTenantWithResponse(ctx, tenantName, clusterName, &params)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete streaming tenant", "error: "+err.Error())
		return
	} else if deleteResponse.HTTPResponse.StatusCode >= 300 || deleteResponse.HTTPResponse.StatusCode < 200 {
		errMsg := fmt.Sprintf("received error code '%v' with message: %v", deleteResponse.HTTPResponse.StatusCode, string(deleteResponse.Body))
		resp.Diagnostics.AddError("failed to delete streaming tenant", errMsg)
		return
	}

}

// ImportState just reads the ID from the CLI and then calls Read() to get the state of the object
func (r *StreamingTenantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tenantID := strings.Split(req.ID, "/")
	if len(tenantID) != 2 {
		resp.Diagnostics.AddError(
			"Error importing streaming tenant",
			"ID must be in the format <cluster>/<tenant>",
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), tenantID[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("tenant_name"), tenantID[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("deletion_protection"), true)...)
}

func setStreamingTenantData(data *StreamingTenantResourceModel, tenantResponse *astrastreaming.TenantClusterPlanResponse) {
	data.ClusterName = types.StringPointerValue(tenantResponse.ClusterName)
	data.CloudProvider = types.StringPointerValue(tenantResponse.CloudProvider)
	if data.Region.IsNull() || data.Region.ValueString() == "" {
		// The region returned from a streaming request will have the dashes removed,
		// so we only set it if it's currently empty.
		data.Region = types.StringPointerValue(tenantResponse.CloudProviderRegion)
	}
	data.TenantID = types.StringPointerValue(tenantResponse.Id)
	data.TenantName = types.StringPointerValue(tenantResponse.TenantName)

	data.BrokerServiceURL = types.StringPointerValue(tenantResponse.PulsarURL)
	data.WebServiceURL = types.StringPointerValue(tenantResponse.AdminURL)
	data.WebSocketURL = types.StringPointerValue(tenantResponse.WebsocketURL)
	data.WebsocketQueryParamURL = types.StringPointerValue(tenantResponse.WebsocketQueryParamURL)
	data.UserMetricsURL = types.StringPointerValue(tenantResponse.UserMetricsURL)
	data.ID = types.StringValue(data.ClusterName.ValueString() + "/" + data.TenantName.ValueString())

}

// requiresReplaceIfClusterChange only require replace if the cluster name, cloud provider, or region has changed.
func requiresReplaceIfClusterChange() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			state := &StreamingTenantResourceModel{}
			diags := req.State.Get(ctx, state)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			plan := &StreamingTenantResourceModel{}
			diags = req.Plan.Get(ctx, plan)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if state.ClusterName.ValueString() != plan.ClusterName.ValueString() && !plan.ClusterName.IsUnknown() {
				resp.RequiresReplace = true
				return
			}

		},
		"If the value of this attribute changes, Terraform will destroy and recreate the resource.",
		"If the value of this attribute changes, Terraform will destroy and recreate the resource.",
	)
}

// requiresReplaceIfRegionChange only require replace if the cluster name, cloud provider, or region has changed.
func requiresReplaceIfRegionChange() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			state := &StreamingTenantResourceModel{}
			diags := req.State.Get(ctx, state)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			plan := &StreamingTenantResourceModel{}
			diags = req.Plan.Get(ctx, plan)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if removeDashes(state.Region.ValueString()) != removeDashes(plan.Region.ValueString()) && plan.Region.ValueString() != "" {
				resp.RequiresReplace = true
				return
			}

		},
		"If the value of this attribute changes, Terraform will destroy and recreate the resource.",
		"If the value of this attribute changes, Terraform will destroy and recreate the resource.",
	)
}
