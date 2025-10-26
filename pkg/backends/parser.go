package backends

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// parseCommand parses a command string into a target struct
func parseCommand(command string, executableNames []string, subcommandNames []string, multiValuedFlags map[string]struct{}, target any) error {
	// Normalize multiline commands
	command = normalizeCommand(command)
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Extract arguments and positional model
	args, modelFromPositional, err := extractArgs(command, executableNames, subcommandNames)
	if err != nil {
		return err
	}

	// Parse flags into map
	options, err := parseFlags(args, multiValuedFlags)
	if err != nil {
		return err
	}

	// If we found a positional model and no --model flag was provided, set the model
	if modelFromPositional != "" {
		if _, hasModelFlag := options["model"]; !hasModelFlag {
			options["model"] = modelFromPositional
		}
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

// normalizeCommand handles multiline commands with backslashes
func normalizeCommand(command string) string {
	re := regexp.MustCompile(`\\\s*\n\s*`)
	normalized := re.ReplaceAllString(command, " ")
	re = regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(normalized, " "))
}

// extractArgs extracts arguments from command, removing executable and subcommands
// Returns: args, modelFromPositional, error
func extractArgs(command string, executableNames []string, subcommandNames []string) ([]string, string, error) {
	// Check for unterminated quotes
	if strings.Count(command, `"`)%2 != 0 || strings.Count(command, `'`)%2 != 0 {
		return nil, "", fmt.Errorf("unterminated quoted string")
	}

	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return nil, "", fmt.Errorf("no tokens found")
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

	args := tokens[start:]

	// Extract first positional argument (model) if present and not a flag
	var modelFromPositional string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		modelFromPositional = args[0]
		args = args[1:] // Remove the model from args to process remaining flags
	}

	return args, modelFromPositional, nil
}

// parseFlags parses command line flags into a map
func parseFlags(args []string, multiValuedFlags map[string]struct{}) (map[string]any, error) {
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
			if _, isMultiValue := multiValuedFlags[flagName]; isMultiValue {
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
