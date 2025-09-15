package llamacpp

import (
	"testing"
)

func TestParseLlamaCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
	}{
		{
			name:      "basic command with model",
			command:   "llama-server --model /path/to/model.gguf",
			expectErr: false,
		},
		{
			name:      "command with multiple flags",
			command:   "llama-server --model /path/to/model.gguf --gpu-layers 32 --ctx-size 4096",
			expectErr: false,
		},
		{
			name:      "command with short flags",
			command:   "llama-server -m /path/to/model.gguf -ngl 32 -c 4096",
			expectErr: false,
		},
		{
			name:      "command with equals format",
			command:   "llama-server --model=/path/to/model.gguf --gpu-layers=32",
			expectErr: false,
		},
		{
			name:      "command with boolean flags",
			command:   "llama-server --model /path/to/model.gguf --verbose --no-mmap",
			expectErr: false,
		},
		{
			name:      "empty command",
			command:   "",
			expectErr: true,
		},
		{
			name:      "case insensitive command",
			command:   "LLAMA-SERVER --model /path/to/model.gguf",
			expectErr: false,
		},
		// New test cases for improved functionality
		{
			name:      "args only without llama-server",
			command:   "--model /path/to/model.gguf --gpu-layers 32",
			expectErr: false,
		},
		{
			name:      "full path to executable",
			command:   "/usr/local/bin/llama-server --model /path/to/model.gguf",
			expectErr: false,
		},
		{
			name:      "negative number handling",
			command:   "llama-server --gpu-layers -1 --model test.gguf",
			expectErr: false,
		},
		{
			name:      "multiline command with backslashes",
			command:   "llama-server --model /path/to/model.gguf \\\n  --ctx-size 4096 \\\n  --batch-size 512",
			expectErr: false,
		},
		{
			name:      "quoted string with special characters",
			command:   `llama-server --model test.gguf --chat-template "{% for message in messages %}{{ message.role }}: {{ message.content }}\n{% endfor %}"`,
			expectErr: false,
		},
		{
			name:      "unterminated quoted string",
			command:   `llama-server --model test.gguf --chat-template "unterminated quote`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseLlamaCommand(tt.command)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}
		})
	}
}

func TestParseLlamaCommandSpecificValues(t *testing.T) {
	// Test specific value parsing
	command := "llama-server --model /test/model.gguf --gpu-layers 32 --ctx-size 4096 --verbose"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "/test/model.gguf" {
		t.Errorf("expected model '/test/model.gguf', got '%s'", result.Model)
	}

	if result.GPULayers != 32 {
		t.Errorf("expected gpu_layers 32, got %d", result.GPULayers)
	}

	if result.CtxSize != 4096 {
		t.Errorf("expected ctx_size 4096, got %d", result.CtxSize)
	}

	if !result.Verbose {
		t.Errorf("expected verbose to be true, got %v", result.Verbose)
	}
}

func TestParseLlamaCommandArrayFlags(t *testing.T) {
	// Test array flag handling (critical for lora, override-tensor, etc.)
	command := "llama-server --model test.gguf --lora adapter1.bin --lora adapter2.bin"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Lora) != 2 {
		t.Errorf("expected 2 lora adapters, got %d", len(result.Lora))
	}

	if result.Lora[0] != "adapter1.bin" || result.Lora[1] != "adapter2.bin" {
		t.Errorf("expected lora adapters [adapter1.bin, adapter2.bin], got %v", result.Lora)
	}
}

func TestParseLlamaCommandMixedFormats(t *testing.T) {
	// Test mixing --flag=value and --flag value formats
	command := "llama-server --model=/path/model.gguf --gpu-layers 16 --batch-size=512 --verbose"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "/path/model.gguf" {
		t.Errorf("expected model '/path/model.gguf', got '%s'", result.Model)
	}

	if result.GPULayers != 16 {
		t.Errorf("expected gpu_layers 16, got %d", result.GPULayers)
	}

	if result.BatchSize != 512 {
		t.Errorf("expected batch_size 512, got %d", result.BatchSize)
	}

	if !result.Verbose {
		t.Errorf("expected verbose to be true, got %v", result.Verbose)
	}
}

func TestParseLlamaCommandTypeConversion(t *testing.T) {
	// Test that values are converted to appropriate types
	command := "llama-server --model test.gguf --temp 0.7 --top-k 40 --no-mmap"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", result.Temperature)
	}

	if result.TopK != 40 {
		t.Errorf("expected top_k 40, got %d", result.TopK)
	}

	if !result.NoMmap {
		t.Errorf("expected no_mmap to be true, got %v", result.NoMmap)
	}
}

func TestParseLlamaCommandArgsOnly(t *testing.T) {
	// Test parsing arguments without llama-server command
	command := "--model /path/to/model.gguf --gpu-layers 32 --ctx-size 4096"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "/path/to/model.gguf" {
		t.Errorf("expected model '/path/to/model.gguf', got '%s'", result.Model)
	}

	if result.GPULayers != 32 {
		t.Errorf("expected gpu_layers 32, got %d", result.GPULayers)
	}

	if result.CtxSize != 4096 {
		t.Errorf("expected ctx_size 4096, got %d", result.CtxSize)
	}
}

