package mlx

import (
	"context"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"os/exec"
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
	return backends.BuildCommandArgs(o, multipleFlags)
}

// BuildCommandArgsWithDocker converts to command line arguments,
// handling Docker transformations if needed
func (o *MlxServerOptions) BuildCommandArgsWithDocker(dockerImage string) []string {
	args := o.BuildCommandArgs()

	// No special Docker transformations needed for MLX
	return args
}

// BuildCommand creates the complete command for execution, handling Docker vs native execution
func (o *MlxServerOptions) BuildCommand(ctx context.Context, backendConfig *config.BackendSettings) (*exec.Cmd, error) {
	// Build instance-specific arguments using backend functions
	var instanceArgs []string
	if backendConfig.Docker != nil && backendConfig.Docker.Enabled {
		// Use Docker-aware argument building
		instanceArgs = o.BuildCommandArgsWithDocker(backendConfig.Docker.Image)
	} else {
		// Use regular argument building for native execution
		instanceArgs = o.BuildCommandArgs()
	}

	// Combine backend args with instance args
	finalArgs := append(backendConfig.Args, instanceArgs...)

	// Choose Docker vs Native execution
	if backendConfig.Docker != nil && backendConfig.Docker.Enabled {
		return buildDockerCommand(ctx, backendConfig, finalArgs)
	} else {
		return exec.CommandContext(ctx, backendConfig.Command, finalArgs...), nil
	}
}

// buildDockerCommand builds a Docker command with the specified configuration and arguments
func buildDockerCommand(ctx context.Context, backendConfig *config.BackendSettings, args []string) (*exec.Cmd, error) {
	// Start with configured Docker arguments (should include "run", "--rm", etc.)
	dockerArgs := make([]string, len(backendConfig.Docker.Args))
	copy(dockerArgs, backendConfig.Docker.Args)

	// Add environment variables
	for key, value := range backendConfig.Docker.Environment {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add image and container arguments
	dockerArgs = append(dockerArgs, backendConfig.Docker.Image)
	dockerArgs = append(dockerArgs, args...)

	return exec.CommandContext(ctx, "docker", dockerArgs...), nil
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
	if err := backends.ParseCommand(command, executableNames, subcommandNames, multiValuedFlags, &mlxOptions); err != nil {
		return nil, err
	}

	return &mlxOptions, nil
}
