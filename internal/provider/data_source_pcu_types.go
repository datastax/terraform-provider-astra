package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &pcuTypesDataSource{}
	_ datasource.DataSourceWithConfigure = &pcuTypesDataSource{}
)

func NewPCUTypesDataSource() datasource.DataSource {
	return &pcuTypesDataSource{}
}

type pcuTypesDataSource struct {
	BasePCUDataSource
}

type pcuTypesDataSourceModel struct {
	CloudProvider types.String   `tfsdk:"cloud_provider"`
	Region        types.String   `tfsdk:"region"`
	Results       []PcuTypeModel `tfsdk:"results"`
}

func (d *pcuTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, res *datasource.MetadataResponse) {
	res.TypeName = req.ProviderTypeName + "_pcu_types"
}

func (d *pcuTypesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, res *datasource.SchemaResponse) {
	res.Schema = schema.Schema{
		Description: "Retrieves a list of available PCU types. Can be filtered by cloud provider and region.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider": schema.StringAttribute{
				Optional:    true,
				Description: "Cloud provider to filter the PCU types by (e.g., aws, gcp, azure).",
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Region to filter the PCU types by.",
			},
			"results": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of available PCU types matching the specified criteria.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of the PCU (e.g., generalPurpose, cacheOptimized).",
						},
						"region": schema.StringAttribute{
							Computed:    true,
							Description: "The region of the PCU type.",
						},
						"cloud_provider": schema.StringAttribute{
							Computed:    true,
							Description: "The cloud provider of the PCU type.",
						},
						"details": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "Hardware details of the PCU type.",
							Attributes: map[string]schema.Attribute{
								"vcpu": schema.Int32Attribute{
									Computed:    true,
									Description: "Number of vCPUs.",
								},
								"memory": schema.StringAttribute{
									Computed:    true,
									Description: "Amount of memory.",
								},
								"disk_cache": schema.StringAttribute{
									Computed:    true,
									Description: "Amount of disk cache.",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *pcuTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, res *datasource.ReadResponse) {
	var data pcuTypesDataSourceModel

	diags := req.Config.Get(ctx, &data)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	results, diags := d.groups.FindTypes(ctx, data.CloudProvider, data.Region)
	if res.Diagnostics.Append(diags...); res.Diagnostics.HasError() {
		return
	}

	data.Results = results

	res.Diagnostics.Append(res.State.Set(ctx, &data)...)
}
