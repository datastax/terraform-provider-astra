package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &pcuGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &pcuGroupDataSource{}
)

func NewPCUGroupDataSource() datasource.DataSource {
	return &pcuGroupDataSource{}
}

type pcuGroupDataSource struct {
	BasePCUDataSource
}

type pcuGroupDataSourceModel struct {
	PCUGroupId types.String `tfsdk:"pcu_group_id"`
	PcuGroupModel
}

func (d *pcuGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group"
}

func (d *pcuGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Retrieves details for a specific PCU (Provisioned Capacity Units) group by its ID.",
		Attributes: MergeMaps(
			map[string]schema.Attribute{
				PcuAttrGroupId: schema.StringAttribute{
					Required:    true,
					Description: "The unique identifier of the PCU group to retrieve.",
				},
			},
			MkPcuGroupDataSourceAttributes(),
		),
	}
}

func (d *pcuGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	var data pcuGroupDataSourceModel

	diags := req.Config.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	group, diags := d.groups.FindOne(ctx, data.PCUGroupId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if group == nil {
		res.Diagnostics.AddError("Could not find PCU Group", fmt.Sprintf("PCU Group with ID %s was not found", data.PCUGroupId.ValueString()))
		return
	}

	data.PcuGroupModel = *group

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}
