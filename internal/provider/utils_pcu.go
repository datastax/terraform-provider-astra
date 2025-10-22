package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type BasePCUConstruct struct {
	client       *astra.ClientWithResponses
	groups       PcuGroupsService
	associations PcuGroupAssociationsService
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

	client := providerData.(*astraClients2).astraClient

	b.client = client
	b.groups = &PcuGroupsServiceImpl{client}
	b.associations = &PcuGroupAssociationsServiceImpl{client}
}

type PcuGroupsService interface {
	Create(ctx context.Context, spec PcuGroupSpecModel) (*PcuGroupModel, diag.Diagnostics)
	FindOne(ctx context.Context, id types.String) (*PcuGroupModel, diag.Diagnostics)
	FindMany(ctx context.Context, ids []types.String) (*[]PcuGroupModel, diag.Diagnostics)
	Update(ctx context.Context, id types.String, spec PcuGroupSpecModel) (*PcuGroupModel, diag.Diagnostics)
	Park(ctx context.Context, id types.String) (*PcuGroupModel, diag.Diagnostics)
	Unpark(ctx context.Context, id types.String) (*PcuGroupModel, diag.Diagnostics)
	Delete(ctx context.Context, id types.String) diag.Diagnostics
	AwaitStatus(ctx context.Context, id types.String, target astra.PCUGroupStatus) (*PcuGroupModel, diag.Diagnostics)
}

type PcuGroupAssociationsService interface {
	Create(ctx context.Context, groupId types.String, datacenterId types.String) (*PcuGroupAssociationModel, diag.Diagnostics)
	FindOne(ctx context.Context, groupId types.String, datacenterId types.String) (*PcuGroupAssociationModel, diag.Diagnostics)
	FindMany(ctx context.Context, groupId types.String) (*[]PcuGroupAssociationModel, diag.Diagnostics)
	Transfer(ctx context.Context, fromGroupId, toGroupId types.String, datacenterId types.String) diag.Diagnostics
	Delete(ctx context.Context, groupId types.String, datacenterId types.String) diag.Diagnostics
}

type PcuGroupsServiceImpl struct {
	client *astra.ClientWithResponses
}

type PcuGroupAssociationsServiceImpl struct {
	client *astra.ClientWithResponses
}

func (s *PcuGroupsServiceImpl) Create(ctx context.Context, spec PcuGroupSpecModel) (*PcuGroupModel, diag.Diagnostics) {
	it := astra.InstanceType(spec.InstanceType.ValueString())   // todo should be optional
	pt := astra.ProvisionType(spec.ProvisionType.ValueString()) // todo should be optional

	reservedCap := int(spec.Reserved.ValueInt32()) // todo should be optional

	body := astra.PCUGroupCreateRequest{
		Title:         spec.Title.ValueString(),
		CloudProvider: astra.CloudProvider(strings.ToUpper(spec.CloudProvider.ValueString())),
		Region:        spec.Region.ValueString(),
		InstanceType:  it,
		ProvisionType: pt,
		Min:           int(spec.Min.ValueInt32()),
		Max:           int(spec.Max.ValueInt32()),
		Reserved:      reservedCap,
		Description:   spec.Description.ValueStringPointer(),
	}

	tflog.Debug(ctx, "Creating PCU group", map[string]interface{}{"body": body})

	res, err := s.client.PcuCreateWithResponse(ctx, astra.PcuCreateJSONRequestBody{body})

	if diags := ParsedHTTPResponseDiagErr(res, err, "failed to create PCU group"); diags.HasError() {
		return nil, diags
	}

	id := (*res.JSON201)[0].Uuid

	tflog.Debug(ctx, fmt.Sprintf("Created PCU group with ID: %s", *id))

	return s.AwaitStatus(ctx, types.StringPointerValue(id), astra.PCUGroupStatusCREATED)
}

func (s *PcuGroupsServiceImpl) FindOne(ctx context.Context, id types.String) (*PcuGroupModel, diag.Diagnostics) {
	groups, diags := s.FindMany(ctx, []types.String{id})

	if diags.HasError() {
		return nil, diags
	}

	if len(*groups) == 0 {
		return nil, diags
	}

	return &(*groups)[0], diags
}

func (s *PcuGroupsServiceImpl) FindMany(ctx context.Context, ids []types.String) (*[]PcuGroupModel, diag.Diagnostics) {
	nativeStrIds := make([]string, len(ids))

	for i, id := range ids {
		nativeStrIds[i] = id.ValueString()
	}

	resp, err := s.client.PcuGetWithResponse(ctx, astra.PCUGroupGetRequest{
		PcuGroupUUIDs: &nativeStrIds,
	})

	if diags := ParsedHTTPResponseDiagErr(resp, err, "error retrieving PCU groups"); diags.HasError() {
		return nil, diags
	}

	pcuGroups := make([]PcuGroupModel, 0) // don't want to return a nil b/c terraform should serialize it as [] and not null

	for _, rawPCU := range *resp.JSON200 {
		pcuGroups = append(pcuGroups, deserializePcuGroupFromAPI(rawPCU))
	}

	return &pcuGroups, nil
}