func TestParseLlamaCommandFullPath(t *testing.T) {
	// Test full path to executable
	command := "/usr/local/bin/llama-server --model test.gguf --gpu-layers 16"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "test.gguf" {
		t.Errorf("expected model 'test.gguf', got '%s'", result.Model)
	}

	if result.GPULayers != 16 {
		t.Errorf("expected gpu_layers 16, got %d", result.GPULayers)
	}
}

func TestParseLlamaCommandNegativeNumbers(t *testing.T) {
	// Test negative number parsing
	command := "llama-server --model test.gguf --gpu-layers -1 --seed -12345"
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.GPULayers != -1 {
		t.Errorf("expected gpu_layers -1, got %d", result.GPULayers)
	}

	if result.Seed != -12345 {
		t.Errorf("expected seed -12345, got %d", result.Seed)
	}
}

func TestParseLlamaCommandMultiline(t *testing.T) {
	// Test multiline command with backslashes
	command := `llama-server --model /path/to/model.gguf \
  --ctx-size 4096 \
  --batch-size 512 \
  --gpu-layers 32`

	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "/path/to/model.gguf" {
		t.Errorf("expected model '/path/to/model.gguf', got '%s'", result.Model)
	}

	if result.CtxSize != 4096 {
		t.Errorf("expected ctx_size 4096, got %d", result.CtxSize)
	}

	if result.BatchSize != 512 {
		t.Errorf("expected batch_size 512, got %d", result.BatchSize)
	}

	if result.GPULayers != 32 {
		t.Errorf("expected gpu_layers 32, got %d", result.GPULayers)
	}
}

func TestParseLlamaCommandQuotedStrings(t *testing.T) {
	// Test quoted strings with special characters
	command := `llama-server --model test.gguf --api-key "sk-1234567890abcdef" --chat-template "User: {user}\nAssistant: "`
	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "test.gguf" {
		t.Errorf("expected model 'test.gguf', got '%s'", result.Model)
	}

	if result.APIKey != "sk-1234567890abcdef" {
		t.Errorf("expected api_key 'sk-1234567890abcdef', got '%s'", result.APIKey)
	}

	expectedTemplate := "User: {user}\\nAssistant: "
	if result.ChatTemplate != expectedTemplate {
		t.Errorf("expected chat_template '%s', got '%s'", expectedTemplate, result.ChatTemplate)
	}
}

func TestParseLlamaCommandUnslothExample(t *testing.T) {
	// Test with realistic unsloth-style command
	command := `llama-server --model /path/to/model.gguf \
  --ctx-size 4096 \
  --batch-size 512 \
  --gpu-layers -1 \
  --temp 0.7 \
  --repeat-penalty 1.1 \
  --top-k 40 \
  --top-p 0.95 \
  --host 0.0.0.0 \
  --port 8000 \
  --api-key "sk-1234567890abcdef"`

	result, err := ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key fields
	if result.Model != "/path/to/model.gguf" {
		t.Errorf("expected model '/path/to/model.gguf', got '%s'", result.Model)
	}

	if result.CtxSize != 4096 {
		t.Errorf("expected ctx_size 4096, got %d", result.CtxSize)
	}

	if result.BatchSize != 512 {
		t.Errorf("expected batch_size 512, got %d", result.BatchSize)
	}

	if result.GPULayers != -1 {
		t.Errorf("expected gpu_layers -1, got %d", result.GPULayers)
	}

	if result.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", result.Temperature)
	}

	if result.RepeatPenalty != 1.1 {
		t.Errorf("expected repeat_penalty 1.1, got %f", result.RepeatPenalty)
	}

	if result.TopK != 40 {
		t.Errorf("expected top_k 40, got %d", result.TopK)
	}

	if result.TopP != 0.95 {
		t.Errorf("expected top_p 0.95, got %f", result.TopP)
	}

	if result.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got '%s'", result.Host)
	}

	if result.Port != 8000 {
		t.Errorf("expected port 8000, got %d", result.Port)
	}

	if result.APIKey != "sk-1234567890abcdef" {
		t.Errorf("expected api_key 'sk-1234567890abcdef', got '%s'", result.APIKey)
	}
}

// Focused additional edge case tests (kept minimal per guidance)
func TestParseLlamaCommandSingleQuotedValue(t *testing.T) {
	cmd := "llama-server --model 'my model.gguf' --alias 'Test Alias'"
	result, err := ParseLlamaCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Model != "my model.gguf" {
		t.Errorf("expected model 'my model.gguf', got '%s'", result.Model)
	}
	if result.Alias != "Test Alias" {
		t.Errorf("expected alias 'Test Alias', got '%s'", result.Alias)
	}
}

func TestParseLlamaCommandMixedArrayForms(t *testing.T) {
	// Same multi-value flag using --flag value and --flag=value forms
	cmd := "llama-server --lora adapter1.bin --lora=adapter2.bin --lora adapter3.bin"
	result, err := ParseLlamaCommand(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Lora) != 3 {
		t.Fatalf("expected 3 lora values, got %d (%v)", len(result.Lora), result.Lora)
	}
	expected := []string{"adapter1.bin", "adapter2.bin", "adapter3.bin"}
	for i, v := range expected {
		if result.Lora[i] != v {
			t.Errorf("expected lora[%d]=%s got %s", i, v, result.Lora[i])
		}
	}
}

func TestParseLlamaCommandMalformedFlag(t *testing.T) {
	cmd := "llama-server ---model test.gguf"
	_, err := ParseLlamaCommand(cmd)
	if err == nil {
		t.Fatalf("expected error for malformed flag but got none")
	}
}
