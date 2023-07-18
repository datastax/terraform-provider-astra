package provider

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

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

func protectedFromDelete(resourceData *schema.ResourceData) bool {
	return resourceData.Get("deletion_protection").(bool)
}

// checkRequiredTestVars returns true if the given environment variables are not empty
func checkRequiredTestVars(t *testing.T, vars ...string) {
	for _, v := range vars {
		if strings.TrimSpace(os.Getenv(v)) == "" {
			t.Skipf("skipping test due to missing %s environment variable", v)
		}
	}
}

// firstNonEmptyString returns the first non-empty string from the given list or a default value
func firstNonEmptyString(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// randomString returns a random string of length n
func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var chars = []rune("0123456789abcdefghijklmnopqrstuvwxyz")
	s := make([]rune, n)
	for i := range s {
		s[i] = chars[rand.Intn(len(chars))]
	}
	return string(s)
}

// envVarOrDefault returns the value in the given environment variable or a default value
func envVarOrDefault(envVar, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v
	}
	return defaultValue
}
