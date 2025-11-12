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

	// ExtraArgs are additional command line arguments.
	// Example: {"verbose": "", "log-file": "/logs/mlx.log"}
	ExtraArgs map[string]string `json:"extra_args,omitempty"`
}

func (o *MlxServerOptions) GetPort() int {
	return o.Port
}

func (o *MlxServerOptions) SetPort(port int) {
	o.Port = port
}

func (o *MlxServerOptions) GetHost() string {
	return o.Host
}

func (o *MlxServerOptions) Validate() error {
	if o == nil {
		return validation.ValidationError(fmt.Errorf("MLX server options cannot be nil for MLX backend"))
	}

	if err := validation.ValidateStructStrings(o, ""); err != nil {
		return err
	}

	// Basic network validation for port
	if o.Port < 0 || o.Port > 65535 {
		return validation.ValidationError(fmt.Errorf("invalid port range: %d", o.Port))
	}

	// Validate extra_args keys and values
	for key, value := range o.ExtraArgs {
		if err := validation.ValidateStringForInjection(key); err != nil {
			return validation.ValidationError(fmt.Errorf("extra_args key %q: %w", key, err))
		}
		if value != "" {
			if err := validation.ValidateStringForInjection(value); err != nil {
				return validation.ValidationError(fmt.Errorf("extra_args value for %q: %w", key, err))
			}
		}
	}

	return nil
}

// BuildCommandArgs converts to command line arguments
func (o *MlxServerOptions) BuildCommandArgs() []string {
	multipleFlags := map[string]struct{}{} // MLX doesn't currently have []string fields
	args := BuildCommandArgs(o, multipleFlags)

	// Append extra args at the end
	args = append(args, convertExtraArgsToFlags(o.ExtraArgs)...)

	return args
}

func (o *MlxServerOptions) BuildDockerArgs() []string {
	return []string{}
}

// ParseCommand parses a mlx_lm.server command string into MlxServerOptions
// Supports multiple formats:
// 1. Full command: "mlx_lm.server --model model/path"
// 2. Full path: "/usr/local/bin/mlx_lm.server --model model/path"
// 3. Args only: "--model model/path --host 0.0.0.0"
// 4. Multiline commands with backslashes
func (o *MlxServerOptions) ParseCommand(command string) (any, error) {
	executableNames := []string{"mlx_lm.server"}
	var subcommandNames []string            // MLX has no subcommands
	multiValuedFlags := map[string]struct{}{} // MLX has no multi-valued flags

	var mlxOptions MlxServerOptions
	if err := parseCommand(command, executableNames, subcommandNames, multiValuedFlags, &mlxOptions); err != nil {
		return nil, err
	}

	return &mlxOptions, nil
}
