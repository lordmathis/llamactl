# zodgen - Zod Schema Generator

This tool generates TypeScript Zod schemas for the frontend from the Go struct definition.

## Usage

Generate the Zod schemas to stdout:

```bash
go run cmd/zodgen/main.go
```

Then manually copy the generated schemas into `webui/src/schemas/backends/llamacpp.ts`, replacing the schema definitions.

## How It Works

1. Parses `pkg/backends/llama.go` using Go's AST parser
2. Extracts the `LlamaServerOptions` struct definition
3. Reads field names, types, JSON tags, and comments
4. Maps Go types to Zod types:
   - `string` → `z.string()`
   - `int`, `int64`, `float64` → `z.number()`
   - `bool` → `z.boolean()`
   - `[]string` → `z.array(z.string())`
   - `map[string]string` → `z.record(z.string(), z.string())`
5. Preserves section header comments (e.g., "Common params", "Sampling params")
6. Generates TypeScript code with proper formatting

## Output

The tool generates two schemas:

1. **Main Schema** (`LlamaCppBackendOptionsSchema`): The primary schema with all backend options
2. **Alt Keys Schema** (`LlamaCppAltKeysSchema`): Alternative flag names that can be used in preset.ini files (e.g., `n-predict` instead of `predict`)

### Alt Keys Schema

The alt keys schema includes alternative command-line flag names extracted from the comments. For example:

- `-t, --threads N` → alt key: `t`
- `-n, --predict, --n-predict N` → alt keys: `n`, `n-predict`
- `-v, --verbose, --log-verbose` → alt keys: `v`, `log-verbose`

These alt keys can be used in preset.ini files and will be suggested for autocompletion.

