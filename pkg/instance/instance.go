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

// TimeProvider interface allows for testing with mock time
type TimeProvider interface {
	Now() time.Time
}

// realTimeProvider implements TimeProvider using the actual time
type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time {
	return time.Now()
}

// Process represents a running instance of the llama server
type Process struct {
	Name                   string                 `json:"name"`
	options                *CreateInstanceOptions `json:"-"`
	globalInstanceSettings *config.InstancesConfig
	globalBackendSettings  *config.BackendConfig

	// Status
	Status         InstanceStatus `json:"status"`
	onStatusChange func(oldStatus, newStatus InstanceStatus)

	// Creation time
	Created int64 `json:"created,omitempty"` // Unix timestamp when the instance was created

	// Logging file
	logger *Logger `json:"-"`

	// Proxy component
	proxy *Proxy `json:"-"` // HTTP proxy and request tracking

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

	// Time provider for testing (kept for backward compatibility during refactor)
	timeProvider TimeProvider `json:"-"` // Time provider for testing
}

// NewInstance creates a new instance with the given name, log path, and options
func NewInstance(name string, globalBackendSettings *config.BackendConfig, globalInstanceSettings *config.InstancesConfig, options *CreateInstanceOptions, onStatusChange func(oldStatus, newStatus InstanceStatus)) *Process {
	// Validate and copy options
	options.ValidateAndApplyDefaults(name, globalInstanceSettings)

	// Create the instance logger
	logger := NewInstanceLogger(name, globalInstanceSettings.LogsDir)

	instance := &Process{
		Name:                   name,
		options:                options,
		globalInstanceSettings: globalInstanceSettings,
		globalBackendSettings:  globalBackendSettings,
		logger:                 logger,
		timeProvider:           realTimeProvider{},
		Created:                time.Now().Unix(),
		Status:                 Stopped,
		onStatusChange:         onStatusChange,
	}

	// Create Proxy component
	instance.proxy = NewProxy(instance)

	return instance
}

func (i *Process) GetOptions() *CreateInstanceOptions {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.options
}

func (i *Process) GetPort() int {
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

func (i *Process) GetHost() string {
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

func (i *Process) SetOptions(options *CreateInstanceOptions) {
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
func (i *Process) SetTimeProvider(tp TimeProvider) {
	i.timeProvider = tp
	if i.proxy != nil {
		i.proxy.SetTimeProvider(tp)
	}
}

// GetProxy returns the reverse proxy for this instance, delegating to Proxy component
func (i *Process) GetProxy() (*httputil.ReverseProxy, error) {
	if i.proxy == nil {
		return nil, fmt.Errorf("instance %s has no proxy component", i.Name)
	}
	return i.proxy.GetProxy()
}

// MarshalJSON implements json.Marshaler for Instance
func (i *Process) MarshalJSON() ([]byte, error) {
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

	// Use anonymous struct to avoid recursion
	type Alias Process
	return json.Marshal(&struct {
		*Alias
		Options       *CreateInstanceOptions `json:"options,omitempty"`
		DockerEnabled bool                   `json:"docker_enabled,omitempty"`
	}{
		Alias:         (*Alias)(i),
		Options:       i.options,
		DockerEnabled: dockerEnabled,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Instance
func (i *Process) UnmarshalJSON(data []byte) error {
	// Use anonymous struct to avoid recursion
	type Alias Process
	aux := &struct {
		*Alias
		Options *CreateInstanceOptions `json:"options,omitempty"`
	}{
		Alias: (*Alias)(i),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Handle options with validation and defaults
	if aux.Options != nil {
		aux.Options.ValidateAndApplyDefaults(i.Name, i.globalInstanceSettings)
		i.options = aux.Options
	}

	// Initialize fields that are not serialized
	if i.timeProvider == nil {
		i.timeProvider = realTimeProvider{}
	}
	if i.logger == nil && i.globalInstanceSettings != nil {
		i.logger = NewInstanceLogger(i.Name, i.globalInstanceSettings.LogsDir)
	}
	if i.proxy == nil {
		i.proxy = NewProxy(i)
	}

	return nil
}

func (i *Process) IsRemote() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	if i.options == nil {
		return false
	}

	return len(i.options.Nodes) > 0
}

func (i *Process) GetLogs(num_lines int) (string, error) {
	return i.logger.GetLogs(num_lines)
}
