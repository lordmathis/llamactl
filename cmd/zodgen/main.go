package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type FieldInfo struct {
	Name    string
	Type    string
	JSONTag string
	Comment string
	AltKeys []string
}

func main() {
	// Parse the llama.go file
	fset := token.NewFileSet()
	sourceFile := filepath.Join("pkg", "backends", "llama.go")

	file, err := parser.ParseFile(fset, sourceFile, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Find LlamaServerOptions struct
	var fields []FieldInfo
	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Name.Name != "LlamaServerOptions" {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return false
		}

		// Extract fields with section headers
		for _, field := range structType.Fields.List {
			// Check for section header in Doc comments
			if field.Doc != nil {
				for _, comment := range field.Doc.List {
					text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
					if text != "" && (strings.Contains(text, "params") || strings.Contains(text, "Params") || strings.Contains(text, "Parameters")) {
						fields = append(fields, FieldInfo{
							Comment: text,
						})
					}
				}
			}

			if len(field.Names) == 0 {
				continue // Skip embedded fields
			}

			fieldName := field.Names[0].Name

			// Skip unexported fields
			if !ast.IsExported(fieldName) {
				continue
			}

			// Extract JSON tag
			var jsonTag string
			if field.Tag != nil {
				tag := strings.Trim(field.Tag.Value, "`")
				if strings.Contains(tag, "json:") {
					parts := strings.Split(tag, "\"")
					if len(parts) >= 2 {
						jsonTag = strings.Split(parts[1], ",")[0]
					}
				}
			}

			// Skip fields without JSON tags or with "-"
			if jsonTag == "" || jsonTag == "-" {
				continue
			}

			// Extract type
			goType := typeToString(field.Type)

			// Extract inline comment
			var comment string
			if field.Comment != nil && len(field.Comment.List) > 0 {
				comment = strings.TrimSpace(strings.TrimPrefix(field.Comment.List[0].Text, "//"))
			}

			// Extract alternative keys from comment
			altKeys := parseAltKeys(comment, jsonTag)

			fields = append(fields, FieldInfo{
				Name:    fieldName,
				Type:    goType,
				JSONTag: jsonTag,
				Comment: comment,
				AltKeys: altKeys,
			})
		}

		return false
	})

	// Generate Zod schema
	generateZodSchema(fields)

	// Generate Alt Keys schema
	generateAltKeysSchema(fields)
}

// typeToString converts AST type to string representation
func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		elemType := typeToString(t.Elt)
		return "[]" + elemType
	case *ast.MapType:
		keyType := typeToString(t.Key)
		valueType := typeToString(t.Value)
		return "map[" + keyType + "]" + valueType
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	default:
		return "unknown"
	}
}

// goTypeToZod converts Go type to Zod type
func goTypeToZod(goType string) string {
	switch goType {
	case "string":
		return "z.string()"
	case "int", "int64", "float64":
		return "z.number()"
	case "bool":
		return "z.boolean()"
	case "[]string":
		return "z.array(z.string())"
	case "map[string]string":
		return "z.record(z.string(), z.string())"
	default:
		return "z.string() // unknown type: " + goType
	}
}

// parseAltKeys extracts alternative flag names from comment
// Example: "--temp N", jsonTag="temp" → []string{}
// Example: "-t, --threads N", jsonTag="threads" → []string{"t"}
// Example: "--predict, --n-predict N", jsonTag="predict" → []string{"n-predict"}
// Example: "-v, --verbose, --log-verbose", jsonTag="verbose" → []string{"v", "log-verbose"}
// Example: "--rope-scaling {none,linear,yarn}", jsonTag="rope_scaling" → []string{} (not an alt key, just enum values)
func parseAltKeys(comment string, jsonTag string) []string {
	var altKeys []string

	// Convert jsonTag from snake_case to kebab-case for comparison
	// "threads_batch" → "threads-batch"
	jsonTagKebab := strings.ReplaceAll(jsonTag, "_", "-")

	// Split by comma to get individual flags
	flags := strings.SplitSeq(comment, ",")

	for flag := range flags {
		flag = strings.TrimSpace(flag)

		// Check if this is a flag (starts with -)
		if !strings.HasPrefix(flag, "-") {
			continue
		}

		// Extract flag name by removing prefix
		flagName := flag
		if after, ok := strings.CutPrefix(flagName, "--"); ok {
			flagName = after
		} else if after0, ok0 := strings.CutPrefix(flagName, "-"); ok0 {
			flagName = after0
		}

		// Remove suffix (everything after first space)
		if idx := strings.Index(flagName, " "); idx != -1 {
			flagName = flagName[:idx]
		}

		// Skip if this is the main flag (matches the jsonTag in kebab-case)
		if flagName == jsonTagKebab {
			continue
		}

		// Validate the flag name:
		// - Must not be empty
		// - Must not start with < or [ (placeholders)
		// - Must be lowercase letters, numbers, hyphens, or underscores only
		// - Must start with a letter
		if flagName == "" {
			continue
		}
		if strings.HasPrefix(flagName, "<") || strings.HasPrefix(flagName, "[") {
			continue
		}
		if strings.HasPrefix(flagName, "}") {
			continue
		}
		if flagName[0] >= '0' && flagName[0] <= '9' {
			continue
		}

		// Check if flag name is valid (only lowercase letters, numbers, hyphens, underscores)
		valid := true
		for _, c := range flagName {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}

		altKeys = append(altKeys, flagName)
	}

	return altKeys
}

func generateZodSchema(fields []FieldInfo) {
	fmt.Println("// Define the LlamaCpp backend options schema")
	fmt.Println("export const LlamaCppBackendOptionsSchema = z.object({")

	for i, field := range fields {
		// Section header (no JSON tag)
		if field.JSONTag == "" {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("  // %s\n", field.Comment)
			continue
		}

		// Regular field
		zodType := goTypeToZod(field.Type)
		fmt.Printf("  %s: %s.optional(),", field.JSONTag, zodType)

		if field.Comment != "" {
			fmt.Printf(" // %s", field.Comment)
		}
		fmt.Println()
	}

	fmt.Println("})")
	fmt.Println()
}

func generateAltKeysSchema(fields []FieldInfo) {
	fmt.Println("// Define the LlamaCpp backend options schema with alternative keys")
	fmt.Println("// These are alternative flag names that can be used in preset.ini files")
	fmt.Println("export const LlamaCppAltKeysSchema = z.object({")

	for _, field := range fields {
		// Skip section headers and fields without alt keys
		if field.JSONTag == "" || len(field.AltKeys) == 0 {
			continue
		}

		zodType := goTypeToZod(field.Type)

		for _, altKey := range field.AltKeys {
			fmt.Printf("  '%s': %s.optional(), // alternative to '%s'\n", altKey, zodType, field.JSONTag)
		}
	}

	fmt.Println("})")
}
