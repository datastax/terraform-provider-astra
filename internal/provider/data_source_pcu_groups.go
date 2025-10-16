package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &pcuGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &pcuGroupsDataSource{}
)

func NewPCUGroupsDataSource() datasource.DataSource {
	return &pcuGroupsDataSource{}
}

type pcuGroupsDataSource struct {
	BasePCUDataSource
}

type pcuGroupsDataSourceModel struct {
	PCUGroupIds []types.String  `tfsdk:"pcu_group_ids"`
	PCUGroups   []PcuGroupModel `tfsdk:"pcu_groups"`
}

func (d *pcuGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_groups"
}

func (d *pcuGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"pcu_group_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"pcu_groups": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"org_id": schema.StringAttribute{
							Computed: true,
						},
						"title": schema.StringAttribute{
							Computed: true,
						},
						"cloud_provider": schema.StringAttribute{
							Computed: true,
						},
						"region": schema.StringAttribute{
							Computed: true,
						},
						"cache_type": schema.StringAttribute{
							Computed: true,
						},
						"provision_type": schema.StringAttribute{
							Computed: true,
						},
						"min_capacity": schema.Int64Attribute{
							Computed: true,
						},
						"max_capacity": schema.Int64Attribute{
							Computed: true,
						},
						"reserved_capacity": schema.Int64Attribute{
							Computed: true,
						},
						"description": schema.StringAttribute{
							Computed: true,
						},
						"created_at": schema.StringAttribute{
							Computed: true,
						},
						"updated_at": schema.StringAttribute{
							Computed: true,
						},
						"created_by": schema.StringAttribute{
							Computed: true,
						},
						"updated_by": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *pcuGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	var data pcuGroupsDataSourceModel

	res.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if res.Diagnostics.HasError() {
		return
	}

	pcuGroups, _, err := GetPcuGroups(d.client, ctx, data.PCUGroupIds)

	if err != nil {
		res.Diagnostics.AddError("Unable to Read PCU Groups", err.Error())
		return
	}

	data.PCUGroups = *pcuGroups
	fillPCUGroupIds(data) // even if the user provided IDs, this at least ensures the order is the same as the returned PCU groups

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}

func fillPCUGroupIds(data pcuGroupsDataSourceModel) {
	data.PCUGroupIds = make([]types.String, len(data.PCUGroups))

	for i, pcu := range data.PCUGroups {
		data.PCUGroupIds[i] = pcu.Id
	}
}
