package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

var (
	_ resource.Resource                = &pcuGroupAssociationResource{}
	_ resource.ResourceWithConfigure   = &pcuGroupAssociationResource{}
	_ resource.ResourceWithImportState = &pcuGroupAssociationResource{}
)

func NewPcuGroupAssociationResource() resource.Resource {
	return &pcuGroupAssociationResource{}
}

type pcuGroupAssociationResource struct {
	BasePCUResource
}

type pcuGroupAssociationResourceModel struct {
	PCUGroupId         types.String `tfsdk:"pcu_group_id"`
	DeletionProtection types.Bool   `tfsdk:"deletion_protection"`
	PcuGroupAssociationModel
}

func (r *pcuGroupAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pcu_group_association"
}

func (r *pcuGroupAssociationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"pcu_group_id": schema.StringAttribute{
				Required: true,
			},
			"datacenter_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // TODO check we need anything fancier than this
				},
			},
			"deletion_protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(), // TODO is this necessary here?
				},
			},
			"provisioning_status": schema.StringAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
			"created_by": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_by": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// TODO what error is thrown if the association already exists (if any?) (do we need to manually check for this?)
// TODO what if it's already associated with another PCU group which Terraform doesn't know about?
// TODO basically we need to either notify the user if the association already exists without Terraform's knowledge,
// TODO or we need to execute a secret transfer request to move it to the new PCU group (which may be kinda shady)

func (r *pcuGroupAssociationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pcuGroupAssociationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(createPcuGroupAssociation(r.client, ctx, plan.PCUGroupId.ValueString(), plan.DatacenterId.ValueString(), &plan.PcuGroupAssociationModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pcuGroupAssociationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pcuGroupAssociationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	association, status, err := getPcuGroupAssociation(r.client, ctx, data.PCUGroupId.ValueString(), data.DatacenterId.ValueString())

	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading PCU Group Association", "Could not read PCU Group Association: "+err.Error())
		return
	}

	data.PcuGroupAssociationModel = *association
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *pcuGroupAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pcuGroupAssociationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO do I really need this if statement (isn't it guaranteed by the plan modifier?)
	if !state.PCUGroupId.Equal(plan.PCUGroupId) {
		resp.Diagnostics.Append(transferPcuGroupAssociation(r.client, ctx, state.PCUGroupId.ValueString(), plan.PCUGroupId.ValueString(), state.DatacenterId.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// TODO do we need to use awaitPcuAssociationStatus? (I don't think so but can't test right now)
	association, _, err := getPcuGroupAssociation(r.client, ctx, plan.PCUGroupId.ValueString(), plan.DatacenterId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading PCU Group Association after transfer", "Could not read PCU Group Association after transfer: "+err.Error())
		return
	}

	plan.PcuGroupAssociationModel = *association
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pcuGroupAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pcuGroupAssociationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("Error deleting PCU Group Association", "PCU Group Association has deletion protection enabled, cannot delete.")
		return
	}

	resp.Diagnostics.Append(deletePcuGroupAssociation(r.client, ctx, data.PCUGroupId.ValueString(), data.DatacenterId.ValueString())...)
}

func (r *pcuGroupAssociationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "/")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		res.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: <pcu_group_id>/<datacenter_id>. Got: %q", req.ID),
		)
		return
	}

	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("pcu_group_id"), idParts[0])...)
	res.Diagnostics.Append(res.State.SetAttribute(ctx, path.Root("datacenter_id"), idParts[1])...)
}

func createPcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string, out *PcuGroupAssociationModel) diag.Diagnostics {
	var diags diag.Diagnostics

	resp, err := client.PcuAssociationCreate(ctx, pcuGroupId, datacenterId)

	if err != nil {
		diags.AddError("Error creating PCU Group Association", "Could not create PCU Group Association: "+err.Error())
		return diags
	}

	if resp.StatusCode >= 400 {
		diags.AddError("Error creating PCU Group Association", "Could not create PCU Group Association, unexpected status code: "+resp.Status)
		return diags
	}

	association, moreDiags := awaitPcuAssociationStatus(client, ctx, pcuGroupId, datacenterId, astra.PCUAssociationStatusCreated)
	diags.Append(moreDiags...)

	*out = *association
	return diags
}

func getPcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string) (*PcuGroupAssociationModel, int, error) {
	associations, status, err := GetPcuGroupAssociations(client, ctx, pcuGroupId)

	if err != nil {
		return nil, status, err
	}

	for _, assoc := range *associations {
		if assoc.DatacenterId.ValueString() == datacenterId {
			return &assoc, status, nil
		}
	}

	// simulating a 404 just for convenience ¯\_(ツ)_/¯
	// sorry, not sorry
	return nil, 404, fmt.Errorf("PCU group association not found")
}

// TODO: should deletion_protection also stop moving the association?
// TODO: what's returned if the association didn't exist in the first place?
func transferPcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, fromPcuGroupId string, toPcuGroupId, datacenterId string) diag.Diagnostics {
	var diags diag.Diagnostics

	resp, err := client.PcuAssociationTransfer(ctx, astra.PcuAssociationTransferJSONRequestBody{
		FromPCUGroupUUID: &fromPcuGroupId,
		ToPCUGroupUUID:   &toPcuGroupId,
		DatacenterUUID:   &datacenterId,
	})

	if err != nil {
		diags.AddError("Error transferring PCU Group Association", "Could not transfer PCU Group Association: "+err.Error())
		return diags
	}

	if resp.StatusCode >= 400 {
		diags.AddError("Error transferring PCU Group Association", "Could not transfer PCU Group Association, unexpected status code: "+resp.Status)
		return diags
	}

	return diags
}

func deletePcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string) diag.Diagnostics {
	var diags diag.Diagnostics

	resp, err := client.PcuAssociationDelete(ctx, pcuGroupId, datacenterId)

	if err != nil {
		diags.AddError("Error deleting PCU Group Association", "Could not delete PCU Group Association: "+err.Error())
		return diags
	}

	// TODO does it really return 404 lol
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		diags.AddError("Error deleting PCU Group Association", "Could not delete PCU Group Association, unexpected status code: "+resp.Status)
	}

	return diags
}

// TODO: Are the two values REALLY just "accepted" and "created"?
// TODO: handle durations properly
// TODO: how does it handle retry delay?
func awaitPcuAssociationStatus(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string, target astra.PCUAssociationStatus) (*PcuGroupAssociationModel, diag.Diagnostics) {
	var ret *PcuGroupAssociationModel

	// ok to use retry from terraform-plugin-sdk because terraform-plugin-framework doesn't have an equivalent yet
	// https://discuss.hashicorp.com/t/terraform-plugin-framework-what-is-the-replacement-for-waitforstate-or-retrycontext/45538/2
	if err := retry.RetryContext(ctx, time.Duration(1<<63-1), func() *retry.RetryError {
		association, status, err := getPcuGroupAssociation(client, ctx, pcuGroupId, datacenterId)

		if (err != nil && status == 0) || (status >= 500) {
			return retry.RetryableError(err)
		}

		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("error while fetching status of PCU group association: %w", err))
		}

		if association.ProvisioningStatus.ValueString() == string(target) {
			ret = association
			return nil
		}

		return retry.RetryableError(fmt.Errorf("expected PCU group association to be status '%s' but is '%s'", target, association.ProvisioningStatus.ValueString()))
	}); err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error waiting for PCU Group Association to be provisioned", "Could not wait for PCU Group Association to be provisioned: "+err.Error())
		return nil, diags
	}

	return ret, nil
}
