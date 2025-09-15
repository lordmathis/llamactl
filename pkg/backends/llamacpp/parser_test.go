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
			name:      "invalid command without llama-server",
			command:   "other-command --model /path/to/model.gguf",
			expectErr: true,
		},
		{
			name:      "case insensitive command",
			command:   "LLAMA-SERVER --model /path/to/model.gguf",
			expectErr: false,
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
