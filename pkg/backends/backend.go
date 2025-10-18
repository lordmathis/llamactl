package backends

import (
	"fmt"
	"llamactl/pkg/config"
)

type BackendType string

const (
	BackendTypeLlamaCpp BackendType = "llama_cpp"
	BackendTypeMlxLm    BackendType = "mlx_lm"
	BackendTypeVllm     BackendType = "vllm"
	// BackendTypeMlxVlm BackendType = "mlx_vlm"  // Future expansion
)

// BackendOptions is a forward declaration to avoid circular imports
// The actual type is defined in the instance package
type BackendOptions interface {
	GetBackendType() BackendType
	GetLlamaServerOptions() any
	GetMlxServerOptions() any
	GetVllmServerOptions() any
}

// Backend represents a backend implementation
type Backend interface {
	// GetType returns the backend type
	GetType() BackendType

	// GetConfigKey returns the string key used to look up backend settings in config
	GetConfigKey() string

	// GetPort extracts the port from backend-specific options
	GetPort(options BackendOptions) int

	// SetPort sets the port in backend-specific options
	SetPort(options BackendOptions, port int)

	// GetHost extracts the host from backend-specific options
	GetHost(options BackendOptions) string

	// BuildCommandArgs builds command line arguments
	BuildCommandArgs(options BackendOptions) []string

	// BuildDockerArgs builds Docker-specific arguments
	BuildDockerArgs(options BackendOptions) []string

	// ValidateOptions validates backend-specific options
	ValidateOptions(options BackendOptions) error

	// ParseCommand parses a command string into options
	ParseCommand(command string) (any, error)

	// SupportsDocker returns true if the backend supports Docker
	SupportsDocker() bool

	// GetResponseHeaders returns the response headers configuration for the backend
	GetResponseHeaders(backendConfig *config.BackendSettings) map[string]string
}

// BackendRegistry manages available backends
type BackendRegistry struct {
	backends map[BackendType]Backend
}

var defaultRegistry *BackendRegistry

// NewBackendRegistry creates a new registry with all backends
func NewBackendRegistry() *BackendRegistry {
	return &BackendRegistry{
		backends: make(map[BackendType]Backend),
	}
}

// GetDefaultRegistry returns the default backend registry
// The registry is lazily initialized on first call
func GetDefaultRegistry() *BackendRegistry {
	if defaultRegistry == nil {
		defaultRegistry = NewBackendRegistry()
	}
	return defaultRegistry
}

// Register adds a backend to the registry
func (r *BackendRegistry) Register(backend Backend) {
	r.backends[backend.GetType()] = backend
}

// Get retrieves a backend by type
func (r *BackendRegistry) Get(backendType BackendType) (Backend, error) {
	backend, exists := r.backends[backendType]
	if !exists {
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
	return backend, nil
}
