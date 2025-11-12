package backends_test

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/testutil"
	"reflect"
	"testing"
)

func TestLlamaCppBuildCommandArgs_BooleanFields(t *testing.T) {
	tests := []struct {
		name     string
		options  backends.LlamaServerOptions
		expected []string
		excluded []string
	}{
		{
			name: "verbose true",
			options: backends.LlamaServerOptions{
				Verbose: true,
			},
			expected: []string{"--verbose"},
		},
		{
			name: "verbose false",
			options: backends.LlamaServerOptions{
				Verbose: false,
			},
			excluded: []string{"--verbose"},
		},
		{
			name: "multiple booleans",
			options: backends.LlamaServerOptions{
				Verbose: true,
				Mlock:   false,
				NoMmap:  true,
			},
			expected: []string{"--verbose", "--no-mmap"},
			excluded: []string{"--mlock"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.options.BuildCommandArgs()

			for _, expectedArg := range tt.expected {
				if !testutil.Contains(args, expectedArg) {
					t.Errorf("Expected argument %q not found in %v", expectedArg, args)
				}
			}

			for _, excludedArg := range tt.excluded {
				if testutil.Contains(args, excludedArg) {
					t.Errorf("Excluded argument %q found in %v", excludedArg, args)
				}
			}
		})
	}
}

func TestLlamaCppBuildCommandArgs_ZeroValues(t *testing.T) {
	options := backends.LlamaServerOptions{
		Port:        0,     // Should be excluded
		Threads:     0,     // Should be excluded
		Temperature: 0,     // Should be excluded
		Model:       "",    // Should be excluded
		Verbose:     false, // Should be excluded
	}

	args := options.BuildCommandArgs()

	// Zero values should not appear in arguments
	excludedArgs := []string{
		"--port", "0",
		"--threads", "0",
		"--temperature", "0",
		"--model", "",
		"--verbose",
	}

	for _, excludedArg := range excludedArgs {
		if testutil.Contains(args, excludedArg) {
			t.Errorf("Zero value argument %q should not be present in %v", excludedArg, args)
		}
	}
}

func TestLlamaCppBuildCommandArgs_ArrayFields(t *testing.T) {
	options := backends.LlamaServerOptions{
		Lora:               []string{"adapter1.bin", "adapter2.bin"},
		OverrideTensor:     []string{"tensor1", "tensor2", "tensor3"},
		DrySequenceBreaker: []string{".", "!", "?"},
	}

	args := options.BuildCommandArgs()

	// Check that each array value appears with its flag
	expectedOccurrences := map[string][]string{
		"--lora":                 {"adapter1.bin", "adapter2.bin"},
		"--override-tensor":      {"tensor1", "tensor2", "tensor3"},
		"--dry-sequence-breaker": {".", "!", "?"},
	}

	for flag, values := range expectedOccurrences {
		for _, value := range values {
			if !testutil.ContainsFlagWithValue(args, flag, value) {
				t.Errorf("Expected %s %s, not found in %v", flag, value, args)
			}
		}
	}
}

func TestLlamaCppBuildCommandArgs_EmptyArrays(t *testing.T) {
	options := backends.LlamaServerOptions{
		Lora:           []string{}, // Empty array should not generate args
		OverrideTensor: []string{}, // Empty array should not generate args
	}

	args := options.BuildCommandArgs()

	excludedArgs := []string{"--lora", "--override-tensor"}
	for _, excludedArg := range excludedArgs {
		if testutil.Contains(args, excludedArg) {
			t.Errorf("Empty array should not generate argument %q in %v", excludedArg, args)
		}
	}
}

