package provider

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ignoreCase(_, old, new string, _ *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

func keyFromStrings(s []string) string {
	ss := make([]string, len(s))
	copy(ss, s)
	sort.Strings(ss)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(ss, "|"))))
}
