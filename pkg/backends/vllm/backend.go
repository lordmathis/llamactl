package vllm

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
)

func init() {
	// Register this backend with the default registry
	backends.GetDefaultRegistry().Register(NewVllmBackend())
}

// VllmBackend implements the Backend interface for vLLM
type VllmBackend struct{}

// NewVllmBackend creates a new vLLM backend instance
func NewVllmBackend() *VllmBackend {
	return &VllmBackend{}
}

// GetType returns the backend type
func (b *VllmBackend) GetType() backends.BackendType {
	return backends.BackendTypeVllm
}

// GetConfigKey returns the config key for vLLM
func (b *VllmBackend) GetConfigKey() string {
	return "vllm"
}

// GetPort extracts the port from backend-specific options
func (b *VllmBackend) GetPort(options backends.BackendOptions) int {
	if options == nil {
		return 0
	}
	opts, ok := options.GetVllmServerOptions().(*VllmServerOptions)
	if !ok || opts == nil {
		return 0
	}
	return opts.Port
}

// SetPort sets the port in backend-specific options
func (b *VllmBackend) SetPort(options backends.BackendOptions, port int) {
	if options == nil {
		return
	}
	opts, ok := options.GetVllmServerOptions().(*VllmServerOptions)
	if ok && opts != nil {
		opts.Port = port
	}
}

// GetHost extracts the host from backend-specific options
func (b *VllmBackend) GetHost(options backends.BackendOptions) string {
	if options == nil {
		return ""
	}
	opts, ok := options.GetVllmServerOptions().(*VllmServerOptions)
	if !ok || opts == nil {
		return ""
	}
	return opts.Host
}

// BuildCommandArgs builds command line arguments
func (b *VllmBackend) BuildCommandArgs(options backends.BackendOptions) []string {
	if options == nil {
		return []string{}
	}
	opts, ok := options.GetVllmServerOptions().(*VllmServerOptions)
	if !ok || opts == nil {
		return []string{}
	}
	return opts.BuildCommandArgs()
}

// BuildDockerArgs builds Docker-specific arguments
func (b *VllmBackend) BuildDockerArgs(options backends.BackendOptions) []string {
	if options == nil {
		return []string{}
	}
	opts, ok := options.GetVllmServerOptions().(*VllmServerOptions)
	if !ok || opts == nil {
		return []string{}
	}
	return opts.BuildDockerArgs()
}

// ValidateOptions validates backend-specific options
func (b *VllmBackend) ValidateOptions(options backends.BackendOptions) error {
	if options == nil {
		return fmt.Errorf("vLLM options cannot be nil")
	}
	opts, ok := options.GetVllmServerOptions().(*VllmServerOptions)
	if !ok {
		return fmt.Errorf("invalid vLLM options type")
	}
	if opts == nil {
		return fmt.Errorf("vLLM options cannot be nil")
	}

	// Validate port range
	if opts.Port < 0 || opts.Port > 65535 {
		return fmt.Errorf("invalid port range: %d", opts.Port)
	}

	return nil
}

// ParseCommand parses a command string into options
func (b *VllmBackend) ParseCommand(command string) (any, error) {
	return ParseVllmCommand(command)
}

// SupportsDocker returns true if the backend supports Docker
func (b *VllmBackend) SupportsDocker() bool {
	return true
}

// GetResponseHeaders returns the response headers configuration for the backend
func (b *VllmBackend) GetResponseHeaders(backendConfig *config.BackendSettings) map[string]string {
	if backendConfig == nil || backendConfig.ResponseHeaders == nil {
		return make(map[string]string)
	}
	return backendConfig.ResponseHeaders
}
