package llamacpp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParseLlamaCommand parses a llama-server command string into LlamaServerOptions
// Supports multiple formats:
// 1. Full command: "llama-server --model file.gguf"
// 2. Full path: "/usr/local/bin/llama-server --model file.gguf"
// 3. Args only: "--model file.gguf --gpu-layers 32"
// 4. Multiline commands with backslashes
func ParseLlamaCommand(command string) (*LlamaServerOptions, error) {
	// 1. Normalize the command - handle multiline with backslashes
	trimmed := normalizeMultilineCommand(command)
	if trimmed == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// 2. Extract arguments from command
	args, err := extractArgumentsFromCommand(trimmed)
	if err != nil {
		return nil, err
	}

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
		// Special case: allow negative numbers as values
		if i+1 < len(args) && !isFlag(args[i+1]) {
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

	// Try to parse as integer (handle negative numbers)
	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}

	// Try to parse as float
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	// Default to string (remove quotes if present)
	return strings.Trim(value, `""`)
}

// normalizeMultilineCommand handles multiline commands with backslashes
func normalizeMultilineCommand(command string) string {
	// Handle escaped newlines (backslash followed by newline)
	re := regexp.MustCompile(`\\\s*\n\s*`)
	normalized := re.ReplaceAllString(command, " ")
	
	// Clean up extra whitespace
	re = regexp.MustCompile(`\s+`)
	normalized = re.ReplaceAllString(normalized, " ")
	
	return strings.TrimSpace(normalized)
}

// extractArgumentsFromCommand extracts arguments from various command formats
func extractArgumentsFromCommand(command string) ([]string, error) {
	// Split command into tokens respecting quotes
	tokens, err := splitCommandTokens(command)
	if err != nil {
		return nil, err
	}
	
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no command tokens found")
	}
	
	// Check if first token looks like an executable
	firstToken := tokens[0]
	
	// Case 1: Full path to executable (contains path separator or ends with llama-server)
	if strings.Contains(firstToken, string(filepath.Separator)) || 
	   strings.HasSuffix(filepath.Base(firstToken), "llama-server") {
		return tokens[1:], nil // Return everything except the executable
	}
	
	// Case 2: Just "llama-server" command
	if strings.ToLower(firstToken) == "llama-server" {
		return tokens[1:], nil // Return everything except the command
	}
	
	// Case 3: Arguments only (starts with a flag)
	if strings.HasPrefix(firstToken, "-") {
		return tokens, nil // Return all tokens as arguments
	}
	
	// Case 4: Unknown format - might be a different executable name
	// Be permissive and assume it's the executable
	return tokens[1:], nil
}

// splitCommandTokens splits a command string into tokens, respecting quotes
func splitCommandTokens(command string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)
	escaped := false
	
	for i := 0; i < len(command); i++ {
		c := command[i]
		
		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}
		
		if c == '\\' {
			escaped = true
			current.WriteByte(c)
			continue
		}
		
		if !inQuotes && (c == '"' || c == '\'') {
			inQuotes = true
			quoteChar = c
			current.WriteByte(c)
		} else if inQuotes && c == quoteChar {
			inQuotes = false
			quoteChar = 0
			current.WriteByte(c)
		} else if !inQuotes && (c == ' ' || c == '\t') {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}
	
	if inQuotes {
		return nil, fmt.Errorf("unterminated quoted string")
	}
	
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	
	return tokens, nil
}

// isFlag determines if a string is a command line flag or a value
// Handles the special case where negative numbers should be treated as values, not flags
func isFlag(arg string) bool {
	if !strings.HasPrefix(arg, "-") {
		return false
	}
	
	// Special case: if it's a negative number, treat it as a value
	if _, err := strconv.ParseFloat(arg, 64); err == nil {
		return false
	}
	
	return true
}