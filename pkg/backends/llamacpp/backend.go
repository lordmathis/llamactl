package llamacpp

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
)

func init() {
	// Register this backend with the default registry
	backends.GetDefaultRegistry().Register(NewLlamaCppBackend())
}

// LlamaCppBackend implements the Backend interface for llama.cpp
type LlamaCppBackend struct{}

// NewLlamaCppBackend creates a new llama.cpp backend instance
func NewLlamaCppBackend() *LlamaCppBackend {
	return &LlamaCppBackend{}
}

// GetType returns the backend type
func (b *LlamaCppBackend) GetType() backends.BackendType {
	return backends.BackendTypeLlamaCpp
}

// GetConfigKey returns the config key for llama.cpp
func (b *LlamaCppBackend) GetConfigKey() string {
	return "llama-cpp"
}

// GetPort extracts the port from backend-specific options
func (b *LlamaCppBackend) GetPort(options backends.BackendOptions) int {
	if options == nil {
		return 0
	}
	opts, ok := options.GetLlamaServerOptions().(*LlamaServerOptions)
	if !ok || opts == nil {
		return 0
	}
	return opts.Port
}

// SetPort sets the port in backend-specific options
func (b *LlamaCppBackend) SetPort(options backends.BackendOptions, port int) {
	if options == nil {
		return
	}
	opts, ok := options.GetLlamaServerOptions().(*LlamaServerOptions)
	if ok && opts != nil {
		opts.Port = port
	}
}

// GetHost extracts the host from backend-specific options
func (b *LlamaCppBackend) GetHost(options backends.BackendOptions) string {
	if options == nil {
		return ""
	}
	opts, ok := options.GetLlamaServerOptions().(*LlamaServerOptions)
	if !ok || opts == nil {
		return ""
	}
	return opts.Host
}

// BuildCommandArgs builds command line arguments
func (b *LlamaCppBackend) BuildCommandArgs(options backends.BackendOptions) []string {
	if options == nil {
		return []string{}
	}
	opts, ok := options.GetLlamaServerOptions().(*LlamaServerOptions)
	if !ok || opts == nil {
		return []string{}
	}
	return opts.BuildCommandArgs()
}

// BuildDockerArgs builds Docker-specific arguments
func (b *LlamaCppBackend) BuildDockerArgs(options backends.BackendOptions) []string {
	if options == nil {
		return []string{}
	}
	opts, ok := options.GetLlamaServerOptions().(*LlamaServerOptions)
	if !ok || opts == nil {
		return []string{}
	}
	return opts.BuildDockerArgs()
}

// ValidateOptions validates backend-specific options
func (b *LlamaCppBackend) ValidateOptions(options backends.BackendOptions) error {
	if options == nil {
		return fmt.Errorf("llama.cpp options cannot be nil")
	}
	opts, ok := options.GetLlamaServerOptions().(*LlamaServerOptions)
	if !ok {
		return fmt.Errorf("invalid llama.cpp options type")
	}
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
