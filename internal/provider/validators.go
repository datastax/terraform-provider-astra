package provider

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
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