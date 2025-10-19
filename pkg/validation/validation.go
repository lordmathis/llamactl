package validation

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

// ValidateStringForInjection checks if a string contains dangerous patterns
func ValidateStringForInjection(value string) error {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(value) {
			return ValidationError(fmt.Errorf("value contains potentially dangerous characters: %s", value))
		}
	}
	return nil
}

// ValidateStructStrings recursively validates all string fields in a struct
func ValidateStructStrings(v any, fieldPath string) error {
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
			if err := ValidateStringForInjection(field.String()); err != nil {
				return ValidationError(fmt.Errorf("field %s: %w", fieldName, err))
			}

		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				for j := 0; j < field.Len(); j++ {
					if err := ValidateStringForInjection(field.Index(j).String()); err != nil {
						return ValidationError(fmt.Errorf("field %s[%d]: %w", fieldName, j, err))
					}
				}
			}

		case reflect.Struct:
			if err := ValidateStructStrings(field.Interface(), fieldName); err != nil {
				return err
			}
		}
	}

	return nil
}

func ValidateInstanceName(name string) (string, error) {
	// Validate instance name
	if name == "" {
		return "", ValidationError(fmt.Errorf("name cannot be empty"))
	}
	if !validNamePattern.MatchString(name) {
		return "", ValidationError(fmt.Errorf("name contains invalid characters (only alphanumeric, hyphens, underscores allowed)"))
	}
	if len(name) > 50 {
		return "", ValidationError(fmt.Errorf("name too long (max 50 characters)"))
	}
	return name, nil
}
