package vllm_test

import (
	"encoding/json"
	"llamactl/pkg/backends/vllm"
	"slices"
	"testing"
)

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

	// Check core functionality
	if !containsFlagWithValue(args, "--model", "microsoft/DialoGPT-medium") {
		t.Errorf("Expected --model microsoft/DialoGPT-medium not found in %v", args)
	}
	if !containsFlagWithValue(args, "--tensor-parallel-size", "2") {
		t.Errorf("Expected --tensor-parallel-size 2 not found in %v", args)
	}
	if !contains(args, "--enable-log-outputs") {
		t.Errorf("Expected --enable-log-outputs not found in %v", args)
	}

	// Host and port should NOT be in the arguments (handled by llamactl)
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

func TestUnmarshalJSON(t *testing.T) {
	// Test both underscore and dash formats
	jsonData := `{
		"model": "test-model",
		"tensor_parallel_size": 4,
		"gpu-memory-utilization": 0.9,
		"enable-log-outputs": true
	}`

	var options vllm.VllmServerOptions
	err := json.Unmarshal([]byte(jsonData), &options)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if options.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got %q", options.Model)
	}
	if options.TensorParallelSize != 4 {
		t.Errorf("Expected tensor_parallel_size 4, got %d", options.TensorParallelSize)
	}
	if options.GPUMemoryUtilization != 0.9 {
		t.Errorf("Expected gpu_memory_utilization 0.9, got %f", options.GPUMemoryUtilization)
	}
	if !options.EnableLogOutputs {
		t.Errorf("Expected enable_log_outputs true, got %v", options.EnableLogOutputs)
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
