package provider

import (
	"context"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BasePCUConstruct struct {
	client *astra.ClientWithResponses
}

type BasePCUDataSource struct {
	BasePCUConstruct
}

type BasePCUResource struct {
	BasePCUConstruct
}

func (b *BasePCUDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	b.configure(req.ProviderData)
}

func (b *BasePCUResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	b.configure(req.ProviderData)
}

func (b *BasePCUConstruct) configure(providerData any) {
	if providerData == nil {
		return
	}
	b.client = providerData.(*astraClients2).astraClient
}

func GetPcuGroups(client *astra.ClientWithResponses, ctx context.Context, rawIds []types.String) (*[]PcuGroupModel, diag.Diagnostics) {
	ids := make([]string, len(rawIds))

	for i, id := range rawIds {
		ids[i] = id.ValueString()
	}

	resp, err := client.PcuGetWithResponse(ctx, astra.PCUGroupGetRequest{
		PcuGroupUUIDs: &ids,
	})

	if diags := ParsedHTTPResponseDiagErr(resp, err, "error retrieving PCU groups"); diags.HasError() {
		return nil, diags
	}

	pcuGroups := make([]PcuGroupModel, 0) // don't want to return a nil b/c terraform should serialize it as [] and not null

	for _, rawPCU := range *resp.JSON200 {
		pcuGroups = append(pcuGroups, DeserializePcuGroupFromAPI(rawPCU))
	}

	return &pcuGroups, nil
}

func GetPcuGroup(client *astra.ClientWithResponses, ctx context.Context, rawId types.String) (*PcuGroupModel, diag.Diagnostics) {
	groups, diags := GetPcuGroups(client, ctx, []types.String{rawId})

	if diags.HasError() {
		return nil, diags
	}

	if len(*groups) == 0 {
		return nil, diags
	}

	return &(*groups)[0], diags
}

func DeserializePcuGroupFromAPI(rawPCU astra.PCUGroup) PcuGroupModel {
	return PcuGroupModel{
		Id:        types.StringPointerValue(rawPCU.Uuid),
		OrgId:     types.StringPointerValue(rawPCU.OrgId),
		CreatedAt: types.StringPointerValue(rawPCU.CreatedAt),
		UpdatedAt: types.StringPointerValue(rawPCU.UpdatedAt),
		CreatedBy: types.StringPointerValue(rawPCU.CreatedBy),
		UpdatedBy: types.StringPointerValue(rawPCU.UpdatedBy),
		Status:    stringEnumPtrToStrPtr(rawPCU.Status),
		PcuGroupSpecModel: PcuGroupSpecModel{
			Title:         types.StringPointerValue(rawPCU.Title),
			CloudProvider: stringEnumPtrToStrPtr(rawPCU.CloudProvider),
			Region:        types.StringPointerValue(rawPCU.Region),
			InstanceType:  stringEnumPtrToStrPtr(rawPCU.InstanceType),
			ProvisionType: stringEnumPtrToStrPtr(rawPCU.ProvisionType),
			Min:           intPtrToTypeInt32Ptr(rawPCU.Min),
			Max:           intPtrToTypeInt32Ptr(rawPCU.Max),
			Reserved:      intPtrToTypeInt32Ptr(rawPCU.Reserved),
			Description:   types.StringPointerValue(rawPCU.Description),
		},
	}
}

// GetPcuGroupAssociations TODO what's returned if the PCU group isn't found?
func GetPcuGroupAssociations(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string) (*[]PcuGroupAssociationModel, diag.Diagnostics) {
	resp, err := client.PcuAssociationGetWithResponse(ctx, pcuGroupId)

	if diags := ParsedHTTPResponseDiagErr(resp, err, "error retrieving PCU groups"); diags.HasError() {
		return nil, diags
	}

	associations := make([]PcuGroupAssociationModel, 0) // don't want to return a nil b/c terraform should serialize it as [] and not null

	for _, rawAssociation := range *resp.JSON200 {
		associations = append(associations, deserializePcuGroupAssociation(rawAssociation))
	}

	return &associations, nil
}

//func MergePcuGroupModels(base PcuGroupModel, override PcuGroupModel) PcuGroupModel {
//	baseCopy := PcuGroupModel{
//		Id:        base.Id,
//		OrgId:     base.OrgId,
//		CreatedAt: base.CreatedAt,
//		UpdatedAt: base.UpdatedAt,
//		CreatedBy: base.CreatedBy,
//		UpdatedBy: base.UpdatedBy,
//		Status:    base.Status,
//		PcuGroupSpecModel: PcuGroupSpecModel{
//			Title:         base.Title,
//			CloudProvider: base.CloudProvider,
//			Region:        base.Region,
//			InstanceType:  base.InstanceType,
//			ProvisionType: base.ProvisionType,
//			Min:           base.Min,
//			Max:           base.Max,
//			Reserved:      base.Reserved,
//			Description:   base.Description,
//		},
//	}
//
//	if !override.Title.IsNull() {
//		baseCopy.Title = override.Title
//	}
//
//	if !override.Description.IsNull() {
//		baseCopy.Description = override.Description
//	}
//
//	if !override.InstanceType.IsNull() {
//		baseCopy.InstanceType = override.InstanceType
//	}
//
//	if !override.ProvisionType.IsNull() {
//		baseCopy.ProvisionType = override.ProvisionType
//	}
//
//	return baseCopy
//}

func stringEnumPtrToStrPtr[E ~string](value *E) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(string(*value))
}

func intPtrToTypeInt32Ptr(value *int) types.Int32 {
	if value == nil {
		return types.Int32Null()
	}
	return types.Int32Value(int32(*value))
}

func deserializePcuGroupAssociation(rawAssociation astra.PCUAssociation) PcuGroupAssociationModel {
	return PcuGroupAssociationModel{
		DatacenterId:       types.StringValue(*rawAssociation.DatacenterUUID),
		ProvisioningStatus: types.StringValue(string(*rawAssociation.ProvisioningStatus)),
		CreatedAt:          types.StringValue(*rawAssociation.CreatedAt),
		UpdatedAt:          types.StringValue(*rawAssociation.UpdatedAt),
		CreatedBy:          types.StringValue(*rawAssociation.CreatedBy),
	}
}
