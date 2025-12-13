package instance

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"llamactl/pkg/config"
)

// Instance represents a running instance of llama server
type Instance struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Created int64  `json:"created,omitempty"` // Unix timestamp when instance was created

	// Global configuration
	globalInstanceSettings *config.InstancesConfig
	globalBackendSettings  *config.BackendConfig
	globalNodesConfig      map[string]config.NodeConfig
	localNodeName          string `json:"-"` // Name of the local node for remote detection

	status  *status  `json:"-"`
	options *options `json:"-"`

	// Components (can be nil for remote instances)
	process *process `json:"-"`
	proxy   *proxy   `json:"-"`
	logger  *logger  `json:"-"`
}

// New creates a new instance with the given name, log path, options and local node name
func New(name string, globalConfig *config.AppConfig, opts *Options, onStatusChange func(oldStatus, newStatus Status)) *Instance {

	globalInstanceSettings := &globalConfig.Instances
	globalBackendSettings := &globalConfig.Backends
	globalNodesConfig := globalConfig.Nodes
	localNodeName := globalConfig.LocalNode

	// Validate and copy options
	opts.validateAndApplyDefaults(name, globalInstanceSettings)

	// Create status wrapper
	status := newStatus(Stopped)
	status.onStatusChange = onStatusChange

	// Create options wrapper
	options := newOptions(opts)

	instance := &Instance{
		ID:                     0, // Will be set by database
		Name:                   name,
		options:                options,
		globalInstanceSettings: globalInstanceSettings,
		globalBackendSettings:  globalBackendSettings,
		globalNodesConfig:      globalNodesConfig,
		localNodeName:          localNodeName,
		Created:                time.Now().Unix(),
		status:                 status,
	}

	var err error
	instance.proxy, err = newProxy(instance)
	if err != nil {
		log.Println("Warning: Failed to create proxy for instance", instance.Name, "-", err)
	}

	// Only create logger, proxy, and process for local instances
	if !instance.IsRemote() {
		logRotationConfig := &LogRotationConfig{
			Enabled:   globalInstanceSettings.LogRotationEnabled,
			MaxSizeMB: globalInstanceSettings.LogRotationMaxSize,
			Compress:  globalInstanceSettings.LogRotationCompress,
		}
		instance.logger = newLogger(
			name,
			globalInstanceSettings.LogsDir,
			logRotationConfig,
		)
		instance.process = newProcess(instance)
	}

	return instance
}

// Start starts the instance
func (i *Instance) Start() error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be started locally)", i.Name)
	}
	return i.process.start()
}

// Stop stops the instance
func (i *Instance) Stop() error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be stopped locally)", i.Name)
	}
	return i.process.stop()
}

// Restart restarts the instance
func (i *Instance) Restart() error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be restarted locally)", i.Name)
	}
	return i.process.restart()
}

// WaitForHealthy waits for the instance to become healthy
func (i *Instance) WaitForHealthy(timeout int) error {
	if i.process == nil {
		return fmt.Errorf("instance %s has no process component (remote instances cannot be health checked locally)", i.Name)
	}
	return i.process.waitForHealthy(timeout)
}

// GetOptions returns the current options
func (i *Instance) GetOptions() *Options {
	if i.options == nil {
		return nil
	}
	return i.options.get()
}

// GetStatus returns the current status
func (i *Instance) GetStatus() Status {
	if i.status == nil {
		return Stopped
	}
	return i.status.get()
}

// SetStatus sets the status
func (i *Instance) SetStatus(s Status) {
	if i.status != nil {
		i.status.set(s)
	}
}

// IsRunning returns true if the status is Running
func (i *Instance) IsRunning() bool {
	if i.status == nil {
		return false
	}
	return i.status.isRunning()
}

