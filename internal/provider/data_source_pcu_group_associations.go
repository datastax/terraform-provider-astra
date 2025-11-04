package provider

import (
	"context"

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
			PcuAttrGroupId: schema.StringAttribute{
				Required: true,
			},
			"results": schema.ListNestedAttribute{ // using "results" here to be consistent with other data sources
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						PcuAssocAttrDatacenterId: schema.StringAttribute{
							Computed: true,
						},
						PcuAssocAttrProvisioningStatus: schema.StringAttribute{
							Computed: true,
						},
						PcuAssocAttrCreatedAt: schema.StringAttribute{
							Computed: true,
						},
						PcuAssocAttrUpdatedAt: schema.StringAttribute{
							Computed: true,
						},
						PcuAssocAttrCreatedBy: schema.StringAttribute{
							Computed: true,
						},
						PcuAssocAttrUpdatedBy: schema.StringAttribute{
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

	diags := req.Config.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	associations, diags := d.associations.FindMany(ctx, data.PCUGroupId)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	if associations == nil {
		res.Diagnostics.AddError("PCU Group Associations Not Found", "No associations found for the given PCU Group ID.")
		return
	}

	data.Associations = *associations

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}
