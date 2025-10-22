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
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

// TODO should this wait for both the db and pcu group to become active??? is it even the job of this resource???

func (r *pcuGroupAssociationResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	var plan pcuGroupAssociationResourceModel

	diags := req.Plan.Get(ctx, &plan)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	created, diags := r.associations.Create(ctx, plan.PCUGroupId, plan.DatacenterId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	diags = awaitDbAndPcuGroupActiveStatus(ctx, r.BasePCUResource, plan.PCUGroupId, plan.DatacenterId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	plan.PcuGroupAssociationModel = *created
	res.Diagnostics.Append(req.Plan.Set(ctx, &plan)...)
}

func (r *pcuGroupAssociationResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	var state pcuGroupAssociationResourceModel

	diags := req.State.Get(ctx, &state)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	association, diags := r.associations.FindOne(ctx, state.PCUGroupId, state.DatacenterId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if association == nil {
		res.State.RemoveResource(ctx)
		return
	}

	state.PcuGroupAssociationModel = *association
	res.Diagnostics.Append(res.State.Set(ctx, state)...)
}

func (r *pcuGroupAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	var plan, state pcuGroupAssociationResourceModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if res.Diagnostics.HasError() {
		return
	}

	if !state.PCUGroupId.Equal(plan.PCUGroupId) {
		diags := r.associations.Transfer(ctx, state.PCUGroupId, plan.PCUGroupId, state.DatacenterId)

		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}

		diags = awaitDbAndPcuGroupActiveStatus(ctx, r.BasePCUResource, plan.PCUGroupId, plan.DatacenterId)

		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	association, diags := r.associations.FindOne(ctx, plan.PCUGroupId, plan.DatacenterId)

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

	res.Diagnostics.Append(r.associations.Delete(ctx, data.PCUGroupId, data.DatacenterId)...)
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

func mkPcuStatusOnlyActivePlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		fmt.Sprintf("The status will always be '%s', given no major errors occurred during provisioning/unprovisioning.", astra.PCUAssociationStatusCreated),
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			//res.PlanValue = types.StringValue(string(astra.PCUAssociationStatusCreated))
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

func awaitDbAndPcuGroupActiveStatus(ctx context.Context, r BasePCUResource, pcuGroupId, datacenterId types.String) diag.Diagnostics {
	if _, diags := r.groups.AwaitStatus(ctx, pcuGroupId, astra.PCUGroupStatusACTIVE); diags.HasError() {
		return diags
	}

	return awaitDbActiveStatus(ctx, r.client, datacenterId)
}

func awaitDbActiveStatus(ctx context.Context, client *astra.ClientWithResponses, datacenterId types.String) diag.Diagnostics {
	idParts := strings.Split(datacenterId.ValueString(), "-")

	if len(idParts) < 6 {
		return DiagErr("Invalid datacenter ID", fmt.Sprintf("Expected format <uuid>-<number>; got: %s", datacenterId.ValueString()))
	}

	databaseId := strings.Join(idParts[:5], "-")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	attempts := 0

	tflog.Debug(ctx, fmt.Sprintf("Waiting for DB %s to reach status ACTIVE (attempt 0)", databaseId))

	for {
		attempts += 1

		resp, err := client.GetDatabaseWithResponse(ctx, databaseId)

		if diags := ParsedHTTPResponseDiagErr(resp, err, "error retrieving database status"); diags.HasError() {
			return diags
		}

		switch resp.JSON200.Status {
		case astra.StatusEnumACTIVE:
			tflog.Debug(ctx, fmt.Sprintf("DB %s reached status ACTIVE", databaseId))
			return nil
		case "ASSOCIATING", astra.StatusEnumINITIALIZING, astra.StatusEnumPENDING:
			tflog.Debug(ctx, fmt.Sprintf("Waiting for DB %s to reach status ACTIVE (attempt %d, currently %s)", databaseId, attempts, resp.JSON200.Status))
		default:
			return DiagErr("Error waiting for database to become ACTIVE", fmt.Sprintf("Database %s reached unexpected status %s", databaseId, resp.JSON200.Status))
		}

		<-ticker.C
	}
}
