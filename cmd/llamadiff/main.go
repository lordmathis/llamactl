package main

import (
	"fmt"
	"llamactl/pkg/backends"
	"os/exec"
	"reflect"
	"regexp"
	"slices"
	"strings"
)

type FlagInfo struct {
	Flag        string
	Description string
	EnvVar      string
}

// Utility flags that are not expected to be in the struct
var utilityFlags = map[string]bool{
	"-h":               true,
	"--help":           true,
	"--usage":          true,
	"--version":        true,
	"--license":        true,
	"-cl":              true,
	"--cache-list":     true,
	"--completion-bash": true,
	"--list-devices":   true,
}

func main() {
	// Run llama-server --help to get all flags
	fmt.Println("Running llama-server --help to extract flags...")
	helpOutput, err := runLlamaServerHelp()
	if err != nil {
		fmt.Printf("Error running llama-server --help: %v\n", err)
		return
	}

	// Parse help output to extract all flags
	flags := parseHelpFlags(helpOutput)
	fmt.Printf("Found %d flags in llama-server --help\n\n", len(flags))

	// Test each flag with the actual parser
	var missing []FlagInfo
	var working []string
	var ignored []string
	fieldToFlags := make(map[string][]string) // Track which flags set which fields

	for _, flag := range flags {
		// Skip utility flags
		if utilityFlags[flag.Flag] {
			ignored = append(ignored, flag.Flag)
			continue
		}

		fieldName, ok := testFlagAndGetField(flag.Flag)
		if ok {
			working = append(working, flag.Flag)
			if fieldName != "" {
				fieldToFlags[fieldName] = append(fieldToFlags[fieldName], flag.Flag)
			}
		} else {
			missing = append(missing, flag)
		}
	}

	// Check for conflicts (positive/negative pairs setting the same field)
	conflicts := findConflicts(fieldToFlags)

	// Report results
	fmt.Println("=== FLAGS MISSING FROM llamactl ===")
	if len(missing) == 0 {
		fmt.Println("(none)")
	} else {
		for _, flag := range missing {
			fmt.Printf("  %s", flag.Flag)
			if flag.EnvVar != "" {
				fmt.Printf(" [env: %s]", flag.EnvVar)
			}
			fmt.Println()
		}
	}

	fmt.Println("\n=== POSITIVE/NEGATIVE FLAG CONFLICTS ===")
	if len(conflicts) == 0 {
		fmt.Println("(none)")
	} else {
		for _, conflict := range conflicts {
			fmt.Printf("  Field '%s' is set by both:\n", conflict.Field)
			for _, flag := range conflict.Flags {
				fmt.Printf("    %s\n", flag)
			}
			fmt.Println()
		}
	}

	fmt.Printf("\n=== SUMMARY ===\n")
	fmt.Printf("Total flags: %d\n", len(flags))
	fmt.Printf("Working: %d\n", len(working))
	fmt.Printf("Missing: %d\n", len(missing))
	fmt.Printf("Ignored (utility): %d\n", len(ignored))
	fmt.Printf("Conflicts: %d\n", len(conflicts))
}

func runLlamaServerHelp() (string, error) {
	cmd := exec.Command("llama-server", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// llama-server --help may return non-zero exit code, but we still get output
		if len(output) == 0 {
			return "", fmt.Errorf("failed to run llama-server --help: %w", err)
		}
	}
	return string(output), nil
}

func parseHelpFlags(content string) []FlagInfo {
	// Match any flag: one or two dashes followed by alphanumeric/dash characters
	// Must start with a letter after the dash(es) to avoid matching things like -1, -2
	// Must be preceded by whitespace or start of line to avoid matching mid-word dashes
	flagPattern := regexp.MustCompile(`(?m)(?:^|\s)(-{1,2}[a-zA-Z][-a-zA-Z0-9]*)`)

	// Find all flags in the entire output
	allMatches := flagPattern.FindAllStringSubmatch(content, -1)

	// Deduplicate and filter
	seen := make(map[string]bool)
	var flags []FlagInfo

	for _, match := range allMatches {
		if len(match) < 2 {
			continue
		}
		flag := match[1] // Extract the captured flag (group 1)

		// Skip if already seen
		if seen[flag] {
			continue
		}

		// Skip separator lines (just dashes)
		if regexp.MustCompile(`^-+$`).MatchString(flag) {
			continue
		}

		seen[flag] = true
		flags = append(flags, FlagInfo{
			Flag: flag,
		})
	}

	return flags
}

func testFlagAndGetField(flag string) (string, bool) {
	// Try parsing with different value types
	testValues := []string{
		"",          // Boolean flag (no value)
		"1",         // Integer
		"0.5",       // Float
		"test",      // String
		"/tmp/test", // Path/file
		"localhost", // Hostname
		"auto",      // Enum-like
	}

	for _, value := range testValues {
		baseline := &backends.LlamaServerOptions{}
		opts := &backends.LlamaServerOptions{}

		var testCommand string
		if value == "" {
			testCommand = fmt.Sprintf("llama-server %s", flag)
		} else {
			testCommand = fmt.Sprintf("llama-server %s %s", flag, value)
		}

		result, err := opts.ParseCommand(testCommand)
		if err == nil && result != nil {
			// ParseCommand returns a new struct with parsed values
			// Convert result to *LlamaServerOptions
			if parsedOpts, ok := result.(*backends.LlamaServerOptions); ok {
				// Find which field changed
				fieldName := findChangedField(baseline, parsedOpts)
				// Only consider it working if it actually set a real struct field
				// Flags that only go into ExtraArgs are not properly supported
				if fieldName != "" && fieldName != "ExtraArgs" {
					return fieldName, true
				}
			}
		}
	}

	// If all attempts failed, the flag is not supported
	return "", false
}

func findChangedField(baseline, modified *backends.LlamaServerOptions) string {
	baseVal := reflect.ValueOf(baseline).Elem()
	modVal := reflect.ValueOf(modified).Elem()
	baseType := baseVal.Type()

	for i := 0; i < baseVal.NumField(); i++ {
		baseField := baseVal.Field(i)
		modField := modVal.Field(i)

		// Skip if not comparable or unexported
		if !baseField.CanInterface() || !modField.CanInterface() {
			continue
		}

		// Check if the field changed
		if !reflect.DeepEqual(baseField.Interface(), modField.Interface()) {
			return baseType.Field(i).Name
		}
	}

	return ""
}

type Conflict struct {
	Field string
	Flags []string
}

func findConflicts(fieldToFlags map[string][]string) []Conflict {
	var conflicts []Conflict

	for field, flags := range fieldToFlags {
		if len(flags) < 2 {
			continue
		}

		// Check if there are both positive and negative versions
		hasPositive := false
		hasNegative := false

		for _, flag := range flags {
			if strings.HasPrefix(flag, "--no-") {
				hasNegative = true
			} else if after, ok := strings.CutPrefix(flag, "--"); ok {
				// Check if this flag has a corresponding --no- version in the list
				noFlag := "--no-" + after
				if slices.Contains(flags, noFlag) {
					hasPositive = true
				}
			}
		}

		if hasPositive && hasNegative {
			conflicts = append(conflicts, Conflict{
				Field: field,
				Flags: flags,
			})
		}
	}

	return conflicts
}
