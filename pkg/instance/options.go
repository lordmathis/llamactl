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
	"maps"
	"slices"
	"sync"
)

// Options contains the actual configuration (exported - this is the public API).
type Options struct {
	// Auto restart
	AutoRestart  *bool `json:"auto_restart,omitempty"`
	MaxRestarts  *int  `json:"max_restarts,omitempty"`
	RestartDelay *int  `json:"restart_delay,omitempty"` // seconds
	// On demand start
	OnDemandStart *bool `json:"on_demand_start,omitempty"`
	// Idle timeout
	IdleTimeout *int `json:"idle_timeout,omitempty"` // minutes
	//Environment variables
	Environment map[string]string `json:"environment,omitempty"`

	BackendType    backends.BackendType `json:"backend_type"`
	BackendOptions map[string]any       `json:"backend_options,omitempty"`

	Nodes map[string]struct{} `json:"-"`

	// Backend-specific options
	LlamaServerOptions *llamacpp.LlamaServerOptions `json:"-"`
	MlxServerOptions   *mlx.MlxServerOptions        `json:"-"`
	VllmServerOptions  *vllm.VllmServerOptions      `json:"-"`
}

// options wraps Options with thread-safe access (unexported).
type options struct {
	mu   sync.RWMutex
	opts *Options
}

// newOptions creates a new options wrapper with the given Options
func newOptions(opts *Options) *options {
	return &options{
		opts: opts,
	}
}

// get returns a copy of the current options
func (o *options) get() *Options {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.opts
}

// set updates the options
func (o *options) set(opts *Options) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.opts = opts
}

// getPort returns the port using the backend registry
func (o *options) getPort() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.opts == nil {
		return 0
	}
	backend, err := backends.GetDefaultRegistry().Get(o.opts.BackendType)
	if err != nil {
		return 0
	}
	return backend.GetPort(o.opts.GetBackendOptions())
}

// setPort sets the port using the backend registry
func (o *options) setPort(port int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.opts == nil {
		return
	}
	backend, err := backends.GetDefaultRegistry().Get(o.opts.BackendType)
	if err != nil {
		return
	}
	backend.SetPort(o.opts.GetBackendOptions(), port)
}

// getHost returns the host using the backend registry
func (o *options) getHost() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.opts == nil {
		return ""
	}
	backend, err := backends.GetDefaultRegistry().Get(o.opts.BackendType)
	if err != nil {
		return ""
	}
	return backend.GetHost(o.opts.GetBackendOptions())
}

// MarshalJSON implements json.Marshaler for options wrapper
func (o *options) MarshalJSON() ([]byte, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.opts.MarshalJSON()
}

// UnmarshalJSON implements json.Unmarshaler for options wrapper
func (o *options) UnmarshalJSON(data []byte) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.opts == nil {
		o.opts = &Options{}
	}
	return o.opts.UnmarshalJSON(data)
}

// UnmarshalJSON implements custom JSON unmarshaling for Options
func (c *Options) UnmarshalJSON(data []byte) error {
	// Use anonymous struct to avoid recursion
	type Alias Options
	aux := &struct {
		Nodes []string `json:"nodes,omitempty"` // Accept JSON array
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert nodes array to map
	if len(aux.Nodes) > 0 {
		c.Nodes = make(map[string]struct{}, len(aux.Nodes))
		for _, node := range aux.Nodes {
			c.Nodes[node] = struct{}{}
		}
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

// MarshalJSON implements custom JSON marshaling for Options
func (c *Options) MarshalJSON() ([]byte, error) {
	// Use anonymous struct to avoid recursion
	type Alias Options
	aux := struct {
		Nodes []string `json:"nodes,omitempty"` // Output as JSON array
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	// Convert nodes map to array (sorted for consistency)
	if len(c.Nodes) > 0 {
		aux.Nodes = make([]string, 0, len(c.Nodes))
		for node := range c.Nodes {
			aux.Nodes = append(aux.Nodes, node)
		}
		// Sort for consistent output
		slices.Sort(aux.Nodes)
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

// validateAndApplyDefaults validates the instance options and applies constraints
func (c *Options) validateAndApplyDefaults(name string, globalSettings *config.InstancesConfig) {
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

// getCommand builds the command to run the backend
func (c *Options) getCommand(backendConfig *config.BackendSettings) string {
	backend, err := backends.GetDefaultRegistry().Get(c.BackendType)
	if err != nil {
		return backendConfig.Command
	}

	if backendConfig.Docker != nil && backendConfig.Docker.Enabled && backend.SupportsDocker() {
		return "docker"
	}

	return backendConfig.Command
}

// buildCommandArgs builds command line arguments for the backend
func (c *Options) buildCommandArgs(backendConfig *config.BackendSettings) []string {
	backend, err := backends.GetDefaultRegistry().Get(c.BackendType)
	if err != nil {
		return []string{}
	}

	var args []string
	backendOpts := c.GetBackendOptions()

	if backendConfig.Docker != nil && backendConfig.Docker.Enabled && backend.SupportsDocker() {
		// For Docker, start with Docker args
		args = append(args, backendConfig.Docker.Args...)
		args = append(args, backendConfig.Docker.Image)
		args = append(args, backend.BuildDockerArgs(backendOpts)...)
	} else {
		// For native execution, start with backend args
		args = append(args, backendConfig.Args...)
		args = append(args, backend.BuildCommandArgs(backendOpts)...)
	}

	return args
}

// buildEnvironment builds the environment variables for the backend process
func (c *Options) buildEnvironment(backendConfig *config.BackendSettings) map[string]string {
	env := map[string]string{}

	if backendConfig.Environment != nil {
		maps.Copy(env, backendConfig.Environment)
	}

	backend, err := backends.GetDefaultRegistry().Get(c.BackendType)
	if err == nil && backendConfig.Docker != nil && backendConfig.Docker.Enabled && backend.SupportsDocker() {
		if backendConfig.Docker.Environment != nil {
			maps.Copy(env, backendConfig.Docker.Environment)
		}
	}

	if c.Environment != nil {
		maps.Copy(env, c.Environment)
	}

	return env
}

// GetBackendOptions returns the typed backend options based on BackendType
func (c *Options) GetBackendOptions() any {
	switch c.BackendType {
	case backends.BackendTypeLlamaCpp:
		return c.LlamaServerOptions
	case backends.BackendTypeMlxLm:
		return c.MlxServerOptions
	case backends.BackendTypeVllm:
		return c.VllmServerOptions
	default:
		return nil
	}
}
