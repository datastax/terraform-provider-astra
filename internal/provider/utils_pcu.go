package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/datastax/astra-client-go/v2/astra"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
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

func GetPcuGroups(client *astra.ClientWithResponses, ctx context.Context, rawIds []types.String) (*[]PcuGroupModel, int, error) {
	ids := make([]string, len(rawIds))

	for i, id := range rawIds {
		ids[i] = id.ValueString()
	}

	getPCUsResponse, err := client.PcuGetWithResponse(ctx, astra.PCUGroupGetRequest{
		PcuGroupUUIDs: &ids,
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve PCU Groups: %w", err)
	}

	if getPCUsResponse.HTTPResponse.StatusCode != http.StatusOK {
		return nil, getPCUsResponse.HTTPResponse.StatusCode, fmt.Errorf("failed to retrieve PCU Groups (HTTP %s)", getPCUsResponse.HTTPResponse.Status)
	}

	pcuGroups := make([]PcuGroupModel, 0) // don't want to return a nil b/c terraform should serialize it as [] and not null

	for _, rawPCU := range *getPCUsResponse.JSON200 {
		pcuGroups = append(pcuGroups, DeserializePcuGroupFromAPI(rawPCU))
	}

	return &pcuGroups, getPCUsResponse.HTTPResponse.StatusCode, nil
}

func GetPcuGroup(client *astra.ClientWithResponses, ctx context.Context, rawId types.String) (*PcuGroupModel, int, error) {
	pcuGroups, status, err := GetPcuGroups(client, ctx, []types.String{rawId})

	if err != nil {
		return nil, status, err
	}

	// simulating a 404 just for convenience ¯\_(ツ)_/¯
	// sorry, not sorry
	if len(*pcuGroups) == 0 {
		return nil, 404, fmt.Errorf("PCU group with id %s not found", rawId.ValueString())
	}

	return &(*pcuGroups)[0], status, nil
}

func DeserializePcuGroupFromAPI(rawPCU astra.PCUGroup) PcuGroupModel {
	return PcuGroupModel{
		Id:        types.StringValue(*rawPCU.Uuid),
		OrgId:     types.StringValue(*rawPCU.OrgId),
		CreatedAt: types.StringValue(*rawPCU.CreatedAt),
		UpdatedAt: types.StringValue(*rawPCU.UpdatedAt),
		CreatedBy: types.StringValue(*rawPCU.CreatedBy),
		UpdatedBy: types.StringValue(*rawPCU.UpdatedBy),
		Status:    types.StringValue(string(*rawPCU.Status)),
		PcuGroupSpecModel: PcuGroupSpecModel{
			Title:         types.StringValue(*rawPCU.Title),
			CloudProvider: types.StringValue(string(*rawPCU.CloudProvider)),
			Region:        types.StringValue(*rawPCU.Region),
			InstanceType:  types.StringValue(string(*rawPCU.InstanceType)),
			ProvisionType: types.StringValue(string(*rawPCU.ProvisionType)),
			Min:           types.Int32Value(int32(*rawPCU.Min)),
			Max:           types.Int32Value(int32(*rawPCU.Max)),
			Reserved:      types.Int32Value(int32(*rawPCU.Reserved)),
			Description:   types.StringValue(*rawPCU.Description),
		},
	}
}

func GetPcuGroupAssociations(client *astra.ClientWithResponses, ctx context.Context, pcuGroupId string) (*[]PcuGroupAssociationModel, int, error) {
	getAssociationsResponse, err := client.PcuAssociationGetWithResponse(ctx, pcuGroupId)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve PCU Group Associations: %w", err)
	}

	if getAssociationsResponse.HTTPResponse.StatusCode != http.StatusOK {
		return nil, getAssociationsResponse.HTTPResponse.StatusCode, fmt.Errorf("failed to retrieve PCU Group Associations (HTTP %s)", getAssociationsResponse.HTTPResponse.Status)
	}

	associations := make([]PcuGroupAssociationModel, 0) // don't want to return a nil b/c terraform should serialize it as [] and not null

	for _, rawAssociation := range *getAssociationsResponse.JSON200 {
		associations = append(associations, deserializePcuGroupAssociation(rawAssociation))
	}

	return &associations, getAssociationsResponse.HTTPResponse.StatusCode, nil
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