func (s *PcuGroupsServiceImpl) Update(ctx context.Context, id types.String, spec PcuGroupSpecModel) (*PcuGroupModel, diag.Diagnostics) {
	// TODO what if the PCU group doesn't exist in the first place? (what does the API return?)
	res, err := s.client.PcuUpdateWithResponse(ctx, astra.PcuUpdateJSONRequestBody{
		astra.PCUGroupUpdateRequest{
			PcuGroupUUID:  id.ValueString(),
			Title:         spec.Title.ValueString(),
			Description:   spec.Description.ValueStringPointer(),
			Min:           int(spec.Min.ValueInt32()),
			Max:           int(spec.Max.ValueInt32()),
			Reserved:      int(spec.Reserved.ValueInt32()),
			InstanceType:  astra.InstanceType(spec.InstanceType.ValueString()),
			ProvisionType: astra.ProvisionType(spec.ProvisionType.ValueString()),
		},
	})

	if diags := ParsedHTTPResponseDiagErr(res, err, "failed to update PCU group"); diags.HasError() {
		return nil, diags
	}

	deserialized := deserializePcuGroupFromAPI((*res.JSON200)[0])
	return &deserialized, nil
}

func (s *PcuGroupsServiceImpl) Park(ctx context.Context, id types.String) (*PcuGroupModel, diag.Diagnostics) {
	tflog.Debug(ctx, fmt.Sprintf("Parking PCU group %s", id.ValueString()))

	res, err := s.client.PcuGroupPark(ctx, id.ValueString())

	if diags := HTTPResponseDiagErr(res, err, "error parking pcu group"); diags.HasError() {
		return nil, diags
	}

	tflog.Debug(ctx, fmt.Sprintf("Parked request for PCU group %s accepted", id.ValueString()))

	return s.AwaitStatus(ctx, id, astra.PCUGroupStatusPARKED)
}

func (s *PcuGroupsServiceImpl) Unpark(ctx context.Context, id types.String) (*PcuGroupModel, diag.Diagnostics) {
	tflog.Debug(ctx, fmt.Sprintf("Unparking PCU group %s", id.ValueString()))

	res, err := s.client.PcuGroupUnpark(ctx, id.ValueString())

	if diags := HTTPResponseDiagErr(res, err, "error unparking pcu group"); diags.HasError() {
		return nil, diags
	}

	tflog.Debug(ctx, fmt.Sprintf("Unpark request for PCU group %s accepted", id.ValueString()))

	return s.AwaitStatus(ctx, id, astra.PCUGroupStatusCREATED)
}

func (s *PcuGroupsServiceImpl) Delete(ctx context.Context, id types.String) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting PCU group %s", id.ValueString()))

	res, err := s.client.PcuDelete(ctx, id.ValueString())

	if res != nil && res.StatusCode == 404 {
		return nil // whatever
	}

	if diags := HTTPResponseDiagErr(res, err, "error deleting PCU group"); diags.HasError() {
		return diags
	}

	tflog.Debug(ctx, fmt.Sprintf("PCU group %s deleted", id.ValueString()))

	return nil
}

func (s *PcuGroupsServiceImpl) AwaitStatus(ctx context.Context, id types.String, target astra.PCUGroupStatus) (*PcuGroupModel, diag.Diagnostics) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	attempts := 0

	tflog.Debug(ctx, fmt.Sprintf("Waiting for PCU group %s to reach status %s (attempt 0)", id.ValueString(), target))

	for {
		attempts += 1

		group, diags := s.FindOne(ctx, id)

		if diags.HasError() {
			return nil, diags
		}

		if group.Status.ValueString() == string(target) {
			tflog.Debug(ctx, fmt.Sprintf("PCU group %s reached status %s", id.ValueString(), target))
			return group, nil
		}

		tflog.Debug(ctx, fmt.Sprintf("Waiting for PCU group %s to reach status %s (attempt %d, currently %s)", id.ValueString(), target, attempts, group.Status.ValueString()))

		<-ticker.C
	}
}

func (s *PcuGroupAssociationsServiceImpl) Create(ctx context.Context, groupId types.String, datacenterId types.String) (*PcuGroupAssociationModel, diag.Diagnostics) {
	res, err := s.client.PcuAssociationCreate(ctx, groupId.ValueString(), datacenterId.ValueString())

	if diags := HTTPResponseDiagErr(res, err, "error creating PCU group association"); diags.HasError() {
		return nil, diags
	}

	return s.FindOne(ctx, groupId, datacenterId)
}

