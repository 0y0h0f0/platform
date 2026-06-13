// Package xconfig provides shared configuration utilities used across all services.
package xconfig

import (
	"fmt"
	"os"
	"strings"

	"task-platform/pkg/xerr"
)

// ValidateSecret checks that a required secret meets minimum length and is not a
// known placeholder value. Use this at startup to fail fast on misconfiguration.
func ValidateSecret(name, value string, minLen int) error {
	if value == "" {
		return xerr.NewError(xerr.CodeFailedPrecondition, name+" is required")
	}
	for _, p := range placeholders {
		if value == p {
			return xerr.NewError(xerr.CodeFailedPrecondition, name+" must be changed from the default placeholder")
		}
	}
	if len(value) < minLen {
		return xerr.NewError(xerr.CodeFailedPrecondition, name+fmt.Sprintf(" must be at least %d characters", minLen))
	}
	return nil
}

// EnvOrDefault reads an environment variable or returns a fallback value.
func EnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// EnvList reads a comma-separated environment variable and returns the parts.
// Returns the default list when the variable is empty or unset.
func EnvList(key string, defaultVal []string) []string {
	if v := os.Getenv(key); v != "" {
		return strings.Split(v, ",")
	}
	return defaultVal
}

var placeholders = []string{
	"replace-with-a-long-random-secret",
	"replace-with-a-long-random-internal-token",
	"replace-with-a-long-random-secret-at-least-32-chars",
	"replace-with-a-long-random-internal-token-at-least-16-chars",
	"CHANGE_ME_MIN_32_CHARS_REQUIRED",
	"CHANGE_ME_MIN_16_CHARS_REQUIRED",
	"postgres",
	"admin",
}
