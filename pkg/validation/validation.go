package validation

import (
	"fmt"
	"regexp"
)

var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type ValidationError error

func ValidateInstanceName(name string) (string, error) {
	// Validate instance name
	if name == "" {
		return "", ValidationError(fmt.Errorf("name cannot be empty"))
	}
	if !validNamePattern.MatchString(name) {
		return "", ValidationError(fmt.Errorf("name contains invalid characters (only alphanumeric, periods, hyphens, underscores allowed)"))
	}
	if len(name) > 50 {
		return "", ValidationError(fmt.Errorf("name too long (max 50 characters)"))
	}
	return name, nil
}
