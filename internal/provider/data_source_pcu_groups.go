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
			PcuAttrGroupIds: schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
			"results": schema.ListNestedAttribute{ // using "results" here to be consistent with other data sources
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: MkPcuGroupDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *pcuGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	var data pcuGroupsDataSourceModel

	diags := req.Config.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	groups, diags := d.groups.FindMany(ctx, data.PCUGroupIds)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	data.PCUGroups = *groups
	fillPCUGroupIds(data) // even if the user provided IDs, this at least ensures the order is the same as the returned PCU groups

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}

func fillPCUGroupIds(data pcuGroupsDataSourceModel) {
	data.PCUGroupIds = make([]types.String, len(data.PCUGroups))

	for i, pcu := range data.PCUGroups {
		data.PCUGroupIds[i] = pcu.Id
	}
}
