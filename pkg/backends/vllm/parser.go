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
	config := backends.CommandParserConfig{
		ExecutableNames: []string{"vllm"},
		SubcommandNames: []string{"serve"},
		MultiValuedFlags: map[string]struct{}{
			"middleware":      {},
			"api_key":         {},
			"allowed_origins": {},
			"allowed_methods": {},
			"allowed_headers": {},
			"lora_modules":    {},
			"prompt_adapters": {},
		},
	}

	var vllmOptions VllmServerOptions
	if err := backends.ParseCommand(command, config, &vllmOptions); err != nil {
		return nil, err
	}

	return &vllmOptions, nil
}

