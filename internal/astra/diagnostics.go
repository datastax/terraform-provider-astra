package astra

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// HTTPResponseDiagErr takes an HTTP response and error code and creates a Terraform Error Diagnostic if there is an error
func HTTPResponseDiagErr(resp *http.Response, err error, errorSummary string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if err != nil {
		diags.AddError(errorSummary, err.Error())
	} else if resp.StatusCode >= 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			diags.AddError(errorSummary, err.Error())
		} else {
			details := fmt.Sprintf("Received status code: '%v', with content: %s", resp.StatusCode, string(bodyBytes))
			diags.AddError(errorSummary, details)
		}
	}
	return diags
}

// HTTPResponseDiagWarn takes an HTTP response and error code and creates a Terraform Warn Diagnostic if there is an error
func HTTPResponseDiagWarn(resp *http.Response, err error, errorSummary string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	if err != nil {
		diags.AddWarning(errorSummary, err.Error())
	} else if resp.StatusCode >= 300 {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			diags.AddWarning(errorSummary, err.Error())
		} else {
			details := fmt.Sprintf("Received status code: '%v', with content: %s", resp.StatusCode, string(bodyBytes))
			diags.AddWarning(errorSummary, details)
		}
	}
	return diags
}
