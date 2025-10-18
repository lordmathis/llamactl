package backends

import (
	"fmt"
	"llamactl/pkg/config"
)

func init() {
	// Register this backend with the default registry
	GetDefaultRegistry().Register(NewVllmBackend())
}

// vllmOptionsProvider is a local interface to access VllmServerOptions
// without importing the instance package (avoiding circular dependency)
type vllmOptionsProvider interface {
	GetVllmServerOptions() *VllmServerOptions
}

// VllmBackend implements the Backend interface for vLLM
type VllmBackend struct{}

// NewVllmBackend creates a new vLLM backend instance
func NewVllmBackend() *VllmBackend {
	return &VllmBackend{}
}

// GetType returns the backend type
func (b *VllmBackend) GetType() BackendType {
	return BackendTypeVllm
}

// GetConfigKey returns the config key for vLLM
func (b *VllmBackend) GetConfigKey() string {
	return "vllm"
}

// GetPort extracts the port from backend-specific options
func (b *VllmBackend) GetPort(options any) int {
	if options == nil {
		return 0
	}
	provider, ok := options.(vllmOptionsProvider)
	if !ok {
		return 0
	}
	opts := provider.GetVllmServerOptions()
	if opts == nil {
		return 0
	}
	return opts.Port
}

// SetPort sets the port in backend-specific options
func (b *VllmBackend) SetPort(options any, port int) {
	if options == nil {
		return
	}
	provider, ok := options.(vllmOptionsProvider)
	if !ok {
		return
	}
	opts := provider.GetVllmServerOptions()
	if opts != nil {
		opts.Port = port
	}
}

// GetHost extracts the host from backend-specific options
func (b *VllmBackend) GetHost(options any) string {
	if options == nil {
		return ""
	}
	provider, ok := options.(vllmOptionsProvider)
	if !ok {
		return ""
	}
	opts := provider.GetVllmServerOptions()
	if opts == nil {
		return ""
	}
	return opts.Host
}

// BuildCommandArgs builds command line arguments
func (b *VllmBackend) BuildCommandArgs(options any) []string {
	if options == nil {
		return []string{}
	}
	provider, ok := options.(vllmOptionsProvider)
	if !ok {
		return []string{}
	}
	opts := provider.GetVllmServerOptions()
	if opts == nil {
		return []string{}
	}
	return opts.BuildCommandArgs()
}

// BuildDockerArgs builds Docker-specific arguments
func (b *VllmBackend) BuildDockerArgs(options any) []string {
	if options == nil {
		return []string{}
	}
	provider, ok := options.(vllmOptionsProvider)
	if !ok {
		return []string{}
	}
	opts := provider.GetVllmServerOptions()
	if opts == nil {
		return []string{}
	}
	return opts.BuildDockerArgs()
}

// ValidateOptions validates backend-specific options
func (b *VllmBackend) ValidateOptions(options any) error {
	if options == nil {
		return fmt.Errorf("vLLM options cannot be nil")
	}
	provider, ok := options.(vllmOptionsProvider)
	if !ok {
		return fmt.Errorf("invalid vLLM options type")
	}
	opts := provider.GetVllmServerOptions()
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
