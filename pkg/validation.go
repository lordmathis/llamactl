package llamactl

import (
	"fmt"
	"reflect"
	"regexp"
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

	// Simple validation for instance names
	validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

type ValidationError error

// validateStringForInjection checks if a string contains dangerous patterns
func validateStringForInjection(value string) error {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(value) {
			return ValidationError(fmt.Errorf("value contains potentially dangerous characters: %s", value))
		}
	}
	return nil
}

// ValidateInstanceOptions performs minimal security validation
func ValidateInstanceOptions(options *CreateInstanceOptions) error {
	if options == nil {
		return ValidationError(fmt.Errorf("options cannot be nil"))
	}

	// Use reflection to check all string fields for injection patterns
	if err := validateStructStrings(&options.LlamaServerOptions, ""); err != nil {
		return err
	}

	// Basic network validation - only check for reasonable ranges
	if options.Port < 0 || options.Port > 65535 {
		return ValidationError(fmt.Errorf("invalid port range"))
	}

	return nil
}

// validateStructStrings recursively validates all string fields in a struct
func validateStructStrings(v any, fieldPath string) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		fieldName := fieldType.Name
		if fieldPath != "" {
			fieldName = fieldPath + "." + fieldName
		}

		switch field.Kind() {
		case reflect.String:
			if err := validateStringForInjection(field.String()); err != nil {
				return ValidationError(fmt.Errorf("field %s: %w", fieldName, err))
			}

		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					if err := validateStringForInjection(field.Index(j).String()); err != nil {
						return ValidationError(fmt.Errorf("field %s[%d]: %w", fieldName, j, err))
					}
				}
			}

		case reflect.Struct:
			if err := validateStructStrings(field.Interface(), fieldName); err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateInstanceName(name string) error {
	// Validate instance name
	if name == "" {
		return ValidationError(fmt.Errorf("name cannot be empty"))
	}
	if !validNamePattern.MatchString(name) {
		return ValidationError(fmt.Errorf("name contains invalid characters (only alphanumeric, hyphens, underscores allowed)"))
	}
	if len(name) > 50 {
		return ValidationError(fmt.Errorf("name too long (max 50 characters)"))
	}
	return nil
}