func TestLlamaCppUnmarshalJSON_StandardFields(t *testing.T) {
	jsonData := `{
		"model": "/path/to/model.gguf",
		"port": 8080,
		"host": "localhost", 
		"verbose": true,
		"ctx_size": 4096,
		"gpu_layers": 32,
		"temp": 0.7
	}`

	var options backends.LlamaServerOptions
	err := json.Unmarshal([]byte(jsonData), &options)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if options.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", options.Model)
	}
	if options.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", options.Port)
	}
	if options.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %q", options.Host)
	}
	if !options.Verbose {
		t.Error("Expected verbose to be true")
	}
	if options.CtxSize != 4096 {
		t.Errorf("Expected ctx_size 4096, got %d", options.CtxSize)
	}
	if options.GPULayers != 32 {
		t.Errorf("Expected gpu_layers 32, got %d", options.GPULayers)
	}
	if options.Temperature != 0.7 {
		t.Errorf("Expected temperature 0.7, got %f", options.Temperature)
	}
}

func TestLlamaCppUnmarshalJSON_AlternativeFieldNames(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		checkFn  func(backends.LlamaServerOptions) error
	}{
		{
			name:     "threads alternatives",
			jsonData: `{"t": 4, "tb": 2}`,
			checkFn: func(opts backends.LlamaServerOptions) error {
				if opts.Threads != 4 {
					return fmt.Errorf("expected threads 4, got %d", opts.Threads)
				}
				if opts.ThreadsBatch != 2 {
					return fmt.Errorf("expected threads_batch 2, got %d", opts.ThreadsBatch)
				}
				return nil
			},
		},
		{
			name:     "context size alternatives",
			jsonData: `{"c": 2048}`,
			checkFn: func(opts backends.LlamaServerOptions) error {
				if opts.CtxSize != 2048 {
					return fmt.Errorf("expected ctx_size 4096, got %d", opts.CtxSize)
				}
				return nil
			},
		},
		{
			name:     "gpu layers alternatives",
			jsonData: `{"ngl": 16}`,
			checkFn: func(opts backends.LlamaServerOptions) error {
				if opts.GPULayers != 16 {
					return fmt.Errorf("expected gpu_layers 32, got %d", opts.GPULayers)
				}
				return nil
			},
		},
		{
			name:     "model alternatives",
			jsonData: `{"m": "/path/model.gguf"}`,
			checkFn: func(opts backends.LlamaServerOptions) error {
				if opts.Model != "/path/model.gguf" {
					return fmt.Errorf("expected model '/path/model.gguf', got %q", opts.Model)
				}
				return nil
			},
		},
		{
			name:     "temperature alternatives",
			jsonData: `{"temp": 0.8}`,
			checkFn: func(opts backends.LlamaServerOptions) error {
				if opts.Temperature != 0.8 {
					return fmt.Errorf("expected temperature 0.8, got %f", opts.Temperature)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var options backends.LlamaServerOptions
			err := json.Unmarshal([]byte(tt.jsonData), &options)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if err := tt.checkFn(options); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestLlamaCppUnmarshalJSON_InvalidJSON(t *testing.T) {
	invalidJSON := `{"port": "not-a-number", "invalid": syntax}`

	var options backends.LlamaServerOptions
	err := json.Unmarshal([]byte(invalidJSON), &options)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLlamaCppUnmarshalJSON_ArrayFields(t *testing.T) {
	jsonData := `{
		"lora": ["adapter1.bin", "adapter2.bin"],
		"override_tensor": ["tensor1", "tensor2"],
		"dry_sequence_breaker": [".", "!", "?"]
	}`

	var options backends.LlamaServerOptions
	err := json.Unmarshal([]byte(jsonData), &options)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	expectedLora := []string{"adapter1.bin", "adapter2.bin"}
	if !reflect.DeepEqual(options.Lora, expectedLora) {
		t.Errorf("Expected lora %v, got %v", expectedLora, options.Lora)
	}

	expectedTensors := []string{"tensor1", "tensor2"}
	if !reflect.DeepEqual(options.OverrideTensor, expectedTensors) {
		t.Errorf("Expected override_tensor %v, got %v", expectedTensors, options.OverrideTensor)
	}

	expectedBreakers := []string{".", "!", "?"}
	if !reflect.DeepEqual(options.DrySequenceBreaker, expectedBreakers) {
		t.Errorf("Expected dry_sequence_breaker %v, got %v", expectedBreakers, options.DrySequenceBreaker)
	}
}

func TestParseLlamaCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
		validate  func(*testing.T, *backends.LlamaServerOptions)
	}{
		{
			name:      "basic command",
			command:   "llama-server --model /path/to/model.gguf --gpu-layers 32",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.LlamaServerOptions) {
				if opts.Model != "/path/to/model.gguf" {
					t.Errorf("expected model '/path/to/model.gguf', got '%s'", opts.Model)
				}
				if opts.GPULayers != 32 {
					t.Errorf("expected gpu_layers 32, got %d", opts.GPULayers)
				}
			},
		},
		{
			name:      "args only",
			command:   "--model /path/to/model.gguf --ctx-size 4096",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.LlamaServerOptions) {
				if opts.Model != "/path/to/model.gguf" {
					t.Errorf("expected model '/path/to/model.gguf', got '%s'", opts.Model)
				}
				if opts.CtxSize != 4096 {
					t.Errorf("expected ctx_size 4096, got %d", opts.CtxSize)
				}
			},
		},
		{
			name:      "mixed flag formats",
			command:   "llama-server --model=/path/model.gguf --gpu-layers 16 --verbose",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.LlamaServerOptions) {
				if opts.Model != "/path/model.gguf" {
					t.Errorf("expected model '/path/model.gguf', got '%s'", opts.Model)
				}
				if opts.GPULayers != 16 {
					t.Errorf("expected gpu_layers 16, got %d", opts.GPULayers)
				}
				if !opts.Verbose {
					t.Errorf("expected verbose to be true")
				}
			},
		},
		{
			name:      "quoted strings",
			command:   `llama-server --model test.gguf --api-key "sk-1234567890abcdef"`,
			expectErr: false,
			validate: func(t *testing.T, opts *backends.LlamaServerOptions) {
				if opts.APIKey != "sk-1234567890abcdef" {
					t.Errorf("expected api_key 'sk-1234567890abcdef', got '%s'", opts.APIKey)
				}
			},
		},
		{
			name:      "multiple value types",
			command:   "llama-server --model /test/model.gguf --n-gpu-layers 32 --temp 0.7 --verbose --no-mmap",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.LlamaServerOptions) {
				if opts.Model != "/test/model.gguf" {
					t.Errorf("expected model '/test/model.gguf', got '%s'", opts.Model)
				}
				if opts.GPULayers != 32 {
					t.Errorf("expected gpu_layers 32, got %d", opts.GPULayers)
				}
				if opts.Temperature != 0.7 {
					t.Errorf("expected temperature 0.7, got %f", opts.Temperature)
				}
				if !opts.Verbose {
					t.Errorf("expected verbose to be true")
				}
				if !opts.NoMmap {
					t.Errorf("expected no_mmap to be true")
				}
			},
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
			var opts backends.LlamaServerOptions
			resultAny, err := opts.ParseCommand(tt.command)
			result, _ := resultAny.(*backends.LlamaServerOptions)

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

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestParseLlamaCommandArrays(t *testing.T) {
	command := "llama-server --model test.gguf --lora adapter1.bin --lora=adapter2.bin"
	var opts backends.LlamaServerOptions
	resultAny, err := opts.ParseCommand(command)
	result, _ := resultAny.(*backends.LlamaServerOptions)

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

func TestLlamaCppBuildCommandArgs_ExtraArgs(t *testing.T) {
	options := backends.LlamaServerOptions{
		Model: "/models/test.gguf",
		ExtraArgs: map[string]string{
			"flash-attn": "",               // boolean flag
			"log-file":   "/logs/test.log", // value flag
		},
	}

	args := options.BuildCommandArgs()

	// Check that extra args are present
	if !testutil.Contains(args, "--flash-attn") {
		t.Error("Expected --flash-attn flag not found")
	}
	if !testutil.Contains(args, "--log-file") || !testutil.Contains(args, "/logs/test.log") {
		t.Error("Expected --log-file flag or value not found")
	}
}
