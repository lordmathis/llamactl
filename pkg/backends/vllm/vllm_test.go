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
		Port:                 8080, // should be excluded
		Host:                 "localhost", // should be excluded
		TensorParallelSize:   2,
		GPUMemoryUtilization: 0.8,
		EnableLogOutputs:     true,
		APIKey:              []string{"key1", "key2"},
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
	if contains(args, "--host") || contains(args, "--port") {
		t.Errorf("Host and port should not be in command args, found in %v", args)
	}

	// Check array handling (multiple flags)
	apiKeyCount := 0
	for i := range args {
		if args[i] == "--api-key" {
			apiKeyCount++
		}
	}
	if apiKeyCount != 2 {
		t.Errorf("Expected 2 --api-key flags, got %d", apiKeyCount)
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

func TestNewVllmServerOptions(t *testing.T) {
	options := vllm.NewVllmServerOptions()

	if options == nil {
		t.Fatal("NewVllmServerOptions returned nil")
	}
	if options.Host != "127.0.0.1" {
		t.Errorf("Expected default host '127.0.0.1', got %q", options.Host)
	}
	if options.Port != 8000 {
		t.Errorf("Expected default port 8000, got %d", options.Port)
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