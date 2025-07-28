package llamactl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os/exec"
	"sync"
	"time"
)

type CreateInstanceOptions struct {
	// Auto restart
	AutoRestart *bool `json:"auto_restart,omitempty"`
	MaxRestarts *int  `json:"max_restarts,omitempty"`
	// RestartDelay duration in seconds
	RestartDelay *int `json:"restart_delay_seconds,omitempty"`

	LlamaServerOptions `json:",inline"`
}

// UnmarshalJSON implements custom JSON unmarshaling for CreateInstanceOptions
// This is needed because the embedded LlamaServerOptions has its own UnmarshalJSON
// which can interfere with proper unmarshaling of the pointer fields
func (c *CreateInstanceOptions) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a temporary struct without the embedded type
	type tempCreateOptions struct {
		AutoRestart  *bool `json:"auto_restart,omitempty"`
		MaxRestarts  *int  `json:"max_restarts,omitempty"`
		RestartDelay *int  `json:"restart_delay_seconds,omitempty"`
	}

	var temp tempCreateOptions
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy the pointer fields
	c.AutoRestart = temp.AutoRestart
	c.MaxRestarts = temp.MaxRestarts
	c.RestartDelay = temp.RestartDelay

	// Now unmarshal the embedded LlamaServerOptions
	if err := json.Unmarshal(data, &c.LlamaServerOptions); err != nil {
		return err
	}

	return nil
}

// Instance represents a running instance of the llama server
type Instance struct {
	Name           string                 `json:"name"`
	options        *CreateInstanceOptions `json:"-"`
	globalSettings *InstancesConfig

	// Status
	Running bool `json:"running"`

	// Creation time
	Created int64 `json:"created,omitempty"` // Unix timestamp when the instance was created

	// Logging file
	logger *InstanceLogger `json:"-"`

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
}

// validateAndCopyOptions validates and creates a deep copy of the provided options
// It applies validation rules and returns a safe copy
func validateAndCopyOptions(name string, options *CreateInstanceOptions) *CreateInstanceOptions {
	optionsCopy := &CreateInstanceOptions{}

	if options != nil {
		// Copy the embedded LlamaServerOptions
		optionsCopy.LlamaServerOptions = options.LlamaServerOptions

		// Copy and validate pointer fields
		if options.AutoRestart != nil {
			autoRestart := *options.AutoRestart
			optionsCopy.AutoRestart = &autoRestart
		}

		if options.MaxRestarts != nil {
			maxRestarts := *options.MaxRestarts
			if maxRestarts < 0 {
				log.Printf("Instance %s MaxRestarts value (%d) cannot be negative, setting to 0", name, maxRestarts)
				maxRestarts = 0
			}
			optionsCopy.MaxRestarts = &maxRestarts
		}

		if options.RestartDelay != nil {
			restartDelay := *options.RestartDelay
			if restartDelay < 0 {
				log.Printf("Instance %s RestartDelay value (%d) cannot be negative, setting to 0 seconds", name, restartDelay)
				restartDelay = 0
			}
			optionsCopy.RestartDelay = &restartDelay
		}
	}

	return optionsCopy
}

// applyDefaultOptions applies default values from global settings to any nil options
func applyDefaultOptions(options *CreateInstanceOptions, globalSettings *InstancesConfig) {
	if globalSettings == nil {
		return
	}

	if options.AutoRestart == nil {
		defaultAutoRestart := globalSettings.DefaultAutoRestart
		options.AutoRestart = &defaultAutoRestart
	}

	if options.MaxRestarts == nil {
		defaultMaxRestarts := globalSettings.DefaultMaxRestarts
		options.MaxRestarts = &defaultMaxRestarts
	}

	if options.RestartDelay == nil {
		defaultRestartDelay := globalSettings.DefaultRestartDelay
		options.RestartDelay = &defaultRestartDelay
	}
}

// NewInstance creates a new instance with the given name, log path, and options
func NewInstance(name string, globalSettings *InstancesConfig, options *CreateInstanceOptions) *Instance {
	// Validate and copy options
	optionsCopy := validateAndCopyOptions(name, options)
	// Apply defaults
	applyDefaultOptions(optionsCopy, globalSettings)
	// Create the instance logger
	logger := NewInstanceLogger(name, globalSettings.LogDirectory)

	return &Instance{
		Name:           name,
		options:        optionsCopy,
		globalSettings: globalSettings,
		logger:         logger,

		Running: false,

		Created: time.Now().Unix(),
	}
}

func (i *Instance) GetOptions() *CreateInstanceOptions {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.options
}

func (i *Instance) SetOptions(options *CreateInstanceOptions) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if options == nil {
		log.Println("Warning: Attempted to set nil options on instance", i.Name)
		return
	}

	// Validate and copy options and apply defaults
	optionsCopy := validateAndCopyOptions(i.Name, options)
	applyDefaultOptions(optionsCopy, i.globalSettings)

	i.options = optionsCopy
	// Clear the proxy so it gets recreated with new options
	i.proxy = nil
}

// GetProxy returns the reverse proxy for this instance, creating it if needed
func (i *Instance) GetProxy() (*httputil.ReverseProxy, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.proxy != nil {
		return i.proxy, nil
	}

	if i.options == nil {
		return nil, fmt.Errorf("instance %s has no options set", i.Name)
	}

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", i.options.Host, i.options.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL for instance %s: %w", i.Name, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove CORS headers from llama-server response to avoid conflicts
		// llamactl will add its own CORS headers
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Max-Age")
		resp.Header.Del("Access-Control-Expose-Headers")
		return nil
	}

	i.proxy = proxy

	return i.proxy, nil
}

// MarshalJSON implements json.Marshaler for Instance
func (i *Instance) MarshalJSON() ([]byte, error) {
	// Use read lock since we're only reading data
	i.mu.RLock()
	defer i.mu.RUnlock()

	// Create a temporary struct with exported fields for JSON marshalling
	temp := struct {
		Name    string                 `json:"name"`
		Options *CreateInstanceOptions `json:"options,omitempty"`
		Running bool                   `json:"running"`
	}{
		Name:    i.Name,
		Options: i.options,
		Running: i.Running,
	}

	return json.Marshal(temp)
}

// UnmarshalJSON implements json.Unmarshaler for Instance
func (i *Instance) UnmarshalJSON(data []byte) error {
	// Create a temporary struct for unmarshalling
	temp := struct {
		Name    string                 `json:"name"`
		Options *CreateInstanceOptions `json:"options,omitempty"`
		Running bool                   `json:"running"`
	}{}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Set the fields
	i.Name = temp.Name
	i.Running = temp.Running

	// Handle options with validation but no defaults
	if temp.Options != nil {
		i.options = validateAndCopyOptions(i.Name, temp.Options)
	}

	return nil
}
