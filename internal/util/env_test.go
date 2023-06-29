package util

import (
	"os"
	"strings"
	"testing"
)

// getTestEnvVarOrSkip tries to get the given environment variable, otherwise skips the test
func getTestEnvVarOrSkip(t *testing.T, envVar string) string {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v
	}
	t.Skipf("skipping test due to missing %s environment variable", envVar)
	return ""
}
