# llamadiff - llama-server Flag Validator

This tool validates that all flags supported by `llama-server` are properly handled by the llamactl parser.

## Usage

Run the validator:

```bash
go run cmd/llamadiff/main.go
```

The tool will output a report showing:
- **Working**: Flags that are correctly parsed and mapped to struct fields
- **Missing**: Flags from `llama-server --help` that are not yet supported
- **Ignored**: Utility flags (--help, --version, etc.) that don't need struct fields
- **Conflicts**: Fields that have both positive and negative flag variants mapping to the same struct field

## How It Works

1. Runs `llama-server --help` to extract all available flags
2. Parses the help output using regex to identify flag names
3. Tests each flag by attempting to parse it with different value types
4. Uses reflection to detect which struct field (if any) was set by each flag
5. Reports discrepancies between llama-server's flags and our parser


## Understanding the Output

### Working Flags
These flags are correctly parsed and set their corresponding struct fields.

### Missing Flags
These flags exist in llama-server but aren't handled by our parser. You'll need to:
1. Add the field to `LlamaServerOptions` in `pkg/backends/llama.go`
2. Add any necessary aliases to `llamaFieldMappings`
3. Update multi-valued flag list if needed

### Conflicts
These indicate fields where multiple flag variants (like `--flag` and `--no-flag`) map to the same struct field. This may be intentional (for boolean flags) or indicate a parsing issue.
