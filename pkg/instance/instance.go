package instance

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"log"
	"net/http/httputil"
	"time"
)

// Instance represents a running instance of the llama server
type Instance struct {
	// Immutable identity (no locking needed after creation)
	Name    string `json:"name"`
	Created int64  `json:"created,omitempty"` // Unix timestamp when the instance was created

	// Mutable state - each owns its own lock
	status  *status  `json:"-"` // unexported - status owns its lock
	options *options `json:"-"` // unexported - options owns its lock

	// Global configuration (read-only, no lock needed)
	globalInstanceSettings *config.InstancesConfig
	globalBackendSettings  *config.BackendConfig
	localNodeName          string `json:"-"` // Name of the local node for remote detection

	// Components (can be nil for remote instances or when stopped)
	process *process `json:"-"` // nil for remote instances, nil when stopped
	proxy   *proxy   `json:"-"` // nil for remote instances, created on demand
	logger  *logger  `json:"-"` // nil for remote instances
}

// New creates a new instance with the given name, log path, and options
func New(name string, globalBackendSettings *config.BackendConfig, globalInstanceSettings *config.InstancesConfig, opts *Options, localNodeName string, onStatusChange func(oldStatus, newStatus Status)) *Instance {
	// Validate and copy options
	opts.ValidateAndApplyDefaults(name, globalInstanceSettings)

	// Create the instance logger
	logger := NewLogger(name, globalInstanceSettings.LogsDir)

	// Create status wrapper
	status := newStatus(Stopped)
	status.onStatusChange = onStatusChange

	// Create options wrapper
	options := newOptions(opts)

	instance := &Instance{
		Name:                   name,
		options:                options,
		globalInstanceSettings: globalInstanceSettings,
		globalBackendSettings:  globalBackendSettings,
		localNodeName:          localNodeName,
		logger:                 logger,
		Created:                time.Now().Unix(),
		status:                 status,
	}

	// Create Proxy component
	instance.proxy = NewProxy(instance)

	// Create Process component (will be initialized on first Start)
	instance.process = newProcess(instance)

	return instance
}

// GetOptions returns the current options, delegating to options component
func (i *Instance) GetOptions() *Options {
	if i.options == nil {
		return nil
	}
	return i.options.Get()
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
	opts := i.GetOptions()
	if opts != nil {
		switch opts.BackendType {
		case backends.BackendTypeLlamaCpp:
			if opts.LlamaServerOptions != nil {
				return opts.LlamaServerOptions.Port
			}
		case backends.BackendTypeMlxLm:
			if opts.MlxServerOptions != nil {
				return opts.MlxServerOptions.Port
			}
		case backends.BackendTypeVllm:
			if opts.VllmServerOptions != nil {
				return opts.VllmServerOptions.Port
			}
		}
	}
	return 0
}

func (i *Instance) GetHost() string {
	opts := i.GetOptions()
	if opts != nil {
		switch opts.BackendType {
		case backends.BackendTypeLlamaCpp:
			if opts.LlamaServerOptions != nil {
				return opts.LlamaServerOptions.Host
			}
		case backends.BackendTypeMlxLm:
			if opts.MlxServerOptions != nil {
				return opts.MlxServerOptions.Host
			}
		case backends.BackendTypeVllm:
			if opts.VllmServerOptions != nil {
				return opts.VllmServerOptions.Host
			}
		}
	}
	return ""
}

// SetOptions sets the options, delegating to options component
func (i *Instance) SetOptions(opts *Options) {
	if opts == nil {
		log.Println("Warning: Attempted to set nil options on instance", i.Name)
		return
	}

	// Preserve the original nodes to prevent changing instance location
	if i.options != nil && i.options.Get() != nil && i.options.Get().Nodes != nil {
		opts.Nodes = i.options.Get().Nodes
	}

	// Validate and copy options
	opts.ValidateAndApplyDefaults(i.Name, i.globalInstanceSettings)

	if i.options != nil {
		i.options.Set(opts)
	}

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

	// Remote instances should not use local proxy - they are handled by RemoteInstanceProxy
	opts := i.GetOptions()
	if opts != nil && len(opts.Nodes) > 0 {
		if _, isLocal := opts.Nodes[i.localNodeName]; !isLocal {
			return nil, fmt.Errorf("instance %s is a remote instance and should not use local proxy", i.Name)
		}
	}

	return i.proxy.GetProxy()
}

