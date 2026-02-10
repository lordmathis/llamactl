# zodgen - Zod Schema Generator

This tool generates the TypeScript Zod schema for the frontend from the Go struct definition.

## Usage

Generate the Zod schema to stdout:

```bash
go run cmd/zodgen/main.go
```

Then manually copy the generated schema into `webui/src/schemas/backends/llamacpp.ts`, replacing the schema definition while preserving the hand-written helper functions at the end of the file.

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
