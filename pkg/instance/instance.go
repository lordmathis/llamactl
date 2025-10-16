package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"log"
	"net/http/httputil"
	"os/exec"
	"sync"
	"time"
)

// Instance represents a running instance of the llama server
type Instance struct {
	// Immutable identity (no locking needed after creation)
	Name    string `json:"name"`
	Created int64  `json:"created,omitempty"` // Unix timestamp when the instance was created

	// Mutable state - each owns its own lock
	status  *status                `json:"-"` // unexported - status owns its lock
	options *CreateInstanceOptions `json:"-"`

	// Global configuration (read-only, no lock needed)
	globalInstanceSettings *config.InstancesConfig
	globalBackendSettings  *config.BackendConfig

	// Components (can be nil for remote instances or when stopped)
	logger *logger `json:"-"` // nil for remote instances
	proxy  *proxy  `json:"-"` // nil for remote instances

	// internal
	cmd      *exec.Cmd          `json:"-"` // Command to run the instance
	ctx      context.Context    `json:"-"` // Context for managing the instance lifecycle
	cancel   context.CancelFunc `json:"-"` // Function to cancel the context
	stdout   io.ReadCloser      `json:"-"` // Standard output stream
	stderr   io.ReadCloser      `json:"-"` // Standard error stream
	mu       sync.RWMutex       `json:"-"` // RWMutex for better read/write separation
	restarts int                `json:"-"` // Number of restarts

	// Restart control
	restartCancel context.CancelFunc `json:"-"` // Cancel function for pending restarts
	monitorDone   chan struct{}      `json:"-"` // Channel to signal monitor goroutine completion
}

// NewInstance creates a new instance with the given name, log path, and options
func NewInstance(name string, globalBackendSettings *config.BackendConfig, globalInstanceSettings *config.InstancesConfig, options *CreateInstanceOptions, onStatusChange func(oldStatus, newStatus Status)) *Instance {
	// Validate and copy options
	options.ValidateAndApplyDefaults(name, globalInstanceSettings)

	// Create the instance logger
	logger := NewLogger(name, globalInstanceSettings.LogsDir)

	// Create status wrapper
	status := newStatus(Stopped)
	status.onStatusChange = onStatusChange

	instance := &Instance{
		Name:                   name,
		options:                options,
		globalInstanceSettings: globalInstanceSettings,
		globalBackendSettings:  globalBackendSettings,
		logger:                 logger,
		Created:                time.Now().Unix(),
		status:                 status,
	}

	// Create Proxy component
	instance.proxy = NewProxy(instance)

	return instance
}

func (i *Instance) GetOptions() *CreateInstanceOptions {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.options
}

// GetStatus returns the current status, delegating to status component
func (i *Instance) GetStatus() Status {
	if i.status == nil {
		return Stopped
	}
	return i.status.Get()
}

// SetStatus sets the status, delegating to status component
func (i *Instance) SetStatus(s Status) {
	if i.status != nil {
		i.status.Set(s)
	}
}

// IsRunning returns true if the status is Running, delegating to status component
func (i *Instance) IsRunning() bool {
	if i.status == nil {
		return false
	}
	return i.status.IsRunning()
}

func (i *Instance) GetPort() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.options != nil {
		switch i.options.BackendType {
		case backends.BackendTypeLlamaCpp:
			if i.options.LlamaServerOptions != nil {
				return i.options.LlamaServerOptions.Port
			}
		case backends.BackendTypeMlxLm:
			if i.options.MlxServerOptions != nil {
				return i.options.MlxServerOptions.Port
			}
		case backends.BackendTypeVllm:
			if i.options.VllmServerOptions != nil {
				return i.options.VllmServerOptions.Port
			}
		}
	}
	return 0
}

func (i *Instance) GetHost() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.options != nil {
		switch i.options.BackendType {
		case backends.BackendTypeLlamaCpp:
			if i.options.LlamaServerOptions != nil {
				return i.options.LlamaServerOptions.Host
			}
		case backends.BackendTypeMlxLm:
			if i.options.MlxServerOptions != nil {
				return i.options.MlxServerOptions.Host
			}
		case backends.BackendTypeVllm:
			if i.options.VllmServerOptions != nil {
				return i.options.VllmServerOptions.Host
			}
		}
	}
	return ""
}