// SetOptions sets the options
func (i *Instance) SetOptions(opts *Options) {
	if opts == nil {
		log.Println("Warning: Attempted to set nil options on instance", i.Name)
		return
	}

	// Preserve the original nodes to prevent changing instance location
	if i.options != nil && i.options.get() != nil {
		opts.Nodes = i.options.get().Nodes
	}

	// Validate and copy options
	opts.validateAndApplyDefaults(i.Name, i.globalInstanceSettings)

	if i.options != nil {
		i.options.set(opts)
	}

	// Clear the proxy so it gets recreated with new options
	if i.proxy != nil {
		i.proxy.clear()
	}
}

// SetTimeProvider sets a custom time provider for testing
func (i *Instance) SetTimeProvider(tp TimeProvider) {
	if i.proxy != nil {
		i.proxy.setTimeProvider(tp)
	}
}

func (i *Instance) GetHost() string {
	if i.options == nil {
		return "localhost"
	}
	return i.options.GetHost()
}

func (i *Instance) GetPort() int {
	if i.options == nil {
		return 0
	}
	return i.options.GetPort()
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

// GetLogs retrieves the last n lines of logs from the instance
func (i *Instance) GetLogs(num_lines int) (string, error) {
	if i.logger == nil {
		return "", fmt.Errorf("instance %s has no logger (remote instances don't have logs)", i.Name)
	}
	return i.logger.getLogs(num_lines)
}

// LastRequestTime returns the last request time as a Unix timestamp
func (i *Instance) LastRequestTime() int64 {
	if i.proxy == nil {
		return 0
	}
	return i.proxy.getLastRequestTime()
}

// UpdateLastRequestTime updates the last request access time for the instance via proxy
func (i *Instance) UpdateLastRequestTime() {
	if i.proxy != nil {
		i.proxy.updateLastRequestTime()
	}
}

// ShouldTimeout checks if the instance should timeout based on idle time
func (i *Instance) ShouldTimeout() bool {
	if i.proxy == nil {
		return false
	}
	return i.proxy.shouldTimeout()
}

// GetInflightRequests returns the current number of inflight requests
func (i *Instance) GetInflightRequests() int32 {
	if i.proxy == nil {
		return 0
	}
	return i.proxy.getInflightRequests()
}

// ServeHTTP serves HTTP requests through the proxy with request tracking and shutdown handling
func (i *Instance) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	if i.proxy == nil {
		return fmt.Errorf("instance %s has no proxy component", i.Name)
	}
	return i.proxy.serveHTTP(w, r)
}

func (i *Instance) getCommand() string {
	opts := i.GetOptions()
	if opts == nil {
		return ""
	}

	return opts.BackendOptions.GetCommand(i.globalBackendSettings, opts.DockerEnabled, opts.CommandOverride)
}

func (i *Instance) buildCommandArgs() []string {
	opts := i.GetOptions()
	if opts == nil {
		return nil
	}

	return opts.BackendOptions.BuildCommandArgs(i.globalBackendSettings, opts.DockerEnabled)
}

func (i *Instance) buildEnvironment() map[string]string {
	opts := i.GetOptions()
	if opts == nil {
		return nil
	}

	return opts.BackendOptions.BuildEnvironment(i.globalBackendSettings, opts.DockerEnabled, opts.Environment)
}

// MarshalJSON implements json.Marshaler for Instance
func (i *Instance) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID      int      `json:"id"`
		Name    string   `json:"name"`
		Status  *status  `json:"status"`
		Created int64    `json:"created,omitempty"`
		Options *options `json:"options,omitempty"`
	}{
		ID:      i.ID,
		Name:    i.Name,
		Status:  i.status,
		Created: i.Created,
		Options: i.options,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Instance
func (i *Instance) UnmarshalJSON(data []byte) error {
	// Explicitly deserialize to match MarshalJSON format
	aux := &struct {
		ID      int      `json:"id"`
		Name    string   `json:"name"`
		Status  *status  `json:"status"`
		Created int64    `json:"created,omitempty"`
		Options *options `json:"options,omitempty"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Set the fields
	i.ID = aux.ID
	i.Name = aux.Name
	i.Created = aux.Created
	i.status = aux.Status
	i.options = aux.Options

	return nil
}
