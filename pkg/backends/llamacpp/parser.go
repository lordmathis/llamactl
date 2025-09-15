package llamacpp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseLlamaCommand parses a llama-server command string into LlamaServerOptions
func ParseLlamaCommand(command string) (*LlamaServerOptions, error) {
	// 1. Validate command starts with llama-server
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Check if command starts with llama-server (case-insensitive)
	lowerCommand := strings.ToLower(trimmed)
	if !strings.HasPrefix(lowerCommand, "llama-server") {
		return nil, fmt.Errorf("command must start with 'llama-server'")
	}

	// 2. Extract arguments (everything after llama-server)
	parts := strings.Fields(trimmed)
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid command format")
	}

	args := parts[1:] // Skip binary name

	// 3. Parse arguments into map
	options := make(map[string]any)
	i := 0
	for i < len(args) {
		arg := args[i]

		// Skip non-flag arguments
		if !strings.HasPrefix(arg, "-") {
			i++
			continue
		}

		// Handle --flag=value format
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flag := strings.TrimPrefix(parts[0], "-")
			flag = strings.TrimPrefix(flag, "-")

			// Convert flag from kebab-case to snake_case for consistency with JSON field names
			flagName := strings.ReplaceAll(flag, "-", "_")

			// Convert value to appropriate type
			value := parseValue(parts[1])

			// Handle array flags by checking if flag already exists
			if existingValue, exists := options[flagName]; exists {
				// Convert to array if not already
				switch existing := existingValue.(type) {
				case []string:
					options[flagName] = append(existing, parts[1])
				case string:
					options[flagName] = []string{existing, parts[1]}
				default:
					options[flagName] = []string{fmt.Sprintf("%v", existing), parts[1]}
				}
			} else {
				options[flagName] = value
			}
			i++
			continue
		}

		// Handle --flag value format
		flag := strings.TrimPrefix(arg, "-")
		flag = strings.TrimPrefix(flag, "-")

		// Convert flag from kebab-case to snake_case for consistency with JSON field names
		flagName := strings.ReplaceAll(flag, "-", "_")

		// Check if next arg is a value (not a flag)
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			value := parseValue(args[i+1])

			// Handle array flags by checking if flag already exists
			if existingValue, exists := options[flagName]; exists {
				// Convert to array if not already
				switch existing := existingValue.(type) {
				case []string:
					options[flagName] = append(existing, args[i+1])
				case string:
					options[flagName] = []string{existing, args[i+1]}
				default:
					options[flagName] = []string{fmt.Sprintf("%v", existing), args[i+1]}
				}
			} else {
				options[flagName] = value
			}
			i += 2 // Skip flag and value
		} else {
			// Boolean flag
			options[flagName] = true
			i++
		}
	}

	// 4. Convert to LlamaServerOptions using existing UnmarshalJSON
	jsonData, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parsed options: %w", err)
	}

	var llamaOptions LlamaServerOptions
	if err := json.Unmarshal(jsonData, &llamaOptions); err != nil {
		return nil, fmt.Errorf("failed to parse command options: %w", err)
	}

	// 5. Return LlamaServerOptions
	return &llamaOptions, nil
}

// parseValue attempts to parse a string value into the most appropriate type
func parseValue(value string) any {
	// Try to parse as boolean
	if strings.ToLower(value) == "true" {
		return true
	}
	if strings.ToLower(value) == "false" {
		return false
	}

	// Try to parse as integer
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try to parse as float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Default to string
	return value
}
