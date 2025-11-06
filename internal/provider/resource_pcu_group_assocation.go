package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

// TODO should I add back deletion and transfer protection???
type pcuGroupAssociationResourceModel struct {
	PCUGroupId types.String   `tfsdk:"pcu_group_id"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
	PcuGroupAssociationModel
}

func (r *pcuGroupAssociationResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group_association"
}

func (r *pcuGroupAssociationResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Creates a transferable association between an existing PCU group and datacenter.",
		Attributes: MergeMaps(
			map[string]schema.Attribute{
				PcuAttrGroupId: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
					Description: "The PCU group to associate the datacenter with. This may be changed to transfer the association to another PCU group.",
				},
				PcuAssocAttrDatacenterId: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
					Description: "The datacenter to associate with the PCU group. Note that this is a datacenter ID, not a database ID. The `provider::astra::resolve_datacenter` function may be used to easily obtain the datacenter ID from a database ID.",
				},
				PcuAssocAttrProvisioningStatus: schema.StringAttribute{
					Computed: true,
					PlanModifiers: []planmodifier.String{
						mkPcuStatusOnlyActivePlanModifier(),
					},
					Description: "The provisioning status of the PCU group association. This will likely always be 'CREATED'.",
				},
			},
			//MkPcuResourceCreatedUpdatedAttributes(inferPcuGroupAssociationStatusPlanModifier()), TODO add these later once they're added to the server
		),
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

// TODO what error is thrown if the association already exists (if any?) (do we need to manually check for this?)

// TODO what if it's already associated with another PCU group which Terraform doesn't know about?
// TODO basically we need to either notify the user if the association already exists without Terraform's knowledge,
// TODO or we need to execute a secret transfer request to move it to the new PCU group (which may be kinda shady)

// TODO error creating or transferring association to non CREATED or ACTIVE or INITIALIZING pcu group

func (r *pcuGroupAssociationResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	var plan pcuGroupAssociationResourceModel

	diags := req.Plan.Get(ctx, &plan)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 40*time.Minute)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	created, diags := r.associations.Create(ctx, plan.PCUGroupId, plan.DatacenterId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	diags = awaitDbAndPcuGroupActiveStatus(ctx, r.BasePCUResource, plan.PCUGroupId, plan.DatacenterId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, plan.updated(
		plan.PCUGroupId,
		*created,
	))...)
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

	res.Diagnostics.Append(res.State.Set(ctx, state.updated(
		state.PCUGroupId,
		*association,
	))...)
}

func (r *pcuGroupAssociationResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	var plan, state pcuGroupAssociationResourceModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if res.Diagnostics.HasError() {
		return
	}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 40*time.Minute)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

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

	if association == nil {
		res.Diagnostics.AddError("Could not find PCU Group Association after update", "The PCU Group Association could not be found after transfer under the PCU Group ID "+plan.PCUGroupId.ValueString())
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, plan.updated(
		plan.PCUGroupId,
		*association,
	))...)
}

// TODO is deleting an association instantaneous?

func (r *pcuGroupAssociationResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	var data pcuGroupAssociationResourceModel

	diags := req.State.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
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

func inferPcuGroupAssociationStatusPlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		"The updated_[by|at] fields only change when the PCU association is transferred.",
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
		select {
		case <-ctx.Done():
			// Respect context timeout or cancellation
			return DiagErr("Timeout while waiting for database to become ACTIVE", ctx.Err().Error())

		case <-ticker.C:
			attempts++

			resp, err := client.GetDatabaseWithResponse(ctx, databaseId)
			if diags := ParsedHTTPResponseDiagErr(resp, err, "error retrieving database status"); diags.HasError() {
				return diags
			}

			switch resp.JSON200.Status {
			case astra.ACTIVE:
				tflog.Debug(ctx, fmt.Sprintf("DB %s reached status ACTIVE", databaseId))
				return nil
			case "ASSOCIATING", astra.INITIALIZING, astra.PENDING:
				tflog.Debug(ctx, fmt.Sprintf("Waiting for DB %s to reach status ACTIVE (attempt %d, currently %s)", databaseId, attempts, resp.JSON200.Status))
			default:
				return DiagErr("Error waiting for database to become ACTIVE", fmt.Sprintf("Database %s reached unexpected status %s", databaseId, resp.JSON200.Status))
			}
		}
	}
}

func (m pcuGroupAssociationResourceModel) updated(pcuGroupId types.String, association PcuGroupAssociationModel) pcuGroupAssociationResourceModel {
	return pcuGroupAssociationResourceModel{
		PCUGroupId:               pcuGroupId,
		PcuGroupAssociationModel: association,
		Timeouts:                 m.Timeouts,
	}
}
