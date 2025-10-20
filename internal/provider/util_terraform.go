package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// UpdateTerraformObjectWithAttr adds an Attribute to a Terraform object
func UpdateTerraformObjectWithAttr(ctx context.Context, obj types.Object, key string, value attr.Value) (types.Object, diag.Diagnostics) {
	attrTypes := obj.AttributeTypes(ctx)
	attrValues := obj.Attributes()
	attrValues[key] = value
	return types.ObjectValue(attrTypes, attrValues)
}

func CompareTerraformAttrToString(attr attr.Value, s string) bool {
	if sAttr, ok := attr.(types.String); ok {
		return sAttr.ValueString() == s
	}
	return false
}

func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// MergeTerraformObjects combines two Terraform Objects replacing any null or unknown attribute values in `old` with
// matching attributes from `new`.  Object type attributes are handled recursively to avoid overwriting existing
// nested attributes in the old Object. Full type information must be specified.
//
// The reason for this function is to handle situations where a remote resource was created but not all configuration
// was performed successfully.  Instead of deleting the misconfigured resource, we can warn the user, and allow them
// to fix the configuration.  In the case of Pulsar namespaces, it's possible that a namespace has been created, but
// not all of the policy configuration was completed successfully.  If the user is warned of the issues, they can
// re-sync their remote state, and then decide how to proceed, either changing the configuration or deleting the namespace.
func MergeTerraformObjects(old, new types.Object, attributeTypes map[string]attr.Type) (types.Object, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	if attributeTypes == nil {
		diags.AddWarning("Failed to merge state objects", "No type information provided for object: "+old.String())
		return old, diags
	}
	if old.IsNull() || old.IsUnknown() {
		return basetypes.NewObjectValueMust(attributeTypes, new.Attributes()), diags
	}
	oldAttributes := old.Attributes()
	newAttributes := new.Attributes()
	attributes := map[string]attr.Value{}
	for name, newValue := range newAttributes {

		oldValue, ok := oldAttributes[name]
		if !ok || oldValue.IsNull() || oldValue.IsUnknown() {
			attributes[name] = newValue
			continue
		}

		if oldObjValue, ok := oldValue.(types.Object); ok {
			newObjValue, ok := newValue.(types.Object)
			if !ok {
				diags.AddWarning("Non matching types for attribute: "+name,
					fmt.Sprintf("Existing object attribute can't be replaced with type `%v`", newValue.Type(context.Background()).String()))
				continue
			}
			typeInfo, ok := attributeTypes[name].(types.ObjectType)
			if !ok {
				diags.AddWarning("Missing type information for attribute "+name, "No type information found when merging objects")
				continue
			}
			newObjValue, mergeDiags := MergeTerraformObjects(oldObjValue, newObjValue, typeInfo.AttributeTypes())
			diags.Append(mergeDiags...)
			if diags.HasError() {
				return old, diags
			}
			attributes[name] = newObjValue
			continue
		} else if _, ok := oldValue.(basetypes.MapValue); ok {
			newMapValue, ok := newValue.(basetypes.MapValue)
			if !ok {
				diags.AddWarning("Missing type information for attribute "+name, "No type information found when merging objects")
				continue
			}
			attributes[name] = newMapValue
			continue
		}

		attributes[name] = oldValue
	}

	return basetypes.NewObjectValue(attributeTypes, attributes)
}

func MergeTerraformAttributes(atts ...map[string]schema.Attribute) map[string]schema.Attribute {
	merged := map[string]schema.Attribute{}

	for _, attMap := range atts {
		for k, v := range attMap {
			merged[k] = v
		}
	}

	return merged
}

type responseWithStatus interface {
	Status() string
	StatusCode() int
}

func DiagErr(summary string, details string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	diags.AddError(summary, details)
	return diags
}

func ParsedHTTPResponseDiagErr(resp responseWithStatus, err error, summary string) diag.Diagnostics {
	if err != nil {
		return DiagErr(summary, err.Error())
	}

	if resp.StatusCode() >= 300 {
		body := extractBodyFromParsedResponse(resp)

		if body != nil {
			return DiagErr(summary, fmt.Sprintf("Received status code: '%v', with response: %s", resp.StatusCode(), string(body)))
		} else {
			return DiagErr(summary, fmt.Sprintf("Received status code: '%v'", resp.StatusCode()))
		}
	}

	return diag.Diagnostics{}
}

func extractBodyFromParsedResponse(v any) []byte {
	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	field := val.FieldByName("Body")
	if !field.IsValid() {
		return nil
	}

	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.Uint8 {
		return field.Bytes()
	}

	return nil
}

