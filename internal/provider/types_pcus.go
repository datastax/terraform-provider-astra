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
			Computed:    true,
			Description: "The unique identifier of the PCU group.",
		},
		PcuAttrOrgId: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The organization ID that owns this PCU group.",
		},
		PcuAttrTitle: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The user-defined title/name of the PCU group.",
		},
		PcuAttrCloudProvider: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The cloud provider where the PCU group is provisioned (e.g., AWS, GCP, Azure).",
		},
		PcuAttrRegion: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The cloud region where the PCU group is provisioned.",
		},
		PcuAttrCacheType: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The instance type/cache type for the PCU group.",
		},
		PcuAttrProvisionType: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The provisioning type for the PCU group (e.g., PROVISIONED, ON_DEMAND).",
		},
		PcuAttrMinCapacity: datasourceSchema.Int32Attribute{
			Computed:    true,
			Description: "The minimum capacity in PCUs for the group.",
		},
		PcuAttrMaxCapacity: datasourceSchema.Int32Attribute{
			Computed:    true,
			Description: "The maximum capacity in PCUs for the group.",
		},
		PcuAttrReservedCapacity: datasourceSchema.Int32Attribute{
			Computed:    true,
			Description: "The reserved capacity in PCUs for the group.",
		},
		PcuAttrDescription: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The user-defined description of the PCU group.",
		},
		PcuAttrCreatedAt: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "Timestamp when the PCU group was created.",
		},
		PcuAttrUpdatedAt: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "Timestamp when the PCU group was last updated.",
		},
		PcuAttrCreatedBy: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The user who created the PCU group.",
		},
		PcuAttrUpdatedBy: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The user who last updated the PCU group.",
		},
		PcuAttrStatus: datasourceSchema.StringAttribute{
			Computed:    true,
			Description: "The current status of the PCU group (e.g., ACTIVE, CREATING, TERMINATING).",
		},
	}
}

func MkPcuResourceCreatedUpdatedAttributes(updatePlanModifier planmodifier.String) map[string]resourceSchema.Attribute {
	return map[string]resourceSchema.Attribute{
		"created_at": resourceSchema.StringAttribute{
			Computed:    true,
			Description: "Timestamp when the PCU group was created.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"updated_at": resourceSchema.StringAttribute{
			Computed:    true,
			Description: "Timestamp when the PCU group was last updated.",
			PlanModifiers: []planmodifier.String{
				updatePlanModifier,
			},
		},
		"created_by": resourceSchema.StringAttribute{
			Computed:    true,
			Description: "The user who created the PCU group.",
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"updated_by": resourceSchema.StringAttribute{
			Computed:    true,
			Description: "The user who last updated the PCU group.",
			PlanModifiers: []planmodifier.String{
				updatePlanModifier,
			},
		},
	}
}

func MkPcuResourceProtectionAttribute(thing string) map[string]resourceSchema.Attribute {
	return map[string]resourceSchema.Attribute{
		thing + "_protection": resourceSchema.BoolAttribute{
			Optional:    true,
			Computed:    true,
			Default:     booldefault.StaticBool(true),
			Description: "When enabled, prevents accidental " + thing + " of the PCU group. Defaults to true.",
			PlanModifiers: []planmodifier.Bool{
				boolplanmodifier.UseStateForUnknown(),
			},
		},
	}
}
