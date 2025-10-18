package mlx

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
)

func init() {
	// Register this backend with the default registry
	backends.GetDefaultRegistry().Register(NewMlxBackend())
}

// MlxBackend implements the Backend interface for MLX LM
type MlxBackend struct{}

// NewMlxBackend creates a new MLX backend instance
func NewMlxBackend() *MlxBackend {
	return &MlxBackend{}
}

// GetType returns the backend type
func (b *MlxBackend) GetType() backends.BackendType {
	return backends.BackendTypeMlxLm
}

// GetConfigKey returns the config key for MLX
func (b *MlxBackend) GetConfigKey() string {
	return "mlx"
}

// GetPort extracts the port from backend-specific options
func (b *MlxBackend) GetPort(options any) int {
	opts, ok := options.(*MlxServerOptions)
	if !ok || opts == nil {
		return 0
	}
	return opts.Port
}

// SetPort sets the port in backend-specific options
func (b *MlxBackend) SetPort(options any, port int) {
	opts, ok := options.(*MlxServerOptions)
	if ok && opts != nil {
		opts.Port = port
	}
}

// GetHost extracts the host from backend-specific options
func (b *MlxBackend) GetHost(options any) string {
	opts, ok := options.(*MlxServerOptions)
	if !ok || opts == nil {
		return ""
	}
	return opts.Host
}

// BuildCommandArgs builds command line arguments
func (b *MlxBackend) BuildCommandArgs(options any) []string {
	opts, ok := options.(*MlxServerOptions)
	if !ok || opts == nil {
		return []string{}
	}
	return opts.BuildCommandArgs()
}

// BuildDockerArgs builds Docker-specific arguments
// Note: MLX does not support Docker
func (b *MlxBackend) BuildDockerArgs(options any) []string {
	return []string{}
}

// ValidateOptions validates backend-specific options
func (b *MlxBackend) ValidateOptions(options any) error {
	opts, ok := options.(*MlxServerOptions)
	if !ok {
		return fmt.Errorf("invalid MLX options type")
	}
	if opts == nil {
		return fmt.Errorf("MLX options cannot be nil")
	}

	// Validate port range
	if opts.Port < 0 || opts.Port > 65535 {
		return fmt.Errorf("invalid port range: %d", opts.Port)
	}

	return nil
}

// ParseCommand parses a command string into options
func (b *MlxBackend) ParseCommand(command string) (any, error) {
	return ParseMlxCommand(command)
}

// SupportsDocker returns false as MLX does not support Docker
func (b *MlxBackend) SupportsDocker() bool {
	return false
}

// GetResponseHeaders returns the response headers configuration for the backend
func (b *MlxBackend) GetResponseHeaders(backendConfig *config.BackendSettings) map[string]string {
	if backendConfig == nil || backendConfig.ResponseHeaders == nil {
		return make(map[string]string)
	}
	return backendConfig.ResponseHeaders
}
