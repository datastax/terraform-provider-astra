package util

import (
	"os"
	"strings"
)

// EnvVarOrDefault returns the value in the given environment variable or a default value
func EnvVarOrDefault(envVar, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v
	}
	return defaultValue
}
