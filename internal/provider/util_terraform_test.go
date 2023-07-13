package provider

import (
	"log"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMergeTerraformObjects(t *testing.T) {
	attributeTypes := map[string]attr.Type{
		"foo": types.StringType,
		"bar": types.Int64Type,
	}

	oldAttrs := map[string]attr.Value{
		"foo": types.StringValue("foo-old"),
		"bar": types.Int64Unknown(),
	}
	oldValue, diags := types.ObjectValue(attributeTypes, oldAttrs)
	assertNoDiags(diags)

	newAttrs := map[string]attr.Value{
		"foo": types.StringValue("foo-new"),
		"bar": types.Int64Value(32),
	}
	newValue, diags := types.ObjectValue(attributeTypes, newAttrs)
	assertNoDiags(diags)

	mergedObj, diags := MergeTerraformObjects(oldValue, newValue, attributeTypes)
	assertNoDiags(diags)

	mergedAttrFoo := mergedObj.Attributes()["foo"].(types.String)
	if mergedAttrFoo.ValueString() != "foo-old" {
		t.Fatalf("expected foo-old, got: %v", mergedAttrFoo.ValueString())
	}

	mergedAttrBar := mergedObj.Attributes()["bar"].(types.Int64)
	if mergedAttrBar.String() != "32" {
		t.Fatalf("expected 32, got: %v", mergedAttrBar.String())
	}

}

func assertNoDiags(diags diag.Diagnostics) {
	if len(diags) > 0 {
		log.Fatalf("unexpected diagnotics: %v", diags)
	}
}
