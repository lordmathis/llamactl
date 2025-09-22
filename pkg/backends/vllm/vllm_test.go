package vllm_test

import (
	"llamactl/pkg/backends/vllm"
	"slices"
	"testing"
)

func TestParseVllmCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
	}{
		{
			name:      "basic vllm serve command",
			command:   "vllm serve --model microsoft/DialoGPT-medium",
			expectErr: false,
		},
		{
			name:      "serve only command",
			command:   "serve --model microsoft/DialoGPT-medium",
			expectErr: false,
		},
		{
			name:      "args only",
			command:   "--model microsoft/DialoGPT-medium --tensor-parallel-size 2",
			expectErr: false,
		},
		{
			name:      "empty command",
			command:   "",
			expectErr: true,
		},
		{
			name:      "unterminated quote",
			command:   `vllm serve --model "unterminated`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := vllm.ParseVllmCommand(tt.command)

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

func TestParseVllmCommandValues(t *testing.T) {
	command := "vllm serve --model test-model --tensor-parallel-size 4 --gpu-memory-utilization 0.8 --enable-log-outputs"
	result, err := vllm.ParseVllmCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", result.Model)
	}
	if result.TensorParallelSize != 4 {
		t.Errorf("expected tensor_parallel_size 4, got %d", result.TensorParallelSize)
	}
	if result.GPUMemoryUtilization != 0.8 {
		t.Errorf("expected gpu_memory_utilization 0.8, got %f", result.GPUMemoryUtilization)
	}
	if !result.EnableLogOutputs {
		t.Errorf("expected enable_log_outputs true, got %v", result.EnableLogOutputs)
	}
}

func TestBuildCommandArgs(t *testing.T) {
	options := vllm.VllmServerOptions{
		Model:                "microsoft/DialoGPT-medium",
		Port:                 8080,
		Host:                 "localhost",
		TensorParallelSize:   2,
		GPUMemoryUtilization: 0.8,
		EnableLogOutputs:     true,
		AllowedOrigins:       []string{"http://localhost:3000", "https://example.com"},
	}

	args := options.BuildCommandArgs()

	// Check that model is the first positional argument (not a --model flag)
	if len(args) == 0 || args[0] != "microsoft/DialoGPT-medium" {
		t.Errorf("Expected model 'microsoft/DialoGPT-medium' as first positional argument, got args: %v", args)
	}

	// Check that --model flag is NOT present (since model should be positional)
	if contains(args, "--model") {
		t.Errorf("Found --model flag, but model should be positional argument in args: %v", args)
	}

	// Check other flags
	if !containsFlagWithValue(args, "--tensor-parallel-size", "2") {
		t.Errorf("Expected --tensor-parallel-size 2 not found in %v", args)
	}
	if !contains(args, "--enable-log-outputs") {
		t.Errorf("Expected --enable-log-outputs not found in %v", args)
	}
	if !contains(args, "--host") {
		t.Errorf("Expected --host not found in %v", args)
	}
	if !contains(args, "--port") {
		t.Errorf("Expected --port not found in %v", args)
	}

	// Check array handling (multiple flags)
	allowedOriginsCount := 0
	for i := range args {
		if args[i] == "--allowed-origins" {
			allowedOriginsCount++
		}
	}
	if allowedOriginsCount != 2 {
		t.Errorf("Expected 2 --allowed-origins flags, got %d", allowedOriginsCount)
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func containsFlagWithValue(args []string, flag, value string) bool {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) && args[i+1] == value {
			return true
		}
	}
	return false
}
