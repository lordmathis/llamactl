package mlx

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ParseMlxCommand parses a mlx_lm.server command string into MlxServerOptions
// Supports multiple formats:
// 1. Full command: "mlx_lm.server --model model/path"
// 2. Full path: "/usr/local/bin/mlx_lm.server --model model/path"
// 3. Args only: "--model model/path --host 0.0.0.0"
// 4. Multiline commands with backslashes
func ParseMlxCommand(command string) (*MlxServerOptions, error) {
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

		if !strings.HasPrefix(arg, "-") { // skip positional / stray values
			i++
			continue
		}

		// Reject malformed flags with more than two leading dashes (e.g. ---model) to surface user mistakes
		if strings.HasPrefix(arg, "---") {
			return nil, fmt.Errorf("malformed flag: %s", arg)
		}

		// Unified parsing for --flag=value vs --flag value
		var rawFlag, rawValue string
		hasEquals := false
		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			rawFlag = parts[0]
			rawValue = parts[1] // may be empty string
			hasEquals = true
		} else {
			rawFlag = arg
		}

		flagCore := strings.TrimPrefix(strings.TrimPrefix(rawFlag, "-"), "-")
		flagName := strings.ReplaceAll(flagCore, "-", "_")

		// Detect value if not in equals form
		valueProvided := hasEquals
		if !hasEquals {
			if i+1 < len(args) && !isFlag(args[i+1]) { // next token is value
				rawValue = args[i+1]
				valueProvided = true
			}
		}

		if valueProvided {
			// MLX-specific validation for certain flags
			if flagName == "log_level" && !isValidLogLevel(rawValue) {
				return nil, fmt.Errorf("invalid log level: %s", rawValue)
			}
			
			options[flagName] = parseValue(rawValue)
			
			// Advance index: if we consumed a following token as value (non equals form), skip it
			if !hasEquals && i+1 < len(args) && rawValue == args[i+1] {
				i += 2
			} else {
				i++
			}
			continue
		}

		// Boolean flag (no value) - MLX specific boolean flags
		if flagName == "trust_remote_code" || flagName == "use_default_chat_template" {
			options[flagName] = true
		} else {
			options[flagName] = true
		}
		i++
	}

	// 4. Convert to MlxServerOptions using existing UnmarshalJSON
	jsonData, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parsed options: %w", err)
	}

	var mlxOptions MlxServerOptions
	if err := json.Unmarshal(jsonData, &mlxOptions); err != nil {
		return nil, fmt.Errorf("failed to parse command options: %w", err)
	}

	// 5. Return MlxServerOptions
	return &mlxOptions, nil
}

// isValidLogLevel validates MLX log levels
func isValidLogLevel(level string) bool {
	validLevels := []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// parseValue attempts to parse a string value into the most appropriate type
func parseValue(value string) any {
	// Surrounding matching quotes (single or double)
	if l := len(value); l >= 2 {
		if (value[0] == '"' && value[l-1] == '"') || (value[0] == '\'' && value[l-1] == '\'') {
			value = value[1 : l-1]
		}
	}

	lower := strings.ToLower(value)
	if lower == "true" {
		return true
	}
	if lower == "false" {
		return false
	}

	if intVal, err := strconv.Atoi(value); err == nil {
		return intVal
	}
	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}
	return value
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

	// Case 1: Full path to executable (contains path separator or ends with mlx_lm.server)
	if strings.Contains(firstToken, string(filepath.Separator)) ||
		strings.HasSuffix(filepath.Base(firstToken), "mlx_lm.server") {
		return tokens[1:], nil // Return everything except the executable
	}

	// Case 2: Just "mlx_lm.server" command
	if strings.ToLower(firstToken) == "mlx_lm.server" {
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
		} else if !inQuotes && (c == ' ' || c == '\t' || c == '\n') {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(c)
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("unclosed quote in command")
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

// isFlag checks if a string looks like a command line flag
func isFlag(s string) bool {
	return strings.HasPrefix(s, "-")
}