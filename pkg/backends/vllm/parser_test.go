package vllm

import (
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
			result, err := ParseVllmCommand(tt.command)

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
	result, err := ParseVllmCommand(command)

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