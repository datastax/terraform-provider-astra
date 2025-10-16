package provider

import (
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
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	CreatedBy          types.String `tfsdk:"created_by"`
	UpdatedBy          types.String `tfsdk:"updated_by"`
}
