package provider

import (
	"context"

	"github.com/datastax/astra-client-go/v2/astra"
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
	DeletionProtection types.Bool `tfsdk:"deletion_protection"`
	RCUProtection      types.Bool `tfsdk:"rcu_protection"` // TODO use this name? or 'reserved_protection' or something?
	Parked             types.Bool `tfsdk:"park"`
	PcuGroupModel
}

func (r *pcuGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group" // TODO we probably need to set MutableIdentity: true
}

// server should have title, min, max be required

func (r *pcuGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, res *resource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: MergeMaps(
			map[string]schema.Attribute{
				PcuAttrId: schema.StringAttribute{
					Computed: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrOrgId: schema.StringAttribute{
					Computed: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrTitle: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrCloudProvider: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrRegion: schema.StringAttribute{
					Required: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrCacheType: schema.StringAttribute{
					Optional: true,
					Computed: true,
					Default:  stringdefault.StaticString(string(astra.PcuInstanceTypeStandard)), // TODO do we validate the enum? Or let any string go?
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				PcuAttrProvisionType: schema.StringAttribute{
					Optional: true,
					Computed: true,
					Default:  stringdefault.StaticString(string(astra.PcuProvisionTypeShared)), // TODO do we validate the enum? Or let any string go?
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				PcuAttrMinCapacity: schema.Int32Attribute{
					Required: true,
					Validators: []validator.Int32{
						int32validator.AtLeast(1),
						Int32IsGTE(path.Root("reserved_capacity")),
					},
					PlanModifiers: []planmodifier.Int32{
						int32planmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrMaxCapacity: schema.Int32Attribute{
					Required: true,
					Validators: []validator.Int32{
						int32validator.AtLeast(1),
						Int32IsGTE(path.Root("min_capacity")),
					},
					PlanModifiers: []planmodifier.Int32{
						int32planmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrReservedCapacity: schema.Int32Attribute{
					Optional: true,
					Validators: []validator.Int32{
						int32validator.AtLeast(0),
					},
					PlanModifiers: []planmodifier.Int32{
						int32planmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrDescription: schema.StringAttribute{
					Optional: true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				PcuAttrStatus: schema.StringAttribute{
					Computed: true,
					PlanModifiers: []planmodifier.String{
						inferPcuGroupStatusPlanModifier(),
					},
				},
				"park": schema.BoolAttribute{ // This should technically be a WriteOnly param but that requires TF 1.12+
					Optional: true,
					Computed: true,
					Default:  booldefault.StaticBool(false),
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(), // TODO is this necessary here?
					},
				},
			},
			MkPcuResourceCreatedUpdatedAttributes(mkPcuUpdateFieldsKnownIfNoChangesOccurPlanModifier()),
			MkPcuResourceProtectionAttribute("deletion"),
			MkPcuResourceProtectionAttribute("rcu"),
		),
	}
}

func (r *pcuGroupResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	var plan pcuGroupResourceModel

	diags := req.Plan.Get(ctx, &plan)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() { // remind me to forcefully add algebraic effects to go
		return
	}

	created, diags := r.groups.Create(ctx, plan.PcuGroupSpecModel)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if plan.Parked.ValueBool() {
		created, diags = r.groups.Park(ctx, plan.Id)
		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	res.Diagnostics.Append(res.State.Set(ctx, mkPcuGroupResourceModel(
		*created,
		plan.DeletionProtection,
		plan.RCUProtection,
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

	res.Diagnostics.Append(res.State.Set(ctx, mkPcuGroupResourceModel(
		*group,
		state.DeletionProtection,
		state.RCUProtection,
	))...)
}

func (r *pcuGroupResource) Update(ctx context.Context, req resource.UpdateRequest, res *resource.UpdateResponse) {
	var plan, state pcuGroupResourceModel

	res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	res.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if res.Diagnostics.HasError() {
		return
	}

	if state.Reserved.ValueInt32() != plan.Reserved.ValueInt32() && plan.RCUProtection.ValueBool() {
		res.Diagnostics.AddError("Error Updating PCU Group", "Cannot change reserved capacity when RCU protection is enabled.")
		return
	}

	updated := &state.PcuGroupModel
	diags := diag.Diagnostics{}

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

	res.Diagnostics.Append(res.State.Set(ctx, mkPcuGroupResourceModel(
		*updated,
		plan.DeletionProtection,
		plan.RCUProtection,
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
		"The status will always be 'ACTIVE' or 'PARKED', given no major errors occurred during provisioning/unprovisioning.",
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var curr, plan *pcuGroupResourceModel

			res.Diagnostics.Append(req.State.Get(ctx, &curr)...)
			res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

			if res.Diagnostics.HasError() {
				return
			}

			if curr != nil && !shouldUpdatePcuGroup(*curr, *plan) {
				res.PlanValue = req.StateValue
			} else if plan.Parked.ValueBool() {
				res.PlanValue = types.StringValue(string(astra.PCUGroupStatusPARKED))
			}
		},
	)
}

func mkPcuUpdateFieldsKnownIfNoChangesOccurPlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		"The updated_[by|at] fields only change when properties of the PCU do.",
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var curr, plan *pcuGroupResourceModel

			res.Diagnostics.Append(req.State.Get(ctx, &curr)...)
			res.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

			if res.Diagnostics.HasError() {
				return
			}

			if curr != nil && !shouldUpdatePcuGroup(*curr, *plan) {
				res.PlanValue = req.StateValue
			}
		},
	)
}

func shouldUpdatePcuGroup(curr, plan pcuGroupResourceModel) bool {
	return !curr.Title.Equal(plan.Title) ||
		!curr.Min.Equal(plan.Min) ||
		!curr.Max.Equal(plan.Max) ||
		!curr.Reserved.Equal(plan.Reserved) ||
		!curr.Description.Equal(plan.Description)
}

func mkPcuGroupResourceModel(group PcuGroupModel, deletionProt, rcuProt types.Bool) pcuGroupResourceModel {
	return pcuGroupResourceModel{
		PcuGroupModel:      group,
		Parked:             types.BoolValue(group.Status.ValueString() == string(astra.PCUGroupStatusPARKED)),
		DeletionProtection: ElvisTF(&deletionProt, types.BoolValue(true)),
		RCUProtection:      ElvisTF(&rcuProt, types.BoolValue(true)),
	}
}
