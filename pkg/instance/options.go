package instance

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"log"
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
	// Environment variables
	Environment map[string]string `json:"environment,omitempty"`
	// Assigned nodes
	Nodes map[string]struct{} `json:"-"`
	// Backend options
	BackendOptions backends.Options `json:"-"`
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

func (o *options) GetHost() string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.opts.BackendOptions.GetHost()
}

func (o *options) GetPort() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.opts.BackendOptions.GetPort()
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
		Nodes          []string             `json:"nodes,omitempty"`
		BackendType    backends.BackendType `json:"backend_type"`
		BackendOptions map[string]any       `json:"backend_options,omitempty"`
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

	// Create backend options struct and unmarshal
	c.BackendOptions = backends.Options{
		BackendType:    aux.BackendType,
		BackendOptions: aux.BackendOptions,
	}

	// Marshal the backend options to JSON for proper unmarshaling
	backendJson, err := json.Marshal(struct {
		BackendType    backends.BackendType `json:"backend_type"`
		BackendOptions map[string]any       `json:"backend_options,omitempty"`
	}{
		BackendType:    aux.BackendType,
		BackendOptions: aux.BackendOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal backend options: %w", err)
	}

	// Unmarshal into the backends.Options struct to trigger its custom unmarshaling
	if err := json.Unmarshal(backendJson, &c.BackendOptions); err != nil {
		return fmt.Errorf("failed to unmarshal backend options: %w", err)
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for Options
func (c *Options) MarshalJSON() ([]byte, error) {
	// Use anonymous struct to avoid recursion
	type Alias Options
	aux := struct {
		Nodes          []string             `json:"nodes,omitempty"` // Output as JSON array
		BackendType    backends.BackendType `json:"backend_type"`
		BackendOptions map[string]any       `json:"backend_options,omitempty"`
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

	// Set backend type
	aux.BackendType = c.BackendOptions.BackendType

	// Marshal the backends.Options struct to get the properly formatted backend options
	backendData, err := json.Marshal(c.BackendOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backend options: %w", err)
	}

	// Unmarshal into a temporary struct to extract the backend_options map
	var tempBackend struct {
		BackendOptions map[string]any `json:"backend_options,omitempty"`
	}
	if err := json.Unmarshal(backendData, &tempBackend); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backend data: %w", err)
	}

	aux.BackendOptions = tempBackend.BackendOptions

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
