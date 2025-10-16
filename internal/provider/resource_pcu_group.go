package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
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
	ShouldBeParked     types.Bool `tfsdk:"park"`
	PcuGroupModel
}

func (r *pcuGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pcu_group"
}

func (r *pcuGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"deletion_protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(), // TODO is this necessary here?
				},
			},
			"rcu_protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(), // TODO is this necessary here?
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
			"org_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // TODO it's not possible for this to change... right?
				},
			},
			"title": schema.StringAttribute{
				Optional: true,
			},
			"cloud_provider": schema.StringAttribute{
				Required: true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"cache_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(string(astra.InstanceTypeStandard)), // TODO is 'InstanceType' too generic a name? (clashing possibility)
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provision_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(string(astra.Shared)), // TODO 'Shared' should probably be ProvisionTypeShared
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"min_capacity": schema.Int32Attribute{
				Optional: true,
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
					Int32IsGTE(path.Root("reserved_capacity")),
				},
			},
			"max_capacity": schema.Int32Attribute{
				Optional: true,
				Validators: []validator.Int32{
					int32validator.AtLeast(1),
					Int32IsGTE(path.Root("min_capacity")),
				},
			},
			"reserved_capacity": schema.Int32Attribute{
				Optional: true,
				Validators: []validator.Int32{
					int32validator.AtLeast(0),
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
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
			"status": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *pcuGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pcuGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(createPcuGroup(r.client, ctx, plan, &plan.PcuGroupModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ShouldBeParked.ValueBool() {
		resp.Diagnostics.Append(parkPcuGroup(r.client, ctx, plan.Id, &plan.PcuGroupModel)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pcuGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data pcuGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pcuGroup, status, err := GetPcuGroup(r.client, ctx, data.Id)

	if status == 404 {
		resp.State.RemoveResource(ctx)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error Reading PCU Group", "Could not read PCU Group: "+err.Error())
		return
	}

	data.PcuGroupModel = *pcuGroup
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *pcuGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pcuGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if state.Reserved.ValueInt32() != plan.Reserved.ValueInt32() && plan.RCUProtection.ValueBool() {
		resp.Diagnostics.AddError("Error Updating PCU Group", "Cannot change reserved capacity when RCU protection is enabled.")
		return
	}

	resp.Diagnostics.Append(updatePcuGroup(r.client, ctx, plan, &plan.PcuGroupModel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.ShouldBeParked.ValueBool() != plan.ShouldBeParked.ValueBool() {
		if plan.ShouldBeParked.ValueBool() {
			resp.Diagnostics.Append(parkPcuGroup(r.client, ctx, plan.Id, &plan.PcuGroupModel)...)
		} else {
			resp.Diagnostics.Append(unparkPcuGroup(r.client, ctx, plan.Id, &plan.PcuGroupModel)...)
		}

		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *pcuGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data pcuGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.DeletionProtection.ValueBool() {
		resp.Diagnostics.AddError("Error Deleting PCU Group", "PCU Group has deletion protection enabled, cannot delete.")
		return
	}

	// TODO any more validations? (need to check docs)
	resp.Diagnostics.Append(deletePcuGroup(r.client, ctx, data.Id)...)
}

func (r *pcuGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createPcuGroup(client *astra.ClientWithResponses, ctx context.Context, spec pcuGroupResourceModel, out *PcuGroupModel) diag.Diagnostics {
	var diags diag.Diagnostics

	cp := astra.CloudProvider(spec.CloudProvider.ValueString())
	it := astra.InstanceType(spec.InstanceType.ValueString())
	pt := astra.ProvisionType(spec.ProvisionType.ValueString())

	minCap := int(spec.Min.ValueInt32())
	maxCap := int(spec.Max.ValueInt32())
	reservedCap := int(spec.Reserved.ValueInt32())

	resp, err := client.PcuCreateWithResponse(ctx, astra.PcuCreateJSONRequestBody{
		astra.PCUGroupCreateRequest{
			Title:         spec.Title.ValueStringPointer(),
			CloudProvider: &cp,
			Region:        spec.Region.ValueStringPointer(),
			InstanceType:  &it,
			ProvisionType: &pt,
			Min:           &minCap, // TODO can we make these just take v
			Max:           &maxCap,
			Reserved:      &reservedCap,
			Description:   spec.Description.ValueStringPointer(),
		},
	})

	if err != nil {
		diags.AddError("Error creating PCU Group Association", "Could not create PCU Group Association: "+err.Error())
		return diags
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		diags.AddError("Error creating PCU Group Association", "Could not create PCU Group Association, unexpected status code: "+resp.HTTPResponse.Status)
		return diags
	}

	awaitedPcuGroup, moreDiags := awaitPcuStatus(client, ctx, types.StringValue(*(*resp.JSON201)[0].Uuid), astra.PCUGroupStatusACTIVE)
	diags.Append(moreDiags...)

	*out = *awaitedPcuGroup
	return diags
}

// TODO what if the PCU group doesn't exist in the first place? (what does the API return?)
func updatePcuGroup(client *astra.ClientWithResponses, ctx context.Context, spec pcuGroupResourceModel, out *PcuGroupModel) diag.Diagnostics {
	var diags diag.Diagnostics

	minCap := int(spec.Min.ValueInt32())
	maxCap := int(spec.Max.ValueInt32())
	reservedCap := int(spec.Reserved.ValueInt32())

	resp, err := client.PcuUpdateWithResponse(ctx, astra.PcuUpdateJSONRequestBody{ // TODO is it reliable to depend on the response from updating the PCU? Or is it better to do a GET after?
		astra.PCUGroupUpdateRequest{
			Title:       spec.Title.ValueStringPointer(), // TODO can you really update instance and provision type? Also do we still lose them if they're not provided?
			Min:         &minCap,
			Max:         &maxCap,
			Reserved:    &reservedCap,
			Description: spec.Description.ValueStringPointer(),
		},
	})

	if err != nil {
		diags.AddError("Error updating PCU Group", "Could not update PCU Group: "+err.Error())
		return diags
	}

	if resp.HTTPResponse.StatusCode >= 400 {
		diags.AddError("Error updating PCU Group", "Could not update PCU Group, unexpected status code: "+resp.HTTPResponse.Status)
		return diags
	}

	*out = DeserializePcuGroupFromAPI((*resp.JSON200)[0])
	return diags
}

func parkPcuGroup(client *astra.ClientWithResponses, ctx context.Context, id types.String, out *PcuGroupModel) diag.Diagnostics {
	var diags diag.Diagnostics

	_, err := client.PcuGroupPark(ctx, id.ValueString())
	if err != nil {
		diags.AddError("Error parking PCU Group", "Could not park PCU Group: "+err.Error())
		return diags
	}

	awaitedPcuGroup, moreDiags := awaitPcuStatus(client, ctx, id, astra.PCUGroupStatusPARKED)
	diags.Append(moreDiags...)

	*out = *awaitedPcuGroup
	return diags
}

func unparkPcuGroup(client *astra.ClientWithResponses, ctx context.Context, id types.String, out *PcuGroupModel) diag.Diagnostics {
	var diags diag.Diagnostics

	_, err := client.PcuGroupUnpark(ctx, id.ValueString())
	if err != nil {
		diags.AddError("Error unparking PCU Group", "Could not unpark PCU Group: "+err.Error())
		return diags
	}

	awaitedPcuGroup, moreDiags := awaitPcuStatus(client, ctx, id, astra.PCUGroupStatusACTIVE)
	diags.Append(moreDiags...)

	*out = *awaitedPcuGroup
	return diags
}

func deletePcuGroup(client *astra.ClientWithResponses, ctx context.Context, id types.String) diag.Diagnostics {
	var diags diag.Diagnostics

	resp, err := client.PcuDelete(ctx, id.ValueString())

	if err != nil {
		diags.AddError("Error deleting PCU Group", "Could not delete PCU Group: "+err.Error())
		return diags
	}

	// TODO does it really return 404 lol
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		diags.AddError("Error deleting PCU Group", "Could not delete PCU Group, unexpected status code: "+resp.Status)
	}

	return diags
}

func awaitPcuStatus(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId types.String, target astra.PCUGroupStatus) (*PcuGroupModel, diag.Diagnostics) {
	var ret *PcuGroupModel

	// ok to use retry from terraform-plugin-sdk because terraform-plugin-framework doesn't have an equivalent yet
	// https://discuss.hashicorp.com/t/terraform-plugin-framework-what-is-the-replacement-for-waitforstate-or-retrycontext/45538/2
	if err := retry.RetryContext(ctx, time.Duration(1<<63-1), func() *retry.RetryError {
		pcuGroup, status, err := GetPcuGroup(client, ctx, pcuGroupId)

		if (err != nil && status == 0) || (status >= 500) {
			return retry.RetryableError(err)
		}

		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("error while fetching status of PCU: %w", err))
		}

		if pcuGroup.Status.ValueString() == string(target) {
			ret = pcuGroup
			return nil
		}

		return retry.RetryableError(fmt.Errorf("expected PCU group to be status '%s' but is '%s'", target, pcuGroup.Status.ValueString()))
	}); err != nil {
		var diags diag.Diagnostics
		diags.AddError("Error waiting for PCU Group to be provisioned", "Could not wait for PCU Group to be provisioned: "+err.Error())
		return nil, diags
	}

	return ret, nil
}
