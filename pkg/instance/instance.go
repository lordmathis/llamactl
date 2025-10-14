package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"sync"
	"sync/atomic"
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

	// internal
	cmd      *exec.Cmd              `json:"-"` // Command to run the instance
	ctx      context.Context        `json:"-"` // Context for managing the instance lifecycle
	cancel   context.CancelFunc     `json:"-"` // Function to cancel the context
	stdout   io.ReadCloser          `json:"-"` // Standard output stream
	stderr   io.ReadCloser          `json:"-"` // Standard error stream
	mu       sync.RWMutex           `json:"-"` // RWMutex for better read/write separation
	restarts int                    `json:"-"` // Number of restarts
	proxy    *httputil.ReverseProxy `json:"-"` // Reverse proxy for this instance

	// Restart control
	restartCancel context.CancelFunc `json:"-"` // Cancel function for pending restarts
	monitorDone   chan struct{}      `json:"-"` // Channel to signal monitor goroutine completion

	// Timeout management
	lastRequestTime atomic.Int64 // Unix timestamp of last request
	timeProvider    TimeProvider `json:"-"` // Time provider for testing
}

// NewInstance creates a new instance with the given name, log path, and options
func NewInstance(name string, globalBackendSettings *config.BackendConfig, globalInstanceSettings *config.InstancesConfig, options *CreateInstanceOptions, onStatusChange func(oldStatus, newStatus InstanceStatus)) *Process {
	// Validate and copy options
	options.ValidateAndApplyDefaults(name, globalInstanceSettings)

	// Create the instance logger
	logger := NewInstanceLogger(name, globalInstanceSettings.LogsDir)

	return &Process{
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
	i.proxy = nil
}

// SetTimeProvider sets a custom time provider for testing
func (i *Process) SetTimeProvider(tp TimeProvider) {
	i.timeProvider = tp
}

// GetProxy returns the reverse proxy for this instance, creating it if needed
func (i *Process) GetProxy() (*httputil.ReverseProxy, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.proxy != nil {
		return i.proxy, nil
	}

	if i.options == nil {
		return nil, fmt.Errorf("instance %s has no options set", i.Name)
	}

	// Remote instances should not use local proxy - they are handled by RemoteInstanceProxy
	if len(i.options.Nodes) > 0 {
		return nil, fmt.Errorf("instance %s is a remote instance and should not use local proxy", i.Name)
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

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL for instance %s: %w", i.Name, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	var responseHeaders map[string]string
	switch i.options.BackendType {
	case backends.BackendTypeLlamaCpp:
		responseHeaders = i.globalBackendSettings.LlamaCpp.ResponseHeaders
	case backends.BackendTypeVllm:
		responseHeaders = i.globalBackendSettings.VLLM.ResponseHeaders
	case backends.BackendTypeMlxLm:
		responseHeaders = i.globalBackendSettings.MLX.ResponseHeaders
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove CORS headers from llama-server response to avoid conflicts
		// llamactl will add its own CORS headers
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Max-Age")
		resp.Header.Del("Access-Control-Expose-Headers")

		for key, value := range responseHeaders {
			resp.Header.Set(key, value)
		}
		return nil
	}

	i.proxy = proxy

	return i.proxy, nil
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
