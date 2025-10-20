package provider

import (
	"context"
	"time"

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
	Park               types.Bool `tfsdk:"park"`
	PcuGroupModel
}

func (r *pcuGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, res *resource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group"
}

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
						stringplanmodifier.UseStateForUnknown(), // TODO it's not possible for this to change... right?
					},
				},
				PcuAttrTitle: schema.StringAttribute{
					Optional: true,
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
					Default:  stringdefault.StaticString(string(astra.InstanceTypeStandard)), // TODO is 'InstanceType' too generic a name for the astra-go-client? (clashing possibility)
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				PcuAttrProvisionType: schema.StringAttribute{
					Optional: true,
					Computed: true,
					Default:  stringdefault.StaticString(string(astra.Shared)), // TODO 'Shared' should probably be ProvisionTypeShared
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
						stringplanmodifier.RequiresReplace(),
					},
				},
				PcuAttrMinCapacity: schema.Int32Attribute{ // TODO these are technically required then, right? b/c you get "error validating request: min must be greater than or equal to 1" if both aren't set
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
						mkPcuStatusOnlyActiveOrParkedPlanModifier(),
					},
				},
				"park": schema.BoolAttribute{
					Optional: true,
					Computed: true,
					Default:  booldefault.StaticBool(false),
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(), // TODO is this necessary here?
					},
				},
			},
			MkPcuResourceCreatedUpdatedAttributes(mkPcuUpdateFieldsOnlyUnknownWhenChangesOccurPlanModifier()),
			MkPcuResourceProtectionAttribute("deletion"),
			MkPcuResourceProtectionAttribute("rcu"),
		),
	}
}

func (r *pcuGroupResource) Create(ctx context.Context, req resource.CreateRequest, res *resource.CreateResponse) {
	var plan pcuGroupResourceModel

	diags := req.Plan.Get(ctx, &plan)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	diags = createPcuGroup(r.client, ctx, plan, &plan.PcuGroupModel)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if plan.Park.ValueBool() {
		diags = parkPcuGroup(r.client, ctx, plan.Id, &plan.PcuGroupModel)
		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	res.Diagnostics.Append(res.State.Set(ctx, plan)...)
}

func (r *pcuGroupResource) Read(ctx context.Context, req resource.ReadRequest, res *resource.ReadResponse) {
	var data pcuGroupResourceModel

	diags := req.State.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	group, diags := GetPcuGroup(r.client, ctx, data.Id)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if group == nil {
		res.State.RemoveResource(ctx)
		return
	}

	data.PcuGroupModel = *group

	data.Park = types.BoolValue(group.Status.ValueString() == string(astra.PCUGroupStatusPARKED))

	if data.DeletionProtection.IsNull() || data.DeletionProtection.IsUnknown() {
		data.DeletionProtection = types.BoolValue(true)
	}

	if data.RCUProtection.IsNull() || data.RCUProtection.IsUnknown() {
		data.RCUProtection = types.BoolValue(true)
	}

	res.Diagnostics.Append(res.State.Set(ctx, data)...)
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

	if shouldUpdatePcuGroup(state, plan) {
		diags := updatePcuGroup(r.client, ctx, plan, &plan.PcuGroupModel)

		if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
			return
		}
	}

	if state.Park.ValueBool() != plan.Park.ValueBool() {
		if plan.Park.ValueBool() {
			res.Diagnostics.Append(parkPcuGroup(r.client, ctx, plan.Id, &plan.PcuGroupModel)...)
		} else {
			res.Diagnostics.Append(unparkPcuGroup(r.client, ctx, plan.Id, &plan.PcuGroupModel)...)
		}

		if res.Diagnostics.HasError() {
			return
		}
	}

	res.Diagnostics.Append(res.State.Set(ctx, plan)...)
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

	// TODO any more validations? (need to check docs)
	res.Diagnostics.Append(deletePcuGroup(r.client, ctx, data.Id)...)
}

func (r *pcuGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, res *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, res)
}

func createPcuGroup(client *astra.ClientWithResponses, ctx context.Context, spec pcuGroupResourceModel, out *PcuGroupModel) diag.Diagnostics {
	cp := astra.CloudProvider(spec.CloudProvider.ValueString())
	it := astra.InstanceType(spec.InstanceType.ValueString())
	pt := astra.ProvisionType(spec.ProvisionType.ValueString())

	minCap := int(spec.Min.ValueInt32())
	maxCap := int(spec.Max.ValueInt32())
	reservedCap := int(spec.Reserved.ValueInt32())

	res, err := client.PcuCreateWithResponse(ctx, astra.PcuCreateJSONRequestBody{
		astra.PCUGroupCreateRequest{
			Title:         spec.Title.ValueStringPointer(),
			CloudProvider: &cp,
			Region:        spec.Region.ValueStringPointer(),
			InstanceType:  &it,
			ProvisionType: &pt,
			Min:           &minCap, // TODO can we make these just take values instead of references
			Max:           &maxCap,
			Reserved:      &reservedCap,
			Description:   spec.Description.ValueStringPointer(),
		},
	})

	if diags := ParsedHTTPResponseDiagErr(res, err, "failed to create PCU group"); diags.HasError() {
		return diags
	}

	awaitedPcuGroup, diags := awaitPcuStatus(client, ctx, types.StringValue(*(*res.JSON201)[0].Uuid), astra.PCUGroupStatusACTIVE)

	*out = *awaitedPcuGroup
	return diags
}

