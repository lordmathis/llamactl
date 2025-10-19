package backends_test

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/testutil"
	"testing"
)

func TestParseVllmCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
		validate  func(*testing.T, *backends.VllmServerOptions)
	}{
		{
			name:      "basic vllm serve command",
			command:   "vllm serve microsoft/DialoGPT-medium",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.VllmServerOptions) {
				if opts.Model != "microsoft/DialoGPT-medium" {
					t.Errorf("expected model 'microsoft/DialoGPT-medium', got '%s'", opts.Model)
				}
			},
		},
		{
			name:      "serve only command",
			command:   "serve microsoft/DialoGPT-medium",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.VllmServerOptions) {
				if opts.Model != "microsoft/DialoGPT-medium" {
					t.Errorf("expected model 'microsoft/DialoGPT-medium', got '%s'", opts.Model)
				}
			},
		},
		{
			name:      "positional model with flags",
			command:   "vllm serve microsoft/DialoGPT-medium --tensor-parallel-size 2",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.VllmServerOptions) {
				if opts.Model != "microsoft/DialoGPT-medium" {
					t.Errorf("expected model 'microsoft/DialoGPT-medium', got '%s'", opts.Model)
				}
				if opts.TensorParallelSize != 2 {
					t.Errorf("expected tensor_parallel_size 2, got %d", opts.TensorParallelSize)
				}
			},
		},
		{
			name:      "model with path",
			command:   "vllm serve /path/to/model --gpu-memory-utilization 0.8",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.VllmServerOptions) {
				if opts.Model != "/path/to/model" {
					t.Errorf("expected model '/path/to/model', got '%s'", opts.Model)
				}
				if opts.GPUMemoryUtilization != 0.8 {
					t.Errorf("expected gpu_memory_utilization 0.8, got %f", opts.GPUMemoryUtilization)
				}
			},
		},
		{
			name:      "multiple value types",
			command:   "vllm serve test-model --tensor-parallel-size 4 --gpu-memory-utilization 0.8 --enable-log-outputs",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.VllmServerOptions) {
				if opts.Model != "test-model" {
					t.Errorf("expected model 'test-model', got '%s'", opts.Model)
				}
				if opts.TensorParallelSize != 4 {
					t.Errorf("expected tensor_parallel_size 4, got %d", opts.TensorParallelSize)
				}
				if opts.GPUMemoryUtilization != 0.8 {
					t.Errorf("expected gpu_memory_utilization 0.8, got %f", opts.GPUMemoryUtilization)
				}
				if !opts.EnableLogOutputs {
					t.Errorf("expected enable_log_outputs true, got %v", opts.EnableLogOutputs)
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
			command:   `vllm serve "unterminated`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := backends.ParseVllmCommand(tt.command)

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

func TestVllmBuildCommandArgs_BooleanFields(t *testing.T) {
	tests := []struct {
		name     string
		options  backends.VllmServerOptions
		expected []string
		excluded []string
	}{
		{
			name: "enable_log_outputs true",
			options: backends.VllmServerOptions{
				EnableLogOutputs: true,
			},
			expected: []string{"--enable-log-outputs"},
		},
		{
			name: "enable_log_outputs false",
			options: backends.VllmServerOptions{
				EnableLogOutputs: false,
			},
			excluded: []string{"--enable-log-outputs"},
		},
		{
			name: "multiple booleans",
			options: backends.VllmServerOptions{
				EnableLogOutputs:    true,
				TrustRemoteCode:     true,
				EnablePrefixCaching: true,
				DisableLogStats:     false,
			},
			expected: []string{"--enable-log-outputs", "--trust-remote-code", "--enable-prefix-caching"},
			excluded: []string{"--disable-log-stats"},
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

func TestVllmBuildCommandArgs_ZeroValues(t *testing.T) {
	options := backends.VllmServerOptions{
		Port:                 0,   // Should be excluded
		TensorParallelSize:   0,   // Should be excluded
		GPUMemoryUtilization: 0,   // Should be excluded
		Model:                "",  // Should be excluded (positional arg)
		Host:                 "",  // Should be excluded
		EnableLogOutputs:     false, // Should be excluded
	}

	args := options.BuildCommandArgs()

	// Zero values should not appear in arguments
	excludedArgs := []string{
		"--port", "0",
		"--tensor-parallel-size", "0",
		"--gpu-memory-utilization", "0",
		"--host", "",
		"--enable-log-outputs",
	}

	for _, excludedArg := range excludedArgs {
		if testutil.Contains(args, excludedArg) {
			t.Errorf("Zero value argument %q should not be present in %v", excludedArg, args)
		}
	}

	// Model should not be present as positional arg when empty
	if len(args) > 0 && args[0] == "" {
		t.Errorf("Empty model should not be present as positional argument")
	}
}

func TestVllmBuildCommandArgs_ArrayFields(t *testing.T) {
	options := backends.VllmServerOptions{
		AllowedOrigins: []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		Middleware:     []string{"middleware1", "middleware2", "middleware3"},
	}

	args := options.BuildCommandArgs()

	// Check that each array value appears with its flag
	expectedOccurrences := map[string][]string{
		"--allowed-origins": {"http://localhost:3000", "https://example.com"},
		"--allowed-methods": {"GET", "POST"},
		"--middleware":      {"middleware1", "middleware2", "middleware3"},
	}

	for flag, values := range expectedOccurrences {
		for _, value := range values {
			if !testutil.ContainsFlagWithValue(args, flag, value) {
				t.Errorf("Expected %s %s, not found in %v", flag, value, args)
			}
		}
	}
}

func TestVllmBuildCommandArgs_EmptyArrays(t *testing.T) {
	options := backends.VllmServerOptions{
		AllowedOrigins: []string{}, // Empty array should not generate args
		Middleware:     []string{}, // Empty array should not generate args
	}

	args := options.BuildCommandArgs()

	excludedArgs := []string{"--allowed-origins", "--middleware"}
	for _, excludedArg := range excludedArgs {
		if testutil.Contains(args, excludedArg) {
			t.Errorf("Empty array should not generate argument %q in %v", excludedArg, args)
		}
	}
}

func TestVllmBuildCommandArgs_PositionalModel(t *testing.T) {
	options := backends.VllmServerOptions{
		Model:                "microsoft/DialoGPT-medium",
		Port:                 8080,
		Host:                 "localhost",
		TensorParallelSize:   2,
		GPUMemoryUtilization: 0.8,
		EnableLogOutputs:     true,
	}

	args := options.BuildCommandArgs()

	// Check that model is the first positional argument (not a --model flag)
	if len(args) == 0 || args[0] != "microsoft/DialoGPT-medium" {
		t.Errorf("Expected model 'microsoft/DialoGPT-medium' as first positional argument, got args: %v", args)
	}

	// Check that --model flag is NOT present (since model should be positional)
	if testutil.Contains(args, "--model") {
		t.Errorf("Found --model flag, but model should be positional argument in args: %v", args)
	}

	// Check other flags
	if !testutil.ContainsFlagWithValue(args, "--tensor-parallel-size", "2") {
		t.Errorf("Expected --tensor-parallel-size 2 not found in %v", args)
	}
	if !testutil.ContainsFlagWithValue(args, "--gpu-memory-utilization", "0.8") {
		t.Errorf("Expected --gpu-memory-utilization 0.8 not found in %v", args)
	}
	if !testutil.Contains(args, "--enable-log-outputs") {
		t.Errorf("Expected --enable-log-outputs not found in %v", args)
	}
	if !testutil.ContainsFlagWithValue(args, "--host", "localhost") {
		t.Errorf("Expected --host localhost not found in %v", args)
	}
	if !testutil.ContainsFlagWithValue(args, "--port", "8080") {
		t.Errorf("Expected --port 8080 not found in %v", args)
	}
}