func (i *Instance) SetOptions(options *CreateInstanceOptions) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if options == nil {
		log.Println("Warning: Attempted to set nil options on instance", i.Name)
		return
	}

	// Validate and copy options
	options.ValidateAndApplyDefaults(i.Name, i.globalInstanceSettings)

	i.options = options

	// Clear the proxy so it gets recreated with new options
	if i.proxy != nil {
		i.proxy.clearProxy()
	}
}

// SetTimeProvider sets a custom time provider for testing
// Delegates to the Proxy component
func (i *Instance) SetTimeProvider(tp TimeProvider) {
	if i.proxy != nil {
		i.proxy.SetTimeProvider(tp)
	}
}

// GetProxy returns the reverse proxy for this instance, delegating to Proxy component
func (i *Instance) GetProxy() (*httputil.ReverseProxy, error) {
	if i.proxy == nil {
		return nil, fmt.Errorf("instance %s has no proxy component", i.Name)
	}
	return i.proxy.GetProxy()
}

// MarshalJSON implements json.Marshaler for Instance
func (i *Instance) MarshalJSON() ([]byte, error) {
	// Use read lock since we're only reading data
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Determine if docker is enabled for this instance's backend
	var dockerEnabled bool
	if i.options != nil {
		switch i.options.BackendType {
		case backends.BackendTypeLlamaCpp:
			if i.globalBackendSettings != nil && i.globalBackendSettings.LlamaCpp.Docker != nil && i.globalBackendSettings.LlamaCpp.Docker.Enabled {
				dockerEnabled = true
			}
		case backends.BackendTypeVllm:
			if i.globalBackendSettings != nil && i.globalBackendSettings.VLLM.Docker != nil && i.globalBackendSettings.VLLM.Docker.Enabled {
				dockerEnabled = true
			}
		case backends.BackendTypeMlxLm:
			// MLX does not support docker currently
		}
	}

	// Explicitly serialize to maintain backward compatible JSON format
	return json.Marshal(&struct {
		Name          string                 `json:"name"`
		Status        *status                `json:"status"`
		Created       int64                  `json:"created,omitempty"`
		Options       *CreateInstanceOptions `json:"options,omitempty"`
		DockerEnabled bool                   `json:"docker_enabled,omitempty"`
	}{
		Name:          i.Name,
		Status:        i.status,
		Created:       i.Created,
		Options:       i.options,
		DockerEnabled: dockerEnabled,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Instance
func (i *Instance) UnmarshalJSON(data []byte) error {
	// Explicitly deserialize to match MarshalJSON format
	aux := &struct {
		Name    string                 `json:"name"`
		Status  *status                `json:"status"`
		Created int64                  `json:"created,omitempty"`
		Options *CreateInstanceOptions `json:"options,omitempty"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Set the fields
	i.Name = aux.Name
	i.Created = aux.Created
	i.status = aux.Status

	// Handle options with validation and defaults
	if aux.Options != nil {
		aux.Options.ValidateAndApplyDefaults(i.Name, i.globalInstanceSettings)
		i.options = aux.Options
	}

	// Initialize fields that are not serialized or may be nil
	if i.status == nil {
		i.status = newStatus(Stopped)
	}
	if i.logger == nil && i.globalInstanceSettings != nil {
		i.logger = NewLogger(i.Name, i.globalInstanceSettings.LogsDir)
	}
	if i.proxy == nil {
		i.proxy = NewProxy(i)
	}

	return nil
}

func (i *Instance) IsRemote() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.options == nil {
		return false
	}

	return len(i.options.Nodes) > 0
}

func (i *Instance) GetLogs(num_lines int) (string, error) {
	return i.logger.GetLogs(num_lines)
}

// getBackendHostPort extracts the host and port from instance options
// Returns the configured host and port for the backend
func (i *Instance) getBackendHostPort() (string, int) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.options == nil {
		return "localhost", 0
	}

	var host string
	var port int
	switch i.options.BackendType {
	case backends.BackendTypeLlamaCpp:
		if i.options.LlamaServerOptions != nil {
			host = i.options.LlamaServerOptions.Host
			port = i.options.LlamaServerOptions.Port
		}
	case backends.BackendTypeMlxLm:
		if i.options.MlxServerOptions != nil {
			host = i.options.MlxServerOptions.Host
			port = i.options.MlxServerOptions.Port
		}
	case backends.BackendTypeVllm:
		if i.options.VllmServerOptions != nil {
			host = i.options.VllmServerOptions.Host
			port = i.options.VllmServerOptions.Port
		}
	}

	if host == "" {
		host = "localhost"
	}

	return host, port
}
