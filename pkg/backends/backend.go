package backends

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/validation"
	"maps"
)

type BackendType string

const (
	BackendTypeLlamaCpp BackendType = "llama_cpp"
	BackendTypeMlxLm    BackendType = "mlx_lm"
	BackendTypeVllm     BackendType = "vllm"
)

type backend interface {
	BuildCommandArgs() []string
	BuildDockerArgs() []string
	GetPort() int
	SetPort(int)
	GetHost() string
	Validate() error
}

type Options struct {
	BackendType    BackendType    `json:"backend_type"`
	BackendOptions map[string]any `json:"backend_options,omitempty"`

	// Backend-specific options
	LlamaServerOptions *LlamaServerOptions `json:"-"`
	MlxServerOptions   *MlxServerOptions   `json:"-"`
	VllmServerOptions  *VllmServerOptions  `json:"-"`
}

func (o *Options) UnmarshalJSON(data []byte) error {
	// Use anonymous struct to avoid recursion
	type Alias Options
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Parse backend-specific options
	switch o.BackendType {
	case BackendTypeLlamaCpp:
		if o.BackendOptions != nil {
			// Convert map to JSON and then unmarshal to LlamaServerOptions
			optionsData, err := json.Marshal(o.BackendOptions)
			if err != nil {
				return fmt.Errorf("failed to marshal backend options: %w", err)
			}

			o.LlamaServerOptions = &LlamaServerOptions{}
			if err := json.Unmarshal(optionsData, o.LlamaServerOptions); err != nil {
				return fmt.Errorf("failed to unmarshal llama.cpp options: %w", err)
			}
		}
	case BackendTypeMlxLm:
		if o.BackendOptions != nil {
			optionsData, err := json.Marshal(o.BackendOptions)
			if err != nil {
				return fmt.Errorf("failed to marshal backend options: %w", err)
			}

			o.MlxServerOptions = &MlxServerOptions{}
			if err := json.Unmarshal(optionsData, o.MlxServerOptions); err != nil {
				return fmt.Errorf("failed to unmarshal MLX options: %w", err)
			}
		}
	case BackendTypeVllm:
		if o.BackendOptions != nil {
			optionsData, err := json.Marshal(o.BackendOptions)
			if err != nil {
				return fmt.Errorf("failed to marshal backend options: %w", err)
			}

			o.VllmServerOptions = &VllmServerOptions{}
			if err := json.Unmarshal(optionsData, o.VllmServerOptions); err != nil {
				return fmt.Errorf("failed to unmarshal vLLM options: %w", err)
			}
		}
	}

	return nil
}

func (o *Options) MarshalJSON() ([]byte, error) {
	// Use anonymous struct to avoid recursion
	type Alias Options
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}

	// Prepare BackendOptions map
	if o.BackendOptions == nil {
		o.BackendOptions = make(map[string]any)
	}

	// Populate BackendOptions based on backend-specific options
	switch o.BackendType {
	case BackendTypeLlamaCpp:
		if o.LlamaServerOptions != nil {
			optionsData, err := json.Marshal(o.LlamaServerOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal llama.cpp options: %w", err)
			}
			if err := json.Unmarshal(optionsData, &o.BackendOptions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal llama.cpp options to map: %w", err)
			}
		}
	case BackendTypeMlxLm:
		if o.MlxServerOptions != nil {
			optionsData, err := json.Marshal(o.MlxServerOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal MLX options: %w", err)
			}
			if err := json.Unmarshal(optionsData, &o.BackendOptions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal MLX options to map: %w", err)
			}
		}
	case BackendTypeVllm:
		if o.VllmServerOptions != nil {
			optionsData, err := json.Marshal(o.VllmServerOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal vLLM options: %w", err)
			}
			if err := json.Unmarshal(optionsData, &o.BackendOptions); err != nil {
				return nil, fmt.Errorf("failed to unmarshal vLLM options to map: %w", err)
			}
		}
	}

	return json.Marshal(aux)
}

func (o *Options) getBackendSettings(backendConfig *config.BackendConfig) *config.BackendSettings {
	switch o.BackendType {
	case BackendTypeLlamaCpp:
		return &backendConfig.LlamaCpp
	case BackendTypeMlxLm:
		return &backendConfig.MLX
	case BackendTypeVllm:
		return &backendConfig.VLLM
	default:
		return nil
	}
}

// getBackend returns the actual backend implementation
func (o *Options) getBackend() backend {
	switch o.BackendType {
	case BackendTypeLlamaCpp:
		return o.LlamaServerOptions
	case BackendTypeMlxLm:
		return o.MlxServerOptions
	case BackendTypeVllm:
		return o.VllmServerOptions
	default:
		return nil
	}
}

func (o *Options) isDockerEnabled(backend *config.BackendSettings) bool {
	if backend.Docker != nil && backend.Docker.Enabled && o.BackendType != BackendTypeMlxLm {
		return true
	}
	return false
}

func (o *Options) IsDockerEnabled(backendConfig *config.BackendConfig) bool {
	backendSettings := o.getBackendSettings(backendConfig)
	return o.isDockerEnabled(backendSettings)
}

// GetCommand builds the command to run the backend
func (o *Options) GetCommand(backendConfig *config.BackendConfig) string {

	backendSettings := o.getBackendSettings(backendConfig)

	if o.isDockerEnabled(backendSettings) {
		return "docker"
	}

	return backendSettings.Command
}

// buildCommandArgs builds command line arguments for the backend
func (o *Options) BuildCommandArgs(backendConfig *config.BackendConfig) []string {

	var args []string

	backendSettings := o.getBackendSettings(backendConfig)
	backend := o.getBackend()
	if backend == nil {
		return args
	}

	if o.isDockerEnabled(backendSettings) {
		// For Docker, start with Docker args
		args = append(args, backendSettings.Docker.Args...)
		args = append(args, backendSettings.Docker.Image)
		args = append(args, backend.BuildDockerArgs()...)

	} else {
		// For native execution, start with backend args
		args = append(args, backendSettings.Args...)
		args = append(args, backend.BuildCommandArgs()...)
	}

	return args
}

// BuildEnvironment builds the environment variables for the backend process
func (o *Options) BuildEnvironment(backendConfig *config.BackendConfig, environment map[string]string) map[string]string {

	backendSettings := o.getBackendSettings(backendConfig)
	env := map[string]string{}

	if backendSettings.Environment != nil {
		maps.Copy(env, backendSettings.Environment)
	}

	if o.isDockerEnabled(backendSettings) {
		if backendSettings.Docker.Environment != nil {
			maps.Copy(env, backendSettings.Docker.Environment)
		}
	}

	if environment != nil {
		maps.Copy(env, environment)
	}

	return env
}

func (o *Options) GetPort() int {
	backend := o.getBackend()
	if backend != nil {
		return backend.GetPort()
	}
	return 0
}

func (o *Options) SetPort(port int) {
	backend := o.getBackend()
	if backend != nil {
		backend.SetPort(port)
	}
}

func (o *Options) GetHost() string {
	backend := o.getBackend()
	if backend != nil {
		return backend.GetHost()
	}
	return "localhost"
}

func (o *Options) GetResponseHeaders(backendConfig *config.BackendConfig) map[string]string {
	backendSettings := o.getBackendSettings(backendConfig)
	return backendSettings.ResponseHeaders
}

// ValidateInstanceOptions performs validation based on backend type
func (o *Options) ValidateInstanceOptions() error {
	backend := o.getBackend()
	if backend == nil {
		return validation.ValidationError(fmt.Errorf("backend options cannot be nil for backend type %s", o.BackendType))
	}

	return backend.Validate()
}
