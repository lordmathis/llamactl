package backends

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ParseCommand parses a command string into a target struct
func ParseCommand(command string, executableNames []string, subcommandNames []string, multiValuedFlags map[string]bool, target any) error {
	// Normalize multiline commands
	command = normalizeCommand(command)
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Extract arguments
	args, err := extractArgs(command, executableNames, subcommandNames)
	if err != nil {
		return err
	}

	// Parse flags into map
	options, err := parseFlags(args, multiValuedFlags)
	if err != nil {
		return err
	}

	// Convert to target struct via JSON
	jsonData, err := json.Marshal(options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}

	return nil
}

// BuildCommandArgs converts a struct to command line arguments
func BuildCommandArgs(options any, multipleFlags map[string]bool) []string {
	var args []string

	v := reflect.ValueOf(options).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanInterface() {
			continue
		}

		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Get flag name from JSON tag
		flagName := strings.Split(jsonTag, ",")[0]
		flagName = strings.ReplaceAll(flagName, "_", "-")

		switch field.Kind() {
		case reflect.Bool:
			if field.Bool() {
				args = append(args, "--"+flagName)
			}
		case reflect.Int:
			if field.Int() != 0 {
				args = append(args, "--"+flagName, strconv.FormatInt(field.Int(), 10))
			}
		case reflect.Float64:
			if field.Float() != 0 {
				args = append(args, "--"+flagName, strconv.FormatFloat(field.Float(), 'f', -1, 64))
			}
		case reflect.String:
			if field.String() != "" {
				args = append(args, "--"+flagName, field.String())
			}
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String && field.Len() > 0 {
				if multipleFlags[flagName] {
					// Multiple flags: --flag value1 --flag value2
					for j := 0; j < field.Len(); j++ {
						args = append(args, "--"+flagName, field.Index(j).String())
					}
				} else {
					// Comma-separated: --flag value1,value2
					var values []string
					for j := 0; j < field.Len(); j++ {
						values = append(values, field.Index(j).String())
					}
					args = append(args, "--"+flagName, strings.Join(values, ","))
				}
			}
		}
	}

	return args
}

// normalizeCommand handles multiline commands with backslashes
func normalizeCommand(command string) string {
	re := regexp.MustCompile(`\\\s*\n\s*`)
	normalized := re.ReplaceAllString(command, " ")
	re = regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(normalized, " "))
}

// extractArgs extracts arguments from command, removing executable and subcommands
func extractArgs(command string, executableNames []string, subcommandNames []string) ([]string, error) {
	// Check for unterminated quotes
	if strings.Count(command, `"`)%2 != 0 || strings.Count(command, `'`)%2 != 0 {
		return nil, fmt.Errorf("unterminated quoted string")
	}

	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens found")
	}

	// Skip executable
	start := 0
	firstToken := tokens[0]

	// Check for executable name (with or without path)
	if strings.Contains(firstToken, string(filepath.Separator)) {
		baseName := filepath.Base(firstToken)
		for _, execName := range executableNames {
			if strings.HasSuffix(strings.ToLower(baseName), strings.ToLower(execName)) {
				start = 1
				break
			}
		}
	} else {
		for _, execName := range executableNames {
			if strings.EqualFold(firstToken, execName) {
				start = 1
				break
			}
		}
	}

	// Skip subcommand if present
	if start < len(tokens) {
		for _, subCmd := range subcommandNames {
			if strings.EqualFold(tokens[start], subCmd) {
				start++
				break
			}
		}
	}

	// Handle case where command starts with subcommand (no executable)
	if start == 0 {
		for _, subCmd := range subcommandNames {
			if strings.EqualFold(firstToken, subCmd) {
				start = 1
				break
			}
		}
	}

	return tokens[start:], nil
}

// parseFlags parses command line flags into a map
func parseFlags(args []string, multiValuedFlags map[string]bool) (map[string]any, error) {
	options := make(map[string]any)

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !strings.HasPrefix(arg, "-") {
			continue
		}

		// Check for malformed flags (more than two leading dashes)
		if strings.HasPrefix(arg, "---") {
			return nil, fmt.Errorf("malformed flag: %s", arg)
		}

		// Get flag name and value
		var flagName, value string
		var hasValue bool

		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flagName = strings.TrimLeft(parts[0], "-")
			value = parts[1]
			hasValue = true
		} else {
			flagName = strings.TrimLeft(arg, "-")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				hasValue = true
				i++ // Skip next arg since we consumed it
			}
		}

		// Convert kebab-case to snake_case for JSON
		flagName = strings.ReplaceAll(flagName, "-", "_")

		if hasValue {
			// Handle multi-valued flags
			if multiValuedFlags[flagName] {
				if existing, ok := options[flagName].([]string); ok {
					options[flagName] = append(existing, value)
				} else {
					options[flagName] = []string{value}
				}
			} else {
				options[flagName] = parseValue(value)
			}
		} else {
			// Boolean flag
			options[flagName] = true
		}
	}

	return options, nil
}

// parseValue converts string to appropriate type
func parseValue(value string) any {
	// Remove quotes
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	// Try boolean
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	}

	// Try integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Return as string
	return value
}
