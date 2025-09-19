package mlx_test

import (
	"llamactl/pkg/backends/mlx"
	"testing"
)

func TestBuildCommandArgs(t *testing.T) {
	options := &mlx.MlxServerOptions{
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
