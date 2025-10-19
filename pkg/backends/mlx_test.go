package backends_test

import (
	"llamactl/pkg/backends"
	"testing"
)

func TestParseMlxCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		expectErr bool
	}{
		{
			name:      "basic command",
			command:   "mlx_lm.server --model /path/to/model --host 0.0.0.0",
			expectErr: false,
		},
		{
			name:      "args only",
			command:   "--model /path/to/model --port 8080",
			expectErr: false,
		},
		{
			name:      "mixed flag formats",
			command:   "mlx_lm.server --model=/path/model --temp=0.7 --trust-remote-code",
			expectErr: false,
		},
		{
			name:      "quoted strings",
			command:   `mlx_lm.server --model test.mlx --chat-template "User: {user}\nAssistant: "`,
			expectErr: false,
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
			}
		})
	}
}

func TestParseMlxCommandValues(t *testing.T) {
	command := "mlx_lm.server --model /test/model.mlx --port 8080 --temp 0.7 --trust-remote-code --log-level DEBUG"
	result, err := backends.ParseMlxCommand(command)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Model != "/test/model.mlx" {
		t.Errorf("expected model '/test/model.mlx', got '%s'", result.Model)
	}

	if result.Port != 8080 {
		t.Errorf("expected port 8080, got %d", result.Port)
	}

	if result.Temp != 0.7 {
		t.Errorf("expected temp 0.7, got %f", result.Temp)
	}

	if !result.TrustRemoteCode {
		t.Errorf("expected trust_remote_code to be true")
	}

	if result.LogLevel != "DEBUG" {
		t.Errorf("expected log_level 'DEBUG', got '%s'", result.LogLevel)
	}
}

func TestMlxBuildCommandArgs(t *testing.T) {
	options := &backends.MlxServerOptions{
		Model:           "/test/model.mlx",
		Host:            "127.0.0.1",
		Port:            8080,
		Temp:            0.7,
		TopP:            0.9,
		TopK:            40,
		MaxTokens:       2048,
		TrustRemoteCode: true,
		LogLevel:        "DEBUG",
		ChatTemplate:    "custom template",
	}

	args := options.BuildCommandArgs()

	// Check that all expected flags are present
	expectedFlags := map[string]string{
		"--model":         "/test/model.mlx",
		"--host":          "127.0.0.1",
		"--port":          "8080",
		"--log-level":     "DEBUG",
		"--chat-template": "custom template",
		"--temp":          "0.7",
		"--top-p":         "0.9",
		"--top-k":         "40",
		"--max-tokens":    "2048",
	}

	for i := 0; i < len(args); i++ {
		if args[i] == "--trust-remote-code" {
			continue // Boolean flag with no value
		}
		if args[i] == "--use-default-chat-template" {
			continue // Boolean flag with no value
		}

		if expectedValue, exists := expectedFlags[args[i]]; exists && i+1 < len(args) {
			if args[i+1] != expectedValue {
				t.Errorf("expected %s to have value %s, got %s", args[i], expectedValue, args[i+1])
			}
		}
	}

	// Check boolean flags
	foundTrustRemoteCode := false
	for _, arg := range args {
		if arg == "--trust-remote-code" {
			foundTrustRemoteCode = true
		}
	}
	if !foundTrustRemoteCode {
		t.Errorf("expected --trust-remote-code flag to be present")
	}
}
