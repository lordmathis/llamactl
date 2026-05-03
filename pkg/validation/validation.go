package validation

import (
	"fmt"
	"regexp"

	"al.essio.dev/pkg/shellescape"
)

var validInstanceName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type ValidationError error

// EscapeForShell returns the string properly quoted for safe use in shell commands.
// Uses shellescape.Quote to handle all special characters, metacharacters, and control characters.
func EscapeForShell(value string) string {
	return shellescape.Quote(value)
}

func ValidateInstanceName(name string) (string, error) {
	if name == "" {
		return "", ValidationError(fmt.Errorf("name cannot be empty"))
	}
	if len(name) > 50 {
		return "", ValidationError(fmt.Errorf("name too long (max 50 characters)"))
	}
	if !validInstanceName.MatchString(name) {
		return "", ValidationError(fmt.Errorf("name contains invalid characters (allowed: alphanumeric, dots, hyphens, underscores)"))
	}
	return name, nil
}
