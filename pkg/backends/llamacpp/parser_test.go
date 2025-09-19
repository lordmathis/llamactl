package llamacpp_test

import (
	"llamactl/pkg/backends/llamacpp"
	"testing"
)

func TestParseLlamaCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
	}{
		{
			name:      "basic command",
			command:   "llama-server --model /path/to/model.gguf --gpu-layers 32",
			expectErr: false,
		},
		{
			name:      "args only",
			command:   "--model /path/to/model.gguf --ctx-size 4096",
			expectErr: false,
		},
		{
			name:      "mixed flag formats",
			command:   "llama-server --model=/path/model.gguf --gpu-layers 16 --verbose",
			expectErr: false,
		},
		{
			name:      "quoted strings",
			command:   `llama-server --model test.gguf --api-key "sk-1234567890abcdef"`,
			expectErr: false,
		},
		{
			name:      "empty command",
			command:   "",
			expectErr: true,
		},
		{
			name:      "unterminated quote",
			command:   `llama-server --model test.gguf --api-key "unterminated`,
			expectErr: true,
		},
		{
			name:      "malformed flag",
			command:   "llama-server ---model test.gguf",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := llamacpp.ParseLlamaCommand(tt.command)

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
			}
		})
	}
}

func TestParseLlamaCommandValues(t *testing.T) {
	command := "llama-server --model /test/model.gguf --gpu-layers 32 --temp 0.7 --verbose --no-mmap"
	result, err := llamacpp.ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "/test/model.gguf" {
		t.Errorf("expected model '/test/model.gguf', got '%s'", result.Model)
	}

	if result.GPULayers != 32 {
		t.Errorf("expected gpu_layers 32, got %d", result.GPULayers)
	}

	if result.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", result.Temperature)
	}

	if !result.Verbose {
		t.Errorf("expected verbose to be true")
	}

	if !result.NoMmap {
		t.Errorf("expected no_mmap to be true")
	}
}

func TestParseLlamaCommandArrays(t *testing.T) {
	command := "llama-server --model test.gguf --lora adapter1.bin --lora=adapter2.bin"
	result, err := llamacpp.ParseLlamaCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Lora) != 2 {
		t.Errorf("expected 2 lora adapters, got %d", len(result.Lora))
	}

	expected := []string{"adapter1.bin", "adapter2.bin"}
	for i, v := range expected {
		if result.Lora[i] != v {
			t.Errorf("expected lora[%d]=%s got %s", i, v, result.Lora[i])
		}
	}
}
