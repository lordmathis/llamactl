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
	// BackendTypeMlxVlm BackendType = "mlx_vlm"  // Future expansion
)

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

func getBackendSettings(o *Options, backendConfig *config.BackendConfig) *config.BackendSettings {
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

func (o *Options) isDockerEnabled(backend *config.BackendSettings) bool {
	if backend.Docker != nil && backend.Docker.Enabled && o.BackendType != BackendTypeMlxLm {
		return true
	}
	return false
}

func (o *Options) IsDockerEnabled(backendConfig *config.BackendConfig) bool {
	backendSettings := getBackendSettings(o, backendConfig)
	return o.isDockerEnabled(backendSettings)
}

// GetCommand builds the command to run the backend
func (o *Options) GetCommand(backendConfig *config.BackendConfig) string {

	backendSettings := getBackendSettings(o, backendConfig)

	if o.isDockerEnabled(backendSettings) {
		return "docker"
	}

	return backendSettings.Command
}

// buildCommandArgs builds command line arguments for the backend
func (o *Options) BuildCommandArgs(backendConfig *config.BackendConfig) []string {

	var args []string

	backendSettings := getBackendSettings(o, backendConfig)

	if o.isDockerEnabled(backendSettings) {
		// For Docker, start with Docker args
		args = append(args, backendSettings.Docker.Args...)
		args = append(args, backendSettings.Docker.Image)

		switch o.BackendType {
		case BackendTypeLlamaCpp:
			if o.LlamaServerOptions != nil {
				args = append(args, o.LlamaServerOptions.BuildDockerArgs()...)
			}
		case BackendTypeVllm:
			if o.VllmServerOptions != nil {
				args = append(args, o.VllmServerOptions.BuildDockerArgs()...)
			}
		}

	} else {
		// For native execution, start with backend args
		args = append(args, backendSettings.Args...)

		switch o.BackendType {
		case BackendTypeLlamaCpp:
			if o.LlamaServerOptions != nil {
				args = append(args, o.LlamaServerOptions.BuildCommandArgs()...)
			}
		case BackendTypeMlxLm:
			if o.MlxServerOptions != nil {
				args = append(args, o.MlxServerOptions.BuildCommandArgs()...)
			}
		case BackendTypeVllm:
			if o.VllmServerOptions != nil {
				args = append(args, o.VllmServerOptions.BuildCommandArgs()...)
			}
		}
	}

	return args
}

// BuildEnvironment builds the environment variables for the backend process
func (o *Options) BuildEnvironment(backendConfig *config.BackendConfig, environment map[string]string) map[string]string {

	backendSettings := getBackendSettings(o, backendConfig)
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
	if o != nil {
		switch o.BackendType {
		case BackendTypeLlamaCpp:
			if o.LlamaServerOptions != nil {
				return o.LlamaServerOptions.Port
			}
		case BackendTypeMlxLm:
			if o.MlxServerOptions != nil {
				return o.MlxServerOptions.Port
			}
		case BackendTypeVllm:
			if o.VllmServerOptions != nil {
				return o.VllmServerOptions.Port
			}
		}
	}
	return 0
}

func (o *Options) SetPort(port int) {
	if o != nil {
		switch o.BackendType {
		case BackendTypeLlamaCpp:
			if o.LlamaServerOptions != nil {
				o.LlamaServerOptions.Port = port
			}
		case BackendTypeMlxLm:
			if o.MlxServerOptions != nil {
				o.MlxServerOptions.Port = port
			}
		case BackendTypeVllm:
			if o.VllmServerOptions != nil {
				o.VllmServerOptions.Port = port
			}
		}
	}
}

func (o *Options) GetHost() string {
	if o != nil {
		switch o.BackendType {
		case BackendTypeLlamaCpp:
			if o.LlamaServerOptions != nil {
				return o.LlamaServerOptions.Host
			}
		case BackendTypeMlxLm:
			if o.MlxServerOptions != nil {
				return o.MlxServerOptions.Host
			}
		case BackendTypeVllm:
			if o.VllmServerOptions != nil {
				return o.VllmServerOptions.Host
			}
		}
	}
	return "localhost"
}

func (o *Options) GetResponseHeaders(backendConfig *config.BackendConfig) map[string]string {
	backendSettings := getBackendSettings(o, backendConfig)
	return backendSettings.ResponseHeaders
}

// ValidateInstanceOptions performs validation based on backend type
func (o *Options) ValidateInstanceOptions() error {
	// Validate based on backend type
	switch o.BackendType {
	case BackendTypeLlamaCpp:
		return validateLlamaCppOptions(o.LlamaServerOptions)
	case BackendTypeMlxLm:
		return validateMlxOptions(o.MlxServerOptions)
	case BackendTypeVllm:
		return validateVllmOptions(o.VllmServerOptions)
	default:
		return validation.ValidationError(fmt.Errorf("unsupported backend type: %s", o.BackendType))
	}
}
