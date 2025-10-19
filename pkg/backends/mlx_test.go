package backends_test

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/testutil"
	"testing"
)

func TestParseMlxCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
		validate  func(*testing.T, *backends.MlxServerOptions)
	}{
		{
			name:      "basic command",
			command:   "mlx_lm.server --model /path/to/model --host 0.0.0.0",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.MlxServerOptions) {
				if opts.Model != "/path/to/model" {
					t.Errorf("expected model '/path/to/model', got '%s'", opts.Model)
				}
				if opts.Host != "0.0.0.0" {
					t.Errorf("expected host '0.0.0.0', got '%s'", opts.Host)
				}
			},
		},
		{
			name:      "args only",
			command:   "--model /path/to/model --port 8080",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.MlxServerOptions) {
				if opts.Model != "/path/to/model" {
					t.Errorf("expected model '/path/to/model', got '%s'", opts.Model)
				}
				if opts.Port != 8080 {
					t.Errorf("expected port 8080, got %d", opts.Port)
				}
			},
		},
		{
			name:      "mixed flag formats",
			command:   "mlx_lm.server --model=/path/model --temp=0.7 --trust-remote-code",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.MlxServerOptions) {
				if opts.Model != "/path/model" {
					t.Errorf("expected model '/path/model', got '%s'", opts.Model)
				}
				if opts.Temp != 0.7 {
					t.Errorf("expected temp 0.7, got %f", opts.Temp)
				}
				if !opts.TrustRemoteCode {
					t.Errorf("expected trust_remote_code to be true")
				}
			},
		},
		{
			name:      "multiple value types",
			command:   "mlx_lm.server --model /test/model.mlx --port 8080 --temp 0.7 --trust-remote-code --log-level DEBUG",
			expectErr: false,
			validate: func(t *testing.T, opts *backends.MlxServerOptions) {
				if opts.Model != "/test/model.mlx" {
					t.Errorf("expected model '/test/model.mlx', got '%s'", opts.Model)
				}
				if opts.Port != 8080 {
					t.Errorf("expected port 8080, got %d", opts.Port)
				}
				if opts.Temp != 0.7 {
					t.Errorf("expected temp 0.7, got %f", opts.Temp)
				}
				if !opts.TrustRemoteCode {
					t.Errorf("expected trust_remote_code to be true")
				}
				if opts.LogLevel != "DEBUG" {
					t.Errorf("expected log_level 'DEBUG', got '%s'", opts.LogLevel)
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
			command:   `mlx_lm.server --model test.mlx --chat-template "unterminated`,
			expectErr: true,
		},
		{
			name:      "malformed flag",
			command:   "mlx_lm.server ---model test.mlx",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := backends.ParseMlxCommand(tt.command)

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

func TestMlxBuildCommandArgs_BooleanFields(t *testing.T) {
	tests := []struct {
		name     string
		options  backends.MlxServerOptions
		expected []string
		excluded []string
	}{
		{
			name: "trust_remote_code true",
			options: backends.MlxServerOptions{
				TrustRemoteCode: true,
			},
			expected: []string{"--trust-remote-code"},
		},
		{
			name: "trust_remote_code false",
			options: backends.MlxServerOptions{
				TrustRemoteCode: false,
			},
			excluded: []string{"--trust-remote-code"},
		},
		{
			name: "multiple booleans",
			options: backends.MlxServerOptions{
				TrustRemoteCode:        true,
				UseDefaultChatTemplate: true,
			},
			expected: []string{"--trust-remote-code", "--use-default-chat-template"},
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

func TestMlxBuildCommandArgs_ZeroValues(t *testing.T) {
	options := backends.MlxServerOptions{
		Port:      0,   // Should be excluded
		TopK:      0,   // Should be excluded
		Temp:      0,   // Should be excluded
		Model:     "",  // Should be excluded
		LogLevel:  "",  // Should be excluded
		TrustRemoteCode: false, // Should be excluded
	}

	args := options.BuildCommandArgs()

	// Zero values should not appear in arguments
	excludedArgs := []string{
		"--port", "0",
		"--top-k", "0",
		"--temp", "0",
		"--model", "",
		"--log-level", "",
		"--trust-remote-code",
	}

	for _, excludedArg := range excludedArgs {
		if testutil.Contains(args, excludedArg) {
			t.Errorf("Zero value argument %q should not be present in %v", excludedArg, args)
		}
	}
}