package backends

import (
	"fmt"
	"llamactl/pkg/validation"
)

type MlxServerOptions struct {
	// Basic connection options
	Model string `json:"model,omitempty"`
	Host  string `json:"host,omitempty"`
	Port  int    `json:"port,omitempty"`

	// Model and adapter options
	AdapterPath     string `json:"adapter_path,omitempty"`
	DraftModel      string `json:"draft_model,omitempty"`
	NumDraftTokens  int    `json:"num_draft_tokens,omitempty"`
	TrustRemoteCode bool   `json:"trust_remote_code,omitempty"`

	// Logging and templates
	LogLevel               string `json:"log_level,omitempty"`
	ChatTemplate           string `json:"chat_template,omitempty"`
	UseDefaultChatTemplate bool   `json:"use_default_chat_template,omitempty"`
	ChatTemplateArgs       string `json:"chat_template_args,omitempty"` // JSON string

	// Sampling defaults
	Temp      float64 `json:"temp,omitempty"`
	TopP      float64 `json:"top_p,omitempty"`
	TopK      int     `json:"top_k,omitempty"`
	MinP      float64 `json:"min_p,omitempty"`
	MaxTokens int     `json:"max_tokens,omitempty"`
}

// BuildCommandArgs converts to command line arguments
func (o *MlxServerOptions) BuildCommandArgs() []string {
	multipleFlags := map[string]bool{} // MLX doesn't currently have []string fields
	return BuildCommandArgs(o, multipleFlags)
}

// ParseMlxCommand parses a mlx_lm.server command string into MlxServerOptions
// Supports multiple formats:
// 1. Full command: "mlx_lm.server --model model/path"
// 2. Full path: "/usr/local/bin/mlx_lm.server --model model/path"
// 3. Args only: "--model model/path --host 0.0.0.0"
// 4. Multiline commands with backslashes
func ParseMlxCommand(command string) (*MlxServerOptions, error) {
	executableNames := []string{"mlx_lm.server"}
	var subcommandNames []string          // MLX has no subcommands
	multiValuedFlags := map[string]bool{} // MLX has no multi-valued flags

	var mlxOptions MlxServerOptions
	if err := ParseCommand(command, executableNames, subcommandNames, multiValuedFlags, &mlxOptions); err != nil {
		return nil, err
	}

	return &mlxOptions, nil
}

// validateMlxOptions validates MLX backend specific options
func validateMlxOptions(options *MlxServerOptions) error {
	if options == nil {
		return validation.ValidationError(fmt.Errorf("MLX server options cannot be nil for MLX backend"))
	}

	if err := validation.ValidateStructStrings(options, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if options.Port < 0 || options.Port > 65535 {
		return validation.ValidationError(fmt.Errorf("invalid port range: %d", options.Port))
	}

	return nil
}
