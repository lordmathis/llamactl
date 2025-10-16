package validation

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
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

// ValidateInstanceOptions performs validation based on backend type
func ValidateInstanceOptions(options *instance.Options) error {
	if options == nil {
		return ValidationError(fmt.Errorf("options cannot be nil"))
	}

	// Validate based on backend type
	switch options.BackendType {
	case backends.BackendTypeLlamaCpp:
		return validateLlamaCppOptions(options)
	case backends.BackendTypeMlxLm:
		return validateMlxOptions(options)
	case backends.BackendTypeVllm:
		return validateVllmOptions(options)
	default:
		return ValidationError(fmt.Errorf("unsupported backend type: %s", options.BackendType))
	}
}

// validateLlamaCppOptions validates llama.cpp specific options
func validateLlamaCppOptions(options *instance.Options) error {
	if options.LlamaServerOptions == nil {
		return ValidationError(fmt.Errorf("llama server options cannot be nil for llama.cpp backend"))
	}

	// Use reflection to check all string fields for injection patterns
	if err := validateStructStrings(options.LlamaServerOptions, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if options.LlamaServerOptions.Port < 0 || options.LlamaServerOptions.Port > 65535 {
		return ValidationError(fmt.Errorf("invalid port range: %d", options.LlamaServerOptions.Port))
	}

	return nil
}

// validateMlxOptions validates MLX backend specific options
func validateMlxOptions(options *instance.Options) error {
	if options.MlxServerOptions == nil {
		return ValidationError(fmt.Errorf("MLX server options cannot be nil for MLX backend"))
	}

	if err := validateStructStrings(options.MlxServerOptions, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if options.MlxServerOptions.Port < 0 || options.MlxServerOptions.Port > 65535 {
		return ValidationError(fmt.Errorf("invalid port range: %d", options.MlxServerOptions.Port))
	}

	return nil
}

// validateVllmOptions validates vLLM backend specific options
func validateVllmOptions(options *instance.Options) error {
	if options.VllmServerOptions == nil {
		return ValidationError(fmt.Errorf("vLLM server options cannot be nil for vLLM backend"))
	}

	// Use reflection to check all string fields for injection patterns
	if err := validateStructStrings(options.VllmServerOptions, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if options.VllmServerOptions.Port < 0 || options.VllmServerOptions.Port > 65535 {
		return ValidationError(fmt.Errorf("invalid port range: %d", options.VllmServerOptions.Port))
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
