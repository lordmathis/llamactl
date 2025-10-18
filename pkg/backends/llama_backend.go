package backends

import (
	"fmt"
	"llamactl/pkg/config"
)

func init() {
	// Register this backend with the default registry
	GetDefaultRegistry().Register(NewLlamaCppBackend())
}

// llamaOptionsProvider is a local interface to access LlamaServerOptions
// without importing the instance package (avoiding circular dependency)
type llamaOptionsProvider interface {
	GetLlamaServerOptions() *LlamaServerOptions
}

// LlamaCppBackend implements the Backend interface for llama.cpp
type LlamaCppBackend struct{}

// NewLlamaCppBackend creates a new llama.cpp backend instance
func NewLlamaCppBackend() *LlamaCppBackend {
	return &LlamaCppBackend{}
}

// GetType returns the backend type
func (b *LlamaCppBackend) GetType() BackendType {
	return BackendTypeLlamaCpp
}

// GetConfigKey returns the config key for llama.cpp
func (b *LlamaCppBackend) GetConfigKey() string {
	return "llama-cpp"
}

// GetPort extracts the port from backend-specific options
func (b *LlamaCppBackend) GetPort(options any) int {
	if options == nil {
		return 0
	}
	provider, ok := options.(llamaOptionsProvider)
	if !ok {
		return 0
	}
	opts := provider.GetLlamaServerOptions()
	if opts == nil {
		return 0
	}
	return opts.Port
}

// SetPort sets the port in backend-specific options
func (b *LlamaCppBackend) SetPort(options any, port int) {
	if options == nil {
		return
	}
	provider, ok := options.(llamaOptionsProvider)
	if !ok {
		return
	}
	opts := provider.GetLlamaServerOptions()
	if opts != nil {
		opts.Port = port
	}
}

// GetHost extracts the host from backend-specific options
func (b *LlamaCppBackend) GetHost(options any) string {
	if options == nil {
		return ""
	}
	provider, ok := options.(llamaOptionsProvider)
	if !ok {
		return ""
	}
	opts := provider.GetLlamaServerOptions()
	if opts == nil {
		return ""
	}
	return opts.Host
}

// BuildCommandArgs builds command line arguments
func (b *LlamaCppBackend) BuildCommandArgs(options any) []string {
	if options == nil {
		return []string{}
	}
	provider, ok := options.(llamaOptionsProvider)
	if !ok {
		return []string{}
	}
	opts := provider.GetLlamaServerOptions()
	if opts == nil {
		return []string{}
	}
	return opts.BuildCommandArgs()
}

// BuildDockerArgs builds Docker-specific arguments
func (b *LlamaCppBackend) BuildDockerArgs(options any) []string {
	if options == nil {
		return []string{}
	}
	provider, ok := options.(llamaOptionsProvider)
	if !ok {
		return []string{}
	}
	opts := provider.GetLlamaServerOptions()
	if opts == nil {
		return []string{}
	}
	return opts.BuildDockerArgs()
}

// ValidateOptions validates backend-specific options
func (b *LlamaCppBackend) ValidateOptions(options any) error {
	if options == nil {
		return fmt.Errorf("llama.cpp options cannot be nil")
	}
	provider, ok := options.(llamaOptionsProvider)
	if !ok {
		return fmt.Errorf("invalid llama.cpp options type")
	}
	opts := provider.GetLlamaServerOptions()
	if opts == nil {
		return fmt.Errorf("llama.cpp options cannot be nil")
	}

	// Validate port range
	if opts.Port < 0 || opts.Port > 65535 {
		return fmt.Errorf("invalid port range: %d", opts.Port)
	}

	return nil
}

// ParseCommand parses a command string into options
func (b *LlamaCppBackend) ParseCommand(command string) (any, error) {
	return ParseLlamaCommand(command)
}

// SupportsDocker returns true if the backend supports Docker
func (b *LlamaCppBackend) SupportsDocker() bool {
	return true
}

// GetResponseHeaders returns the response headers configuration for the backend
func (b *LlamaCppBackend) GetResponseHeaders(backendConfig *config.BackendSettings) map[string]string {
	if backendConfig == nil || backendConfig.ResponseHeaders == nil {
		return make(map[string]string)
	}
	return backendConfig.ResponseHeaders
}
