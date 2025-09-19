package vllm

import (
	"llamactl/pkg/backends"
)

// ParseVllmCommand parses a vLLM serve command string into VllmServerOptions
// Supports multiple formats:
// 1. Full command: "vllm serve --model MODEL_NAME --other-args"
// 2. Full path: "/usr/local/bin/vllm serve --model MODEL_NAME"
// 3. Serve only: "serve --model MODEL_NAME --other-args"
// 4. Args only: "--model MODEL_NAME --other-args"
// 5. Multiline commands with backslashes
func ParseVllmCommand(command string) (*VllmServerOptions, error) {
	executableNames := []string{"vllm"}
	subcommandNames := []string{"serve"}
	multiValuedFlags := map[string]bool{
		"middleware":      true,
		"api_key":         true,
		"allowed_origins": true,
		"allowed_methods": true,
		"allowed_headers": true,
		"lora_modules":    true,
		"prompt_adapters": true,
	}

	var vllmOptions VllmServerOptions
	if err := backends.ParseCommand(command, executableNames, subcommandNames, multiValuedFlags, &vllmOptions); err != nil {
		return nil, err
	}

	return &vllmOptions, nil
}

