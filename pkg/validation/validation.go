package validation

import (
	"fmt"
	"regexp"

	"al.essio.dev/pkg/shellescape"
)

// Simple security validation that focuses only on actual injection risks
var (
	// Block shell metacharacters that could enable command injection
	dangerousPatterns = []*regexp.Regexp{
		regexp.MustCompile(`[;&|$` + "`" + `]`), // Shell metacharacters
		regexp.MustCompile(`\$\(.*\)`),          // Command substitution $(...)
		regexp.MustCompile("`.*`"),              // Command substitution backticks
		regexp.MustCompile(`[\x00-\x1F\x7F]`),   // Control characters (including newline, tab, null byte, etc.)
	}
)

type ValidationError error

// EscapeForShell returns the string properly quoted for safe use in shell commands.
// Uses shellescape.Quote to handle all special characters, metacharacters, and control characters.
func EscapeForShell(value string) string {
	return shellescape.Quote(value)
}

func ValidateInstanceName(name string) (string, error) {
	// Validate instance name
	if name == "" {
		return "", ValidationError(fmt.Errorf("name cannot be empty"))
	}
	if len(name) > 50 {
		return "", ValidationError(fmt.Errorf("name too long (max 50 characters)"))
	}
	return name, nil
}
