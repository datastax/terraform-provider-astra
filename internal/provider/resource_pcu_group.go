package provider

import (
	"context"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &pcuGroupResource{}
	_ resource.ResourceWithConfigure   = &pcuGroupResource{}
	_ resource.ResourceWithImportState = &pcuGroupResource{}
)

func NewPcuGroupResource() resource.Resource {
	return &pcuGroupResource{}
}

type pcuGroupResource struct {
	BasePCUResource
}

type pcuGroupResourceModel struct {
	DeletionProtection types.Bool     `tfsdk:"deletion_protection"`
	ReservedProtection types.Bool     `tfsdk:"reserved_protection"`
	Parked             types.Bool     `tfsdk:"park"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
	PcuGroupModel
}

func (r *pcuGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group"
}

// server should have title, min, max be required

func (r *pcuGroupResource) Schema(ctx context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Creates and manages a PCU (Provisioned Capacity Units) group. PCU groups provide dedicated compute capacity for databases in a specific cloud provider and region.",
		Attributes: MergeMaps(
			map[string]schema.Attribute{
				PcuAttrId: schema.StringAttribute{
					Computed:    true,
					Description: "The unique identifier of the PCU group.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrOrgId: schema.StringAttribute{
					Computed:    true,
					Description: "The organization ID that owns this PCU group.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrTitle: schema.StringAttribute{
					Required:    true,
					Description: "The user-defined title/name of the PCU group.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrCloudProvider: schema.StringAttribute{
					Required:    true,
					Description: "The cloud provider where the PCU group will be provisioned (e.g., AWS, GCP, AZURE). This cannot be changed after creation.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrRegion: schema.StringAttribute{
					Required:    true,
					Description: "The cloud region where the PCU group will be provisioned. This cannot be changed after creation.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrCacheType: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString("STANDARD"), 
					Description: "The instance type/cache type for the PCU group. Defaults to 'STANDARD'. Changing this value requires replacement.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				PcuAttrProvisionType: schema.StringAttribute{
					Optional:    true,
					Computed:    true,
					Default:     stringdefault.StaticString(string(astra.PcuProvisionTypeShared)), // TODO do we validate the enum? Or let any string go?
					Description: "The provisioning type for the PCU group (e.g., SHARED, DEDICATED). Defaults to 'SHARED'. Changing this value requires replacement.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				PcuAttrMinCapacity: schema.Int32Attribute{
					Required:    true,
					Description: "The minimum capacity units the PCU must be scaled to. Must be at least 1 and greater than or equal to reserved_capacity.",
					Validators: []validator.Int32{
						int32validator.AtLeast(1),
						Int32IsGTE(path.Root("reserved_capacity")),
					},
					PlanModifiers: []planmodifier.Int32{
						int32planmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrMaxCapacity: schema.Int32Attribute{
					Required:    true,
					Description: "The maximum capacity units the PCU group may scale to. Must be at least 1 and greater than or equal to min_capacity.",
					Validators: []validator.Int32{
						int32validator.AtLeast(1),
						Int32IsGTE(path.Root("min_capacity")),
					},
					PlanModifiers: []planmodifier.Int32{
						int32planmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrReservedCapacity: schema.Int32Attribute{
					Optional:    true,
					Description: "The reserved (committed) capacity units for the PCU group. Must be at least 0. Changing this value when reserved_protection is enabled will result in an error.",
					Validators: []validator.Int32{
						int32validator.AtLeast(0),
					},
					PlanModifiers: []planmodifier.Int32{
						int32planmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrDescription: schema.StringAttribute{
					Optional:    true,
					Description: "A user-defined description for the PCU group.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrStatus: schema.StringAttribute{
					Computed:    true,
					Description: "The current status of the PCU group (e.g., ACTIVE, PARKED, CREATING, TERMINATING).",
					PlanModifiers: []planmodifier.String{
						inferPcuGroupStatusPlanModifier(),
					},
				},
				"park": schema.BoolAttribute{ // This should technically be a WriteOnly param but that requires TF 1.12+
					Optional:    true,
					Computed:    true,
					Default:     booldefault.StaticBool(false),
					Description: "When set to true, parks the PCU group and any associated databases, reducing costs. When set to false, unparks the group. Defaults to false.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(), // TODO should it also wait for the dbs to become hibernated/active? or will the PCU group itself wait?
					},
				},
			},
			MkPcuResourceCreatedUpdatedAttributes(mkPcuUpdateFieldsKnownIfNoChangesOccurPlanModifier()),
			MkPcuResourceProtectionAttribute("deletion", "When enabled, prevents accidental deletion of the PCU group."),
			MkPcuResourceProtectionAttribute("reserved", "When enabled, prevents accidental reserved capacity unit increases."),
		),
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func (r *pcuGroupResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	var plan pcuGroupResourceModel

	diags := req.Plan.Get(ctx, &plan)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() { // remind me to forcefully add algebraic effects to go
		return
	}

	if plan.Parked.ValueBool() {
		res.Diagnostics.AddError("Error Creating PCU Group", "PCU Groups cannot be created in a parked state, as they can only be parked after an association has been made. If you want to create a PCU Group and park it, please create it unparked first, then update the resource to be parked afterwards.")
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 20*time.Minute)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	created, diags := r.groups.Create(ctx, plan.PcuGroupSpecModel)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, plan.updated(
		*created,
		plan.DeletionProtection,
		plan.ReservedProtection,
	))...)
}

func (r *pcuGroupResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	var state pcuGroupResourceModel

	diags := req.State.Get(ctx, &state)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	group, diags := r.groups.FindOne(ctx, state.Id)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if group == nil {
		res.State.RemoveResource(ctx)
		return
	}

	res.Diagnostics.Append(res.State.Set(ctx, state.updated(
		*group,
		state.DeletionProtection,
		state.ReservedProtection,
	))...)
}

func (r *pcuGroupResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	var plan, state pcuGroupResourceModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if res.Diagnostics.HasError() {
		return
	}

	if state.Reserved.ValueInt32() != plan.Reserved.ValueInt32() && plan.ReservedProtection.ValueBool() {
		res.Diagnostics.AddError("Error Updating PCU Group", "Cannot change reserved capacity when RCU protection is enabled.")
		return
	}

	updated := &state.PcuGroupModel
	diags := diag.Diagnostics{}

	updateTimeout, diags := plan.Timeouts.Update(ctx, 20*time.Minute)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	if shouldUpdatePcuGroup(state, plan) {
		updated, diags = r.groups.Update(ctx, plan.Id, plan.PcuGroupSpecModel)

		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	if state.Parked.ValueBool() != plan.Parked.ValueBool() {
		if plan.Parked.ValueBool() {
			updated, diags = r.groups.Park(ctx, plan.Id)
		} else {
			updated, diags = r.groups.Unpark(ctx, plan.Id)
		}

		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	res.Diagnostics.Append(res.State.Set(ctx, plan.updated(
		*updated,
		plan.DeletionProtection,
		plan.ReservedProtection,
	))...)
}

func (r *pcuGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, res *resource.DeleteResponse) {
	var data pcuGroupResourceModel

	diags := req.State.Get(ctx, &data)

	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if data.DeletionProtection.ValueBool() {
		res.Diagnostics.AddError("Error Deleting PCU Group", "PCU Group has deletion protection enabled, cannot delete.")
		return
	}

	res.Diagnostics.Append(r.groups.Delete(ctx, data.Id)...)
}

func (r *pcuGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}

func inferPcuGroupStatusPlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		"The status will be 'PARKED' when park=true, 'ACTIVE' or 'CREATED' when park=false.",
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var curr, plan *pcuGroupResourceModel

			res.Diagnostics.Append(req.State.Get(ctx, &curr)...)
			res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

			if res.Diagnostics.HasError() {
				return
			}

			// Determine expected status based on park state
			if plan.Parked.ValueBool() {
				res.PlanValue = types.StringValue(string(astra.PCUGroupStatusPARKED))
			} else if curr != nil && !shouldPcuGroupStateChange(*curr, *plan) {
				// No changes at all - keep current status
				res.PlanValue = req.StateValue
			}
			// Otherwise leave as Unknown - will be ACTIVE or CREATED after apply
		},
	)
}

func mkPcuUpdateFieldsKnownIfNoChangesOccurPlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		"The updated_[by|at] fields change when properties of the PCU change or when parking/unparking.",
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var curr, plan *pcuGroupResourceModel

			res.Diagnostics.Append(req.State.Get(ctx, &curr)...)
			res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

			if res.Diagnostics.HasError() {
				return
			}

			// Only preserve the old value if NOTHING is changing
			if curr != nil && !shouldPcuGroupStateChange(*curr, *plan) {
				res.PlanValue = req.StateValue
			}
			// Otherwise, leave as Unknown so Terraform knows it will change
		},
	)
}

func shouldPcuGroupStateChange(curr, plan pcuGroupResourceModel) bool {
	return shouldUpdatePcuGroup(curr, plan) || !curr.Parked.Equal(plan.Parked)
}

func shouldUpdatePcuGroup(curr, plan pcuGroupResourceModel) bool {
	return !curr.Title.Equal(plan.Title) ||
		!curr.Min.Equal(plan.Min) ||
		!curr.Max.Equal(plan.Max) ||
		!curr.Reserved.Equal(plan.Reserved) ||
		!curr.Description.Equal(plan.Description)
}

func (m pcuGroupResourceModel) updated(group PcuGroupModel, deletionProt, rcuProt types.Bool) pcuGroupResourceModel {
	return pcuGroupResourceModel{
		PcuGroupModel:      group,
		Parked:             types.BoolValue(group.Status.ValueString() == string(astra.PCUGroupStatusPARKED)),
		DeletionProtection: ElvisTF(&deletionProt, types.BoolValue(true)),
		ReservedProtection: ElvisTF(&rcuProt, types.BoolValue(true)),
		Timeouts:           m.Timeouts,
	}
}