// HTTPResponseDiagErr takes an HTTP response and error and creates a Terraform Error Diagnostic if there is an error
func HTTPResponseDiagErr(resp *http.Response, err error, errorSummary string) diag.Diagnostics {
	if err != nil {
		return DiagErr(errorSummary, err.Error())
	}

	if resp != nil && resp.StatusCode >= 300 {
		bodyBytes, readErr := io.ReadAll(resp.Body)

		if readErr != nil {
			return DiagErr(errorSummary, fmt.Sprintf("Received status code: '%v', with error: %s", resp.StatusCode, readErr.Error()))
		} else {
			return DiagErr(errorSummary, fmt.Sprintf("Received status code: '%v', with message: %s", resp.StatusCode, string(bodyBytes)))
		}
	}

	return diag.Diagnostics{}
}

// HTTPResponseDiagErrWithBody takes an HTTP status code, body, and error and creates a Terraform Error Diagnostic if there is an error
func HTTPResponseDiagErrWithBody(statusCode int, body []byte, err error, errorSummary string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if err != nil {
		diags.AddError(errorSummary, err.Error())
	} else if statusCode >= 300 {
		details := fmt.Sprintf("Received status code: '%v', with message: %s", statusCode, body)
		diags.AddError(errorSummary, details)
	}
	return diags
}

// HTTPResponseDiagWarn takes an HTTP response and error and creates a Terraform Warn Diagnostic if there is an error
// or if the status code is not in the 2xx range
func HTTPResponseDiagWarn(resp *http.Response, err error, errorSummary string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if err != nil {
		diags.AddWarning(errorSummary, err.Error())
	} else if resp.StatusCode >= 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			details := fmt.Sprintf("Received status code: '%v', with error: %s", resp.StatusCode, err.Error())
			diags.AddWarning(errorSummary, details)
		} else {
			details := fmt.Sprintf("Received status code: '%v', with message: %s", resp.StatusCode, string(bodyBytes))
			diags.AddWarning(errorSummary, details)
		}
	}
	return diags
}

// HTTPResponseDiagWarnWithBody takes an HTTP status code, body, and error and creates a Terraform Error Diagnostic if there is an error
func HTTPResponseDiagWarnWithBody(statusCode int, body []byte, err error, errorSummary string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if err != nil {
		diags.AddWarning(errorSummary, err.Error())
	} else if statusCode >= 300 {
		details := fmt.Sprintf("Received status code: '%v', with message: %s", statusCode, body)
		diags.AddWarning(errorSummary, details)
	}
	return diags
}

// planModifierStringValueChanged is a terraform plan modifier function to use with 'RequiresReplaceIf' to check if a string value
// changed from one value to another, not including null values.
func planModifierStringValueChanged() stringplanmodifier.RequiresReplaceIfFunc {
	return func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
		if !req.StateValue.IsNull() && !req.ConfigValue.IsNull() && !req.StateValue.Equal(req.ConfigValue) {
			resp.RequiresReplace = true
		}
	}
}

type stringPlanModifierImpl struct {
	description string
	modifyPlan  func(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse)
}

func MkStringPlanModifier(description string, modifyPlan func(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse)) planmodifier.String {
	return &stringPlanModifierImpl{
		description: description,
		modifyPlan:  modifyPlan,
	}
}

func (m *stringPlanModifierImpl) Description(_ context.Context) string {
	return m.description
}

func (m *stringPlanModifierImpl) MarkdownDescription(_ context.Context) string {
	return m.description
}

func (m *stringPlanModifierImpl) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	m.modifyPlan(ctx, req, resp)
}

// planModifierRemoveDashes returns the configured string with all dashes removed
func planModifierRemoveDashes() planmodifier.String {
	return MkStringPlanModifier("Remove dashes from a string value", func(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
		// Do nothing if there is no planned value.
		if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
			return
		}

		// Do nothing if there is a no configuration value, otherwise interpolation gets messed up.
		if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
			return
		}

		resp.PlanValue = types.StringValue(strings.ReplaceAll(req.PlanValue.ValueString(), "-", ""))
	})
}

func planModifierSuppressDashesDiff() planmodifier.String {
	return MkStringPlanModifier("Suppress diffs if old and new values differ only in dashes", func(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
		if req.ConfigValue.IsNull() || req.StateValue.IsNull() {
			return
		}

		oldVal := req.StateValue.ValueString()
		newVal := req.ConfigValue.ValueString()

		if removeDashes(oldVal) == removeDashes(newVal) {
			// Suppress the diff by keeping the existing state value
			resp.PlanValue = req.StateValue
		}
	})
}