func (s *PcuGroupAssociationsServiceImpl) FindOne(ctx context.Context, groupId types.String, datacenterId types.String) (*PcuGroupAssociationModel, diag.Diagnostics) {
	associations, diags := s.FindMany(ctx, groupId)

	if diags.HasError() {
		return nil, diags
	}

	for _, assoc := range *associations {
		if assoc.DatacenterId.Equal(datacenterId) {
			return &assoc, nil
		}
	}

	return nil, diags
}

// FindMany TODO what's returned if the PCU group isn't found?
func (s *PcuGroupAssociationsServiceImpl) FindMany(ctx context.Context, groupId types.String) (*[]PcuGroupAssociationModel, diag.Diagnostics) {
	resp, err := s.client.PcuAssociationGetWithResponse(ctx, groupId.ValueString())

	if diags := ParsedHTTPResponseDiagErr(resp, err, "error retrieving PCU groups"); diags.HasError() {
		return nil, diags
	}

	associations := make([]PcuGroupAssociationModel, 0) // don't want to return a nil b/c terraform should serialize it as [] and not null

	for _, rawAssociation := range *resp.JSON200 {
		associations = append(associations, deserializePcuGroupAssociationFromAPI(rawAssociation))
	}

	return &associations, nil
}

// TODO: should deletion_protection also stop transferring the association?
// TODO: what's returned if the association didn't exist in the first place? (transferring is like deleting and recreating)
// TODO: what's the difference between transfer and delete+recreate?

func (s *PcuGroupAssociationsServiceImpl) Transfer(ctx context.Context, fromGroupId, toGroupId types.String, datacenterId types.String) diag.Diagnostics {
	res, err := s.client.PcuAssociationTransfer(ctx, astra.PcuAssociationTransferJSONRequestBody{
		FromPCUGroupUUID: fromGroupId.ValueStringPointer(),
		ToPCUGroupUUID:   toGroupId.ValueStringPointer(),
		DatacenterUUID:   datacenterId.ValueStringPointer(),
	})

	return HTTPResponseDiagErr(res, err, "error transferring PCU group association")
}

func (s *PcuGroupAssociationsServiceImpl) Delete(ctx context.Context, groupId types.String, datacenterId types.String) diag.Diagnostics {
	res, err := s.client.PcuAssociationDelete(ctx, groupId.ValueString(), datacenterId.ValueString())

	// TODO does it really return 404 lol
	if res != nil && res.StatusCode == 404 {
		return nil // whatever
	}

	return HTTPResponseDiagErr(res, err, "error deleting PCU group association")
}

func deserializePcuGroupFromAPI(rawPCU astra.PCUGroup) PcuGroupModel {
	return PcuGroupModel{
		Id:        types.StringPointerValue(rawPCU.Uuid),
		OrgId:     types.StringPointerValue(rawPCU.OrgId),
		CreatedAt: types.StringPointerValue(rawPCU.CreatedAt),
		UpdatedAt: types.StringPointerValue(rawPCU.UpdatedAt),
		CreatedBy: types.StringPointerValue(rawPCU.CreatedBy),
		UpdatedBy: types.StringPointerValue(rawPCU.UpdatedBy),
		Status:    StringEnumPtrToStrPtr(rawPCU.Status),
		PcuGroupSpecModel: PcuGroupSpecModel{
			Title:         types.StringPointerValue(rawPCU.Title),
			CloudProvider: StringEnumPtrToStrPtr(rawPCU.CloudProvider),
			Region:        types.StringPointerValue(rawPCU.Region),
			InstanceType:  StringEnumPtrToStrPtr(rawPCU.InstanceType),
			ProvisionType: StringEnumPtrToStrPtr(rawPCU.ProvisionType),
			Min:           IntPtrToTypeInt32Ptr(rawPCU.Min),
			Max:           IntPtrToTypeInt32Ptr(rawPCU.Max),
			Reserved:      IntPtrToTypeInt32Ptr(rawPCU.Reserved),
			Description:   types.StringPointerValue(rawPCU.Description),
		},
	}
}

func deserializePcuGroupAssociationFromAPI(rawAssociation astra.PCUAssociation) PcuGroupAssociationModel {
	return PcuGroupAssociationModel{
		DatacenterId:       types.StringPointerValue(rawAssociation.DatacenterUUID),
		ProvisioningStatus: StringEnumPtrToStrPtr(rawAssociation.ProvisioningStatus),
		CreatedAt:          types.StringPointerValue(rawAssociation.CreatedAt),
		UpdatedAt:          types.StringPointerValue(rawAssociation.UpdatedAt),
		CreatedBy:          types.StringPointerValue(rawAssociation.CreatedBy),
	}
}
