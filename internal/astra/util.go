package astra

import (
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

func getEnvVarOrDefault(envVarName, defaultValue string) string {
	if v := os.Getenv(envVarName); v != "" {
		return v
	}
	return defaultValue
}

// getTestEnvVarOrSkip tries to get the given environment variable, otherwise skips the test
func getTestEnvVarOrSkip(t *testing.T, envVar string) string {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v
	}
	t.Skipf("skipping test due to missing %s environment variable", envVar)
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
