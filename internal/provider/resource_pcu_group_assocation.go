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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func (r *pcuGroupAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group_association"
}

func (r *pcuGroupAssociationResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: MergeMaps(
			map[string]schema.Attribute{
				PcuAttrGroupId: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAssocAttrDatacenterId: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(), // TODO check we need anything fancier than this
					},
				},
				PcuAssocAttrProvisioningStatus: schema.StringAttribute{
					Computed: true,
					PlanModifiers: []planmodifier.String{
						mkPcuStatusOnlyActivePlanModifier(),
					},
				},
			},
			MkPcuResourceCreatedUpdatedAttributes(mkPcuAssociationUpdateFieldsOnlyUnknownWhenTransferOccursPlanModifier()),
			MkPcuResourceProtectionAttribute("deletion"),
		),
	}
}

// TODO what error is thrown if the association already exists (if any?) (do we need to manually check for this?)
// TODO what if it's already associated with another PCU group which Terraform doesn't know about?
// TODO basically we need to either notify the user if the association already exists without Terraform's knowledge,
// TODO or we need to execute a secret transfer request to move it to the new PCU group (which may be kinda shady)

func (r *pcuGroupAssociationResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	var plan pcuGroupAssociationResourceModel

	diags := req.Plan.Get(ctx, &plan)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	diags = createPcuGroupAssociation(r.client, ctx, plan.PCUGroupId.ValueString(), plan.DatacenterId.ValueString(), &plan.PcuGroupAssociationModel)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(req.Plan.Set(ctx, &plan)...)
}

func (r *pcuGroupAssociationResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	var data pcuGroupAssociationResourceModel

	diags := req.State.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	association, diags := getPcuGroupAssociation(r.client, ctx, data.PCUGroupId.ValueString(), data.DatacenterId.ValueString())
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if association == nil {
		res.State.RemoveResource(ctx)
		return
	}

	data.PcuGroupAssociationModel = *association
	res.Diagnostics.Append(res.State.Set(ctx, data)...)
}

func (r *pcuGroupAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	var plan, state pcuGroupAssociationResourceModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if res.Diagnostics.HasError() {
		return
	}

	// TODO do I really need this if statement (isn't it guaranteed by the plan modifier?)
	if !state.PCUGroupId.Equal(plan.PCUGroupId) {
		diags := transferPcuGroupAssociation(r.client, ctx, state.PCUGroupId.ValueString(), plan.PCUGroupId.ValueString(), state.DatacenterId.ValueString())

		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	// TODO do we need to use awaitPcuAssociationStatusCreated? (I don't think so but can't test right now)
	association, diags := getPcuGroupAssociation(r.client, ctx, plan.PCUGroupId.ValueString(), plan.DatacenterId.ValueString())

	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	plan.PcuGroupAssociationModel = *association
	res.Diagnostics.Append(res.State.Set(ctx, plan)...)
}

func (r *pcuGroupAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	var data pcuGroupAssociationResourceModel

	diags := req.State.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if data.DeletionProtection.ValueBool() {
		res.Diagnostics.AddError("Error deleting PCU Group Association", "PCU Group Association has deletion protection enabled, cannot delete.")
		return
	}

	res.Diagnostics.Append(deletePcuGroupAssociation(r.client, ctx, data.PCUGroupId.ValueString(), data.DatacenterId.ValueString())...)
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
	res, err := client.PcuAssociationCreate(ctx, pcuGroupId, datacenterId)

	if diags := HTTPResponseDiagErr(res, err, "error creating PCU group association"); diags.HasError() {
		return diags
	}

	association, diags := awaitPcuAssociationStatusCreated(client, ctx, pcuGroupId, datacenterId)

	*out = *association
	return diags
}

func getPcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string) (*PcuGroupAssociationModel, diag.Diagnostics) {
	associations, diags := GetPcuGroupAssociations(client, ctx, pcuGroupId)

	if diags.HasError() {
		return nil, diags
	}

	for _, assoc := range *associations {
		if assoc.DatacenterId.ValueString() == datacenterId {
			return &assoc, nil
		}
	}

	return nil, diags
}

// TODO: should deletion_protection also stop moving the association?
// TODO: what's returned if the association didn't exist in the first place?
func transferPcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, fromPcuGroupId string, toPcuGroupId, datacenterId string) diag.Diagnostics {
	res, err := client.PcuAssociationTransfer(ctx, astra.PcuAssociationTransferJSONRequestBody{
		FromPCUGroupUUID: &fromPcuGroupId,
		ToPCUGroupUUID:   &toPcuGroupId,
		DatacenterUUID:   &datacenterId,
	})

	if diags := HTTPResponseDiagErr(res, err, "error transferring PCU group association"); diags.HasError() {
		return diags
	}

	return nil
}

func deletePcuGroupAssociation(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string) diag.Diagnostics {
	res, err := client.PcuAssociationDelete(ctx, pcuGroupId, datacenterId)

	// TODO does it really return 404 lol
	if res != nil && res.StatusCode == 404 {
		return nil // whatever
	}

	if diags := HTTPResponseDiagErr(res, err, "error deleting PCU group association"); diags.HasError() {
		return diags
	}

	return nil
}

// TODO: Are the two values REALLY just "accepted" and "created"?
// TODO: handle timeouts + how long does an association usually take to be created?
func awaitPcuAssociationStatusCreated(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string, datacenterId string) (*PcuGroupAssociationModel, diag.Diagnostics) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		association, diags := getPcuGroupAssociation(client, ctx, pcuGroupId, datacenterId)

		if diags.HasError() {
			return nil, diags
		}

		if association.ProvisioningStatus.ValueString() == string(astra.PCUAssociationStatusCreated) {
			return association, nil
		}

		<-ticker.C
	}
}

func mkPcuStatusOnlyActivePlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		fmt.Sprintf("The status will always be '%s', given no major errors occurred during provisioning/unprovisioning.", astra.PCUAssociationStatusCreated),
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var data pcuGroupAssociationResourceModel

			diags := req.State.Get(ctx, &data)
			if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
				return
			}

			res.PlanValue = types.StringValue(string(astra.PCUAssociationStatusCreated))
		},
	)
}

func mkPcuAssociationUpdateFieldsOnlyUnknownWhenTransferOccursPlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		"The updated_[by|at] fields only change when the PCU association is transferred.", // TODO is this true?
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var curr, plan *pcuGroupAssociationResourceModel

			res.Diagnostics.Append(req.State.Get(ctx, &curr)...)
			res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

			if res.Diagnostics.HasError() {
				return
			}

			if curr != nil && curr.PCUGroupId.Equal(plan.PCUGroupId) {
				res.PlanValue = req.StateValue
			}
		},
	)
}
