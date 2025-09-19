package instance

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/backends/mlx"
	"llamactl/pkg/backends/vllm"
	"llamactl/pkg/config"
	"log"
)

type CreateInstanceOptions struct {
	// Auto restart
	AutoRestart  *bool `json:"auto_restart,omitempty"`
	MaxRestarts  *int  `json:"max_restarts,omitempty"`
	RestartDelay *int  `json:"restart_delay,omitempty"` // seconds
	// On demand start
	OnDemandStart *bool `json:"on_demand_start,omitempty"`
	// Idle timeout
	IdleTimeout *int `json:"idle_timeout,omitempty"` // minutes

	BackendType    backends.BackendType `json:"backend_type"`
	BackendOptions map[string]any       `json:"backend_options,omitempty"`

	// Backend-specific options
	LlamaServerOptions *llamacpp.LlamaServerOptions `json:"-"`
	MlxServerOptions   *mlx.MlxServerOptions        `json:"-"`
	VllmServerOptions  *vllm.VllmServerOptions      `json:"-"`
}

// UnmarshalJSON implements custom JSON unmarshaling for CreateInstanceOptions
func (c *CreateInstanceOptions) UnmarshalJSON(data []byte) error {
	// Use anonymous struct to avoid recursion
	type Alias CreateInstanceOptions
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Parse backend-specific options
	switch c.BackendType {
	case backends.BackendTypeLlamaCpp:
		if c.BackendOptions != nil {
			// Convert map to JSON and then unmarshal to LlamaServerOptions
			optionsData, err := json.Marshal(c.BackendOptions)
			if err != nil {
				return fmt.Errorf("failed to marshal backend options: %w", err)
			}

			c.LlamaServerOptions = &llamacpp.LlamaServerOptions{}
			if err := json.Unmarshal(optionsData, c.LlamaServerOptions); err != nil {
				return fmt.Errorf("failed to unmarshal llama.cpp options: %w", err)
			}
		}
	case backends.BackendTypeMlxLm:
		if c.BackendOptions != nil {
			optionsData, err := json.Marshal(c.BackendOptions)
			if err != nil {
				return fmt.Errorf("failed to marshal backend options: %w", err)
			}

			c.MlxServerOptions = &mlx.MlxServerOptions{}
			if err := json.Unmarshal(optionsData, c.MlxServerOptions); err != nil {
				return fmt.Errorf("failed to unmarshal MLX options: %w", err)
			}
		}
	case backends.BackendTypeVllm:
		if c.BackendOptions != nil {
			optionsData, err := json.Marshal(c.BackendOptions)
			if err != nil {
				return fmt.Errorf("failed to marshal backend options: %w", err)
			}

			c.VllmServerOptions = &vllm.VllmServerOptions{}
			if err := json.Unmarshal(optionsData, c.VllmServerOptions); err != nil {
				return fmt.Errorf("failed to unmarshal vLLM options: %w", err)
			}
		}
	default:
		return fmt.Errorf("unknown backend type: %s", c.BackendType)
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for CreateInstanceOptions
func (c *CreateInstanceOptions) MarshalJSON() ([]byte, error) {
	// Use anonymous struct to avoid recursion
	type Alias CreateInstanceOptions
	aux := struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	// Convert backend-specific options back to BackendOptions map for JSON
	switch c.BackendType {
	case backends.BackendTypeLlamaCpp:
		if c.LlamaServerOptions != nil {
			data, err := json.Marshal(c.LlamaServerOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal llama server options: %w", err)
			}

			var backendOpts map[string]any
			if err := json.Unmarshal(data, &backendOpts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
			}

			aux.BackendOptions = backendOpts
		}
	case backends.BackendTypeMlxLm:
		if c.MlxServerOptions != nil {
			data, err := json.Marshal(c.MlxServerOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal MLX server options: %w", err)
			}

			var backendOpts map[string]any
			if err := json.Unmarshal(data, &backendOpts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
			}

			aux.BackendOptions = backendOpts
		}
	case backends.BackendTypeVllm:
		if c.VllmServerOptions != nil {
			data, err := json.Marshal(c.VllmServerOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal vLLM server options: %w", err)
			}

			var backendOpts map[string]any
			if err := json.Unmarshal(data, &backendOpts); err != nil {
				return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
			}

			aux.BackendOptions = backendOpts
		}
	}

	return json.Marshal(aux)
}

// ValidateAndApplyDefaults validates the instance options and applies constraints
func (c *CreateInstanceOptions) ValidateAndApplyDefaults(name string, globalSettings *config.InstancesConfig) {
	// Validate and apply constraints
	if c.MaxRestarts != nil && *c.MaxRestarts < 0 {
		log.Printf("Instance %s MaxRestarts value (%d) cannot be negative, setting to 0", name, *c.MaxRestarts)
		*c.MaxRestarts = 0
	}

	if c.RestartDelay != nil && *c.RestartDelay < 0 {
		log.Printf("Instance %s RestartDelay value (%d) cannot be negative, setting to 0 seconds", name, *c.RestartDelay)
		*c.RestartDelay = 0
	}

	if c.IdleTimeout != nil && *c.IdleTimeout < 0 {
		log.Printf("Instance %s IdleTimeout value (%d) cannot be negative, setting to 0 minutes", name, *c.IdleTimeout)
		*c.IdleTimeout = 0
	}

	// Apply defaults from global settings for nil fields
	if globalSettings != nil {
		if c.AutoRestart == nil {
			c.AutoRestart = &globalSettings.DefaultAutoRestart
		}
		if c.MaxRestarts == nil {
			c.MaxRestarts = &globalSettings.DefaultMaxRestarts
		}
		if c.RestartDelay == nil {
			c.RestartDelay = &globalSettings.DefaultRestartDelay
		}
		if c.OnDemandStart == nil {
			c.OnDemandStart = &globalSettings.DefaultOnDemandStart
		}
		if c.IdleTimeout == nil {
			defaultIdleTimeout := 0
			c.IdleTimeout = &defaultIdleTimeout
		}
	}
}

// BuildCommandArgs builds command line arguments for the backend
func (c *CreateInstanceOptions) BuildCommandArgs() []string {
	switch c.BackendType {
	case backends.BackendTypeLlamaCpp:
		if c.LlamaServerOptions != nil {
			return c.LlamaServerOptions.BuildCommandArgs()
		}
	case backends.BackendTypeMlxLm:
		if c.MlxServerOptions != nil {
			return c.MlxServerOptions.BuildCommandArgs()
		}
	case backends.BackendTypeVllm:
		if c.VllmServerOptions != nil {
			// Prepend "serve" as first argument
			args := []string{"serve"}
			args = append(args, c.VllmServerOptions.BuildCommandArgs()...)
			return args
		}
	}
	return []string{}
}
