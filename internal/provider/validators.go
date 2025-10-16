package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

var keyspaceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_]{0,48}$`)
var roleResourcePrefix = "drn:astra:org:"

func validateKeyspace(v interface{}, path cty.Path) diag.Diagnostics {
	keyspaceName := v.(string)

	if !keyspaceNameRegex.MatchString(keyspaceName) {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Invalid keyspace name",
				Detail:        fmt.Sprintf("%s: invalid keyspace name - must match %s", keyspaceName, keyspaceNameRegex),
				AttributePath: path,
			},
		}
	}

	return nil
}

func validateRoleResources(v interface{}, path cty.Path) diag.Diagnostics {
	roleResource := v.(string)

	if !strings.HasPrefix(roleResource, roleResourcePrefix) {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Invalid role resource",
				Detail:        fmt.Sprintf("\"%s\": invalid role resource - must have prefix \"%s\"", roleResource, roleResourcePrefix),
				AttributePath: path,
			},
		}
	}
	return nil
}

func Int32IsGTE(path path.Path) validator.Int32 {
	return &int32IsGreaterThanValidator{path}
}

var _ validator.Int32 = &int32IsGreaterThanValidator{}

type int32IsGreaterThanValidator struct {
	path path.Path
}

func (v int32IsGreaterThanValidator) Description(_ context.Context) string {
	return fmt.Sprintf("If configured, must be greater than %s", v.path)
}

func (v int32IsGreaterThanValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v int32IsGreaterThanValidator) ValidateInt32(ctx context.Context, req validator.Int32Request, resp *validator.Int32Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var otherValue attr.Value

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.path, &otherValue)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if otherValue.IsNull() || otherValue.IsUnknown() {
		return
	}

	var otherInt types.Int32

	resp.Diagnostics.Append(tfsdk.ValueAs(ctx, otherValue, &otherInt)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if otherInt.ValueInt32() > req.ConfigValue.ValueInt32() {
		resp.Diagnostics.AddAttributeError(
			v.path,
			"Invalid Attribute Value",
			fmt.Sprintf("Must be less than or equal to %s value (%d)", req.Path, req.ConfigValue.ValueInt32()),
		)
	}
}