// MarshalJSON implements json.Marshaler for Instance
func (i *Instance) MarshalJSON() ([]byte, error) {
	// Get options
	opts := i.GetOptions()

	// Determine if docker is enabled for this instance's backend
	var dockerEnabled bool
	if opts != nil {
		switch opts.BackendType {
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
		Name          string   `json:"name"`
		Status        *status  `json:"status"`
		Created       int64    `json:"created,omitempty"`
		Options       *options `json:"options,omitempty"`
		DockerEnabled bool     `json:"docker_enabled,omitempty"`
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
		Name    string   `json:"name"`
		Status  *status  `json:"status"`
		Created int64    `json:"created,omitempty"`
		Options *options `json:"options,omitempty"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Set the fields
	i.Name = aux.Name
	i.Created = aux.Created
	i.status = aux.Status
	i.options = aux.Options

	// Handle options with validation and defaults
	if i.options != nil {
		opts := i.options.Get()
		if opts != nil {
			opts.ValidateAndApplyDefaults(i.Name, i.globalInstanceSettings)
		}
	}

	// Initialize fields that are not serialized or may be nil
	if i.status == nil {
		i.status = newStatus(Stopped)
	}
	if i.options == nil {
		i.options = newOptions(&Options{})
	}

	// Only create logger, proxy, and process for non-remote instances
	// Remote instances are metadata only (no logger, proxy, or process)
	if !i.IsRemote() {
		if i.logger == nil && i.globalInstanceSettings != nil {
			i.logger = NewLogger(i.Name, i.globalInstanceSettings.LogsDir)
		}
		if i.proxy == nil {
			i.proxy = NewProxy(i)
		}
		if i.process == nil {
			i.process = newProcess(i)
		}
	}

	return nil
}

func (i *Instance) IsRemote() bool {
	opts := i.GetOptions()
	if opts == nil {
		return false
	}

	// If no nodes specified, it's a local instance
	if len(opts.Nodes) == 0 {
		return false
	}

	// If the local node is in the nodes map, treat it as a local instance
	if _, isLocal := opts.Nodes[i.localNodeName]; isLocal {
		return false
	}

	// Otherwise, it's a remote instance
	return true
}

func (i *Instance) GetLogs(num_lines int) (string, error) {
	if i.logger == nil {
		return "", fmt.Errorf("instance %s has no logger (remote instances don't have logs)", i.Name)
	}
	return i.logger.GetLogs(num_lines)
}

// Start starts the instance, delegating to process component
func (i *Instance) Start() error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be started locally)", i.Name)
	}
	return i.process.Start()
}

// Stop stops the instance, delegating to process component
func (i *Instance) Stop() error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be stopped locally)", i.Name)
	}
	return i.process.Stop()
}

// Restart restarts the instance, delegating to process component
func (i *Instance) Restart() error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be restarted locally)", i.Name)
	}
	return i.process.Restart()
}

// WaitForHealthy waits for the instance to become healthy, delegating to process component
func (i *Instance) WaitForHealthy(timeout int) error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be health checked locally)", i.Name)
	}
	return i.process.WaitForHealthy(timeout)
}

// LastRequestTime returns the last request time as a Unix timestamp
// Delegates to the Proxy component
func (i *Instance) LastRequestTime() int64 {
	if i.proxy == nil {
		return 0
	}
	return i.proxy.LastRequestTime()
}

// getBackendHostPort extracts the host and port from instance options
// Returns the configured host and port for the backend
func (i *Instance) getBackendHostPort() (string, int) {
	opts := i.GetOptions()
	if opts == nil {
		return "localhost", 0
	}

	var host string
	var port int
	switch opts.BackendType {
	case backends.BackendTypeLlamaCpp:
		if opts.LlamaServerOptions != nil {
			host = opts.LlamaServerOptions.Host
			port = opts.LlamaServerOptions.Port
		}
	case backends.BackendTypeMlxLm:
		if opts.MlxServerOptions != nil {
			host = opts.MlxServerOptions.Host
			port = opts.MlxServerOptions.Port
		}
	case backends.BackendTypeVllm:
		if opts.VllmServerOptions != nil {
			host = opts.VllmServerOptions.Host
			port = opts.VllmServerOptions.Port
		}
	}

	if host == "" {
		host = "localhost"
	}

	return host, port
}
