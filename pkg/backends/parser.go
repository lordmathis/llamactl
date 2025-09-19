package backends

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CommandParserConfig holds configuration for parsing command line arguments
type CommandParserConfig struct {
	// ExecutableNames are the names of executables to detect (e.g., "llama-server", "mlx_lm.server")
	ExecutableNames []string
	// SubcommandNames are optional subcommands (e.g., "serve" for vllm)
	SubcommandNames []string
	// MultiValuedFlags are flags that can accept multiple values
	MultiValuedFlags map[string]struct{}
}

// ParseCommand parses a command string using the provided configuration
func ParseCommand(command string, config CommandParserConfig, target any) error {
	// 1. Normalize the command - handle multiline with backslashes
	trimmed := normalizeMultilineCommand(command)
	if trimmed == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// 2. Extract arguments from command
	args, err := extractArgumentsFromCommand(trimmed, config)
	if err != nil {
		return err
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
			return fmt.Errorf("malformed flag: %s", arg)
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

		// Determine if multi-valued flag
		_, isMulti := config.MultiValuedFlags[flagName]

		// Normalization helper: ensure slice for multi-valued flags
		appendValue := func(valStr string) {
			if existing, ok := options[flagName]; ok {
				// Existing value; ensure slice semantics for multi-valued flags or repeated occurrences
				if slice, ok := existing.([]string); ok {
					options[flagName] = append(slice, valStr)
					return
				}
				// Convert scalar to slice
				options[flagName] = []string{fmt.Sprintf("%v", existing), valStr}
				return
			}
			// First value
			if isMulti {
				options[flagName] = []string{valStr}
			} else {
				// We'll parse type below for single-valued flags
				options[flagName] = valStr
			}
		}

		if valueProvided {
			// Use raw token for multi-valued flags; else allow typed parsing
			appendValue(rawValue)
			if !isMulti { // convert to typed value if scalar
				if strVal, ok := options[flagName].(string); ok { // still scalar
					options[flagName] = parseValue(strVal)
				}
			}
			// Advance index: if we consumed a following token as value (non equals form), skip it
			if !hasEquals && i+1 < len(args) && rawValue == args[i+1] {
				i += 2
			} else {
				i++
			}
			continue
		}

		// Boolean flag (no value)
		options[flagName] = true
		i++
	}

	// 4. Convert to target struct using JSON marshaling
	jsonData, err := json.Marshal(options)
	if err != nil {
		return fmt.Errorf("failed to marshal parsed options: %w", err)
	}

	if err := json.Unmarshal(jsonData, target); err != nil {
		return fmt.Errorf("failed to parse command options: %w", err)
	}

	return nil
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
func extractArgumentsFromCommand(command string, config CommandParserConfig) ([]string, error) {
	// Split command into tokens respecting quotes
	tokens, err := splitCommandTokens(command)
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("no command tokens found")
	}

	firstToken := tokens[0]

	// Check for full path executable
	if strings.Contains(firstToken, string(filepath.Separator)) {
		baseName := filepath.Base(firstToken)
		for _, execName := range config.ExecutableNames {
			if strings.HasSuffix(baseName, execName) {
				return skipExecutableAndSubcommands(tokens[1:], config.SubcommandNames)
			}
		}
		// Unknown executable, assume it's still an executable
		return skipExecutableAndSubcommands(tokens[1:], config.SubcommandNames)
	}

	// Check for simple executable names
	lowerFirstToken := strings.ToLower(firstToken)
	for _, execName := range config.ExecutableNames {
		if lowerFirstToken == strings.ToLower(execName) {
			return skipExecutableAndSubcommands(tokens[1:], config.SubcommandNames)
		}
	}

	// Check for subcommands (like "serve" for vllm)
	for _, subCmd := range config.SubcommandNames {
		if lowerFirstToken == strings.ToLower(subCmd) {
			return tokens[1:], nil // Return everything except the subcommand
		}
	}

	// Arguments only (starts with a flag)
	if strings.HasPrefix(firstToken, "-") {
		return tokens, nil // Return all tokens as arguments
	}

	// Unknown format - might be a different executable name
	return skipExecutableAndSubcommands(tokens[1:], config.SubcommandNames)
}

// skipExecutableAndSubcommands removes subcommands from the beginning of tokens
func skipExecutableAndSubcommands(tokens []string, subcommands []string) ([]string, error) {
	if len(tokens) == 0 {
		return tokens, nil
	}

	// Check if first token is a subcommand
	if len(subcommands) > 0 && len(tokens) > 0 {
		lowerFirstToken := strings.ToLower(tokens[0])
		for _, subCmd := range subcommands {
			if lowerFirstToken == strings.ToLower(subCmd) {
				return tokens[1:], nil // Skip the subcommand
			}
		}
	}

	return tokens, nil
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
		return nil, errors.New("unterminated quoted string")
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
