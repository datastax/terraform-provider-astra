package provider

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &pcuGroupAssociationsDataSource{}
	_ datasource.DataSourceWithConfigure = &pcuGroupAssociationsDataSource{}
)

func NewPCUGroupAssociationsDataSource() datasource.DataSource {
	return &pcuGroupAssociationsDataSource{}
}

type pcuGroupAssociationsDataSource struct {
	BasePCUDataSource
}

type pcuGroupAssociationsDataSourceModel struct {
	PCUGroupId   types.String               `tfsdk:"pcu_group_id"`
	Associations []PcuGroupAssociationModel `tfsdk:"associations"`
}

func (d *pcuGroupAssociationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_group_associations"
}

func (d *pcuGroupAssociationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"pcu_group_id": schema.StringAttribute{
				Required: true,
			},
			"associations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"datacenter_id": schema.StringAttribute{
							Computed: true,
						},
						"provisioning_status": schema.StringAttribute{
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
					},
				},
			},
		},
	}
}

func (d *pcuGroupAssociationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	var data pcuGroupAssociationsDataSourceModel

	res.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if res.Diagnostics.HasError() {
		return
	}

	associations, status, err := GetPcuGroupAssociations(d.client, ctx, data.PCUGroupId.ValueString())

	if status == http.StatusNotFound {
		res.Diagnostics.AddError("PCU Group Not Found", "The specified PCU Group could not be found.")
		return
	}

	if err != nil {
		res.Diagnostics.AddError("Error Reading PCU Group Associations", err.Error())
		return
	}

	data.Associations = *associations

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}