func shouldUpdatePcuGroup(state, plan pcuGroupResourceModel) bool {
	return !state.Title.Equal(plan.Title) ||
		!state.Min.Equal(plan.Min) ||
		!state.Max.Equal(plan.Max) ||
		!state.Reserved.Equal(plan.Reserved) ||
		!state.Description.Equal(plan.Description)
}

// TODO what if the PCU group doesn't exist in the first place? (what does the API return?)
func updatePcuGroup(client *astra.ClientWithResponses, ctx context.Context, spec pcuGroupResourceModel, out *PcuGroupModel) diag.Diagnostics {
	minCap := int(spec.Min.ValueInt32())
	maxCap := int(spec.Max.ValueInt32())
	reservedCap := int(spec.Reserved.ValueInt32())

	res, err := client.PcuUpdateWithResponse(ctx, astra.PcuUpdateJSONRequestBody{
		astra.PCUGroupUpdateRequest{
			Title:       spec.Title.ValueStringPointer(), // TODO can you really update instance and provision type? Also do we still lose them if they're not provided?
			Min:         &minCap,
			Max:         &maxCap,
			Reserved:    &reservedCap,
			Description: spec.Description.ValueStringPointer(),
		},
	})

	if diags := ParsedHTTPResponseDiagErr(res, err, "failed to update PCU group"); diags.HasError() {
		return diags
	}

	// TODO is it reliable to depend on the response from updating the PCU? Or is it better to do a GET after?
	*out = DeserializePcuGroupFromAPI((*res.JSON200)[0])
	return nil
}

func parkPcuGroup(client *astra.ClientWithResponses, ctx context.Context, id types.String, out *PcuGroupModel) diag.Diagnostics {
	res, err := client.PcuGroupPark(ctx, id.ValueString())

	if diags := HTTPResponseDiagErr(res, err, "error parking pcu group"); diags.HasError() {
		return diags
	}

	awaitedPcuGroup, diags := awaitPcuStatus(client, ctx, id, astra.PCUGroupStatusPARKED)

	*out = *awaitedPcuGroup
	return diags
}

func unparkPcuGroup(client *astra.ClientWithResponses, ctx context.Context, id types.String, out *PcuGroupModel) diag.Diagnostics {
	res, err := client.PcuGroupUnpark(ctx, id.ValueString())

	if diags := HTTPResponseDiagErr(res, err, "error unparking pcu group"); diags.HasError() {
		return diags
	}

	awaitedPcuGroup, diags := awaitPcuStatus(client, ctx, id, astra.PCUGroupStatusACTIVE)

	*out = *awaitedPcuGroup
	return diags
}

func deletePcuGroup(client *astra.ClientWithResponses, ctx context.Context, id types.String) diag.Diagnostics {
	res, err := client.PcuDelete(ctx, id.ValueString())

	if res != nil && res.StatusCode == 404 {
		return nil // whatever
	}

	if diags := HTTPResponseDiagErr(res, err, "error deleting PCU group"); diags.HasError() {
		return diags
	}

	return nil
}

func awaitPcuStatus(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId types.String, target astra.PCUGroupStatus) (*PcuGroupModel, diag.Diagnostics) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		group, diags := GetPcuGroup(client, ctx, pcuGroupId)

		if diags.HasError() {
			return nil, diags
		}

		if group.Status.ValueString() == string(target) {
			return group, nil
		}

		<-ticker.C
	}
}

func mkPcuStatusOnlyActiveOrParkedPlanModifier() planmodifier.String {
	return MkStringPlanModifier(
		"The status will always be 'ACTIVE' or 'PARKED', given no major errors occurred during provisioning/unprovisioning.",
		func(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
			var data pcuGroupResourceModel

			diags := req.Plan.Get(ctx, &data)
			if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
				return
			}

			if data.Park.ValueBool() {
				res.PlanValue = types.StringValue(string(astra.PCUGroupStatusPARKED))
			} else {
				res.PlanValue = types.StringValue(string(astra.PCUGroupStatusACTIVE))
			}
		},
	)
}

func mkPcuUpdateFieldsOnlyUnknownWhenChangesOccurPlanModifier() planmodifier.String {
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
