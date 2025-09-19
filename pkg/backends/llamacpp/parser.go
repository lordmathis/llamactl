package llamacpp

import (
	"llamactl/pkg/backends"
)

// ParseLlamaCommand parses a llama-server command string into LlamaServerOptions
// Supports multiple formats:
// 1. Full command: "llama-server --model file.gguf"
// 2. Full path: "/usr/local/bin/llama-server --model file.gguf"
// 3. Args only: "--model file.gguf --gpu-layers 32"
// 4. Multiline commands with backslashes
func ParseLlamaCommand(command string) (*LlamaServerOptions, error) {
	config := backends.CommandParserConfig{
		ExecutableNames: []string{"llama-server"},
		MultiValuedFlags: map[string]struct{}{
			"override_tensor":       {},
			"override_kv":           {},
			"lora":                  {},
			"lora_scaled":           {},
			"control_vector":        {},
			"control_vector_scaled": {},
			"dry_sequence_breaker":  {},
			"logit_bias":            {},
		},
	}

	var llamaOptions LlamaServerOptions
	if err := backends.ParseCommand(command, config, &llamaOptions); err != nil {
		return nil, err
	}

	return &llamaOptions, nil
}

