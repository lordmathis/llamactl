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
	executableNames := []string{"llama-server"}
	var subcommandNames []string // Llama has no subcommands
	multiValuedFlags := map[string]bool{
		"override_tensor":       true,
		"override_kv":           true,
		"lora":                  true,
		"lora_scaled":           true,
		"control_vector":        true,
		"control_vector_scaled": true,
		"dry_sequence_breaker":  true,
		"logit_bias":            true,
	}

	var llamaOptions LlamaServerOptions
	if err := backends.ParseCommand(command, executableNames, subcommandNames, multiValuedFlags, &llamaOptions); err != nil {
		return nil, err
	}

	return &llamaOptions, nil
}

