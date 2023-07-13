package util

import (
	"math/rand"
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
