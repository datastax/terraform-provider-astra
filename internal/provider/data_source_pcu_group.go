package provider

import (
	"context"

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
		Attributes: map[string]schema.Attribute{
			"pcu_group_id": schema.StringAttribute{
				Required: true,
			},
			"id": schema.StringAttribute{ // TODO should we bother taking id out since it's the same as pcu_group_id?
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
	}
}

func (d *pcuGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	var data pcuGroupDataSourceModel

	res.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if res.Diagnostics.HasError() {
		return
	}

	pcuGroup, status, err := GetPcuGroup(d.client, ctx, data.PCUGroupId)

	if status == 404 {
		res.Diagnostics.AddError("PCU Group Not Found", "No PCU Group found with the given ID")
		return
	}

	if err != nil {
		res.Diagnostics.AddError("Unable to Read PCU Group", err.Error())
		return
	}

	data.PcuGroupModel = *pcuGroup

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}
