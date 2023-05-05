package common

import (
	"math/rand"
	"os"
	"strings"
	"time"
)

// FirstNonEmptyString returns the first non-empty string from the given list or a default value
func FirstNonEmptyString(s ...string) string {
	for _, v := range s {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// EnvVarOrDefault returns the value in the given environment variable or a default value
func EnvVarOrDefault(envVar, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(envVar)); v != "" {
		return v
	}
	return defaultValue
}

// RandomString returns a random string of length n
func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var chars = []rune("0123456789abcdefghijklmnopqrstuvwxyz")
	s := make([]rune, n)
	for i := range s {
		s[i] = chars[rand.Intn(len(chars))]
	}
	return string(s)
}
