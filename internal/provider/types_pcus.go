package provider

import (
	datasourceSchema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceSchema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PcuGroupSpecModel struct {
	Title         types.String `tfsdk:"title"`          // TODO verify naming changes
	CloudProvider types.String `tfsdk:"cloud_provider"` // TODO maybe use actual domain value types (e.g. CloudProvider) here w/ custom types
	Region        types.String `tfsdk:"region"`
	InstanceType  types.String `tfsdk:"cache_type"`
	ProvisionType types.String `tfsdk:"provision_type"`
	Min           types.Int32  `tfsdk:"min_capacity"`
	Max           types.Int32  `tfsdk:"max_capacity"`
	Reserved      types.Int32  `tfsdk:"reserved_capacity"`
	Description   types.String `tfsdk:"description"`
}

type PcuGroupModel struct {
	PcuGroupSpecModel
	Id        types.String `tfsdk:"id"`
	OrgId     types.String `tfsdk:"org_id"` // TODO verify if org_id is actually part of the user-definable spec or not
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
	CreatedBy types.String `tfsdk:"created_by"`
	UpdatedBy types.String `tfsdk:"updated_by"`
	Status    types.String `tfsdk:"status"`
}

// PcuGroupAssociationModel we'll ignore pcu_group_id here since it's a one-to-many relationship
type PcuGroupAssociationModel struct {
	DatacenterId       types.String `tfsdk:"datacenter_id"`
	ProvisioningStatus types.String `tfsdk:"provisioning_status"`
	//CreatedAt          types.String `tfsdk:"created_at"` TODO what is going on here
	//UpdatedAt          types.String `tfsdk:"updated_at"`
	//CreatedBy          types.String `tfsdk:"created_by"`
	//UpdatedBy          types.String `tfsdk:"updated_by"`
}

var (
	PcuAttrGroupId  = "pcu_group_id"
	PcuAttrGroupIds = PcuAttrGroupId + "s"
)

var (
	PcuAttrId               = "id"
	PcuAttrOrgId            = "org_id"
	PcuAttrTitle            = "title"
	PcuAttrCloudProvider    = "cloud_provider"
	PcuAttrRegion           = "region"
	PcuAttrCacheType        = "cache_type"
	PcuAttrProvisionType    = "provision_type"
	PcuAttrMinCapacity      = "min_capacity"
	PcuAttrMaxCapacity      = "max_capacity"
	PcuAttrReservedCapacity = "reserved_capacity"
	PcuAttrDescription      = "description"
	PcuAttrCreatedAt        = "created_at"
	PcuAttrUpdatedAt        = "updated_at"
	PcuAttrCreatedBy        = "created_by"
	PcuAttrUpdatedBy        = "updated_by"
	PcuAttrStatus           = "status"
)

var (
	PcuAssocAttrDatacenterId       = "datacenter_id"
	PcuAssocAttrProvisioningStatus = "provisioning_status"
	PcuAssocAttrCreatedAt          = PcuAttrCreatedAt
	PcuAssocAttrUpdatedAt          = PcuAttrUpdatedAt
	PcuAssocAttrCreatedBy          = PcuAttrCreatedBy
	PcuAssocAttrUpdatedBy          = PcuAttrUpdatedBy
)

func MkPcuGroupDataSourceAttributes() map[string]datasourceSchema.Attribute {
	return map[string]datasourceSchema.Attribute{
		PcuAttrId: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrOrgId: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrTitle: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrCloudProvider: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrRegion: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrCacheType: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrProvisionType: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrMinCapacity: datasourceSchema.Int64Attribute{
			Computed: true,
		},
		PcuAttrMaxCapacity: datasourceSchema.Int64Attribute{
			Computed: true,
		},
		PcuAttrReservedCapacity: datasourceSchema.Int64Attribute{
			Computed: true,
		},
		PcuAttrDescription: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrCreatedAt: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrUpdatedAt: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrCreatedBy: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrUpdatedBy: datasourceSchema.StringAttribute{
			Computed: true,
		},
		PcuAttrStatus: datasourceSchema.StringAttribute{
			Computed: true,
		},
	}
}

func MkPcuResourceCreatedUpdatedAttributes(updatePlanModifier planmodifier.String) map[string]resourceSchema.Attribute {
	return map[string]resourceSchema.Attribute{
		"created_at": resourceSchema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"updated_at": resourceSchema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				updatePlanModifier,
			},
		},
		"created_by": resourceSchema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"updated_by": resourceSchema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				updatePlanModifier,
			},
		},
	}
}

func MkPcuResourceProtectionAttribute(thing string) map[string]resourceSchema.Attribute {
	return map[string]resourceSchema.Attribute{
		thing + "_protection": resourceSchema.BoolAttribute{
			Optional: true,
			Computed: true,
			Default:  booldefault.StaticBool(true),
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}
