package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ function.Function = &resolveDatacenterFunction{}
)

type resolveDatacenterFunction struct{}

func NewResolveDatacenterFunction() function.Function {
	return &resolveDatacenterFunction{}
}

func (f *resolveDatacenterFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "resolve_datacenter"
}

func (f *resolveDatacenterFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Description: "Resolves the datacenter ID for a given database and optional region (if the database has multiple regions). Helpful for components that require a datacenter ID such as PCU group associations and private links.",
		Parameters: []function.Parameter{
			function.ObjectParameter{
				Name:        "database",
				Description: "The database object to resolve the datacenter from. This should be the result of an \"astra_database\" resource or data source.",
				AttributeTypes: map[string]attr.Type{
					"cloud_provider": types.StringType,
					"datacenters": types.MapType{
						ElemType: types.StringType,
					},
				},
			},
		},
		VariadicParameter: function.StringParameter{
			Description: "The region to resolve the datacenter for. If not provided, the function will attempt to resolve the datacenter if there is only one configured.",
			Name:        "region",
		},
		Return: function.StringReturn{},
	}
}

type partialDb struct {
	CloudProvider string            `tfsdk:"cloud_provider"`
	Datacenters   map[string]string `tfsdk:"datacenters"`
}

func (f *resolveDatacenterFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var db partialDb
	var region []types.String

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &db, &region))

	switch len(region) {
	case 0:
		resp.Error = resolveSingleDatacenter(ctx, db, resp)
	case 1:
		resp.Error = resolveDatacenterForRegion(ctx, db, region, resp)
	default:
		resp.Error = function.NewArgumentFuncError(1, "Only one or zero regions should be provided")
	}
}

func resolveSingleDatacenter(ctx context.Context, db partialDb, resp *function.RunResponse) *function.FuncError {
	if len(db.Datacenters) == 1 {
		for _, dc := range db.Datacenters {
			return function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, dc))
		}
	}

	return function.NewArgumentFuncError(1, "Region is required when multiple datacenters are configured")
}

func resolveDatacenterForRegion(ctx context.Context, db partialDb, region []types.String, resp *function.RunResponse) *function.FuncError {
	key := strings.ToUpper(db.CloudProvider) + "." + strings.ToLower(region[0].ValueString())

	if dc, ok := db.Datacenters[key]; ok {
		return function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, dc))
	} else {
		return function.NewArgumentFuncError(1, "No datacenter found for region: "+region[0].ValueString())
	}
}
