package main

import (
	"bufio"
	"fmt"
	"io"
	"llamactl/pkg/backends"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

const readmeURL = "https://raw.githubusercontent.com/ggml-org/llama.cpp/refs/heads/master/tools/server/README.md"

type FlagInfo struct {
	Flag        string
	Description string
	EnvVar      string
}

func main() {
	// Fetch README from GitHub
	fmt.Println("Fetching README from llama.cpp repository...")
	readmeContent, err := fetchREADME(readmeURL)
	if err != nil {
		fmt.Printf("Error fetching README: %v\n", err)
		return
	}

	// Parse README to extract all flags
	flags := parseREADMEFlags(readmeContent)
	fmt.Printf("Found %d flags in README\n\n", len(flags))

	// Test each flag with the actual parser
	var missing []FlagInfo
	var working []string
	fieldToFlags := make(map[string][]string) // Track which flags set which fields

	for _, flag := range flags {
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
	fmt.Printf("Conflicts: %d\n", len(conflicts))
}

func fetchREADME(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func parseREADMEFlags(content string) []FlagInfo {
	var flags []FlagInfo
	scanner := bufio.NewScanner(strings.NewReader(content))
	inTable := false

	flagRowPattern := regexp.MustCompile(`^\|\s*\x60([^\x60]+)\x60\s*\|`)
	envVarPattern := regexp.MustCompile(`\(env:\s*([A-Z_]+)\)`)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "| Argument | Explanation |") {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(line, "##") {
			inTable = false
			continue
		}

		if !inTable {
			continue
		}

		matches := flagRowPattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		flagText := matches[1]
		flagParts := strings.Split(flagText, ",")

		// Extract environment variable
		envMatches := envVarPattern.FindStringSubmatch(line)
		var envVar string
		if len(envMatches) > 1 {
			envVar = envMatches[1]
		}

		// Extract all flag variations (short and long)
		for _, part := range flagParts {
			part = strings.TrimSpace(part)

			// Skip if it's just a value placeholder or description
			if !strings.HasPrefix(part, "-") {
				continue
			}

			// Extract just the flag name (remove value placeholders)
			flag := strings.Fields(part)[0]

			// Skip if we already have this flag
			exists := false
			for _, f := range flags {
				if f.Flag == flag {
					exists = true
					break
				}
			}
			if !exists {
				flags = append(flags, FlagInfo{
					Flag:        flag,
					Description: flagText,
					EnvVar:      envVar,
				})
			}
		}
	}

	return flags
}

func testFlagAndGetField(flag string) (string, bool) {
	// Try parsing with different value types
	testValues := []string{
		"",           // Boolean flag (no value)
		"1",          // Integer
		"0.5",        // Float
		"test",       // String
		"/tmp/test",  // Path/file
		"localhost",  // Hostname
		"auto",       // Enum-like
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

		_, err := opts.ParseCommand(testCommand)
		if err == nil {
			// Find which field changed
			fieldName := findChangedField(baseline, opts)
			return fieldName, true
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
			} else if strings.HasPrefix(flag, "--") {
				// Check if this flag has a corresponding --no- version in the list
				noFlag := "--no-" + strings.TrimPrefix(flag, "--")
				for _, f := range flags {
					if f == noFlag {
						hasPositive = true
						break
					}
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
