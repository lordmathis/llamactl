package config

import "time"

// BackendSettings contains structured backend configuration
type BackendSettings struct {
	Command         string            `yaml:"command" json:"command"`
	Args            []string          `yaml:"args" json:"args"`
	Environment     map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Docker          *DockerSettings   `yaml:"docker,omitempty" json:"docker,omitempty"`
	ResponseHeaders map[string]string `yaml:"response_headers,omitempty" json:"response_headers,omitempty"`
	CacheDir        string            `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty"`
}

// DockerSettings contains Docker-specific configuration
type DockerSettings struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	Image       string            `yaml:"image" json:"image"`
	Args        []string          `yaml:"args" json:"args"`
	Environment map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
}

// BackendConfig contains backend executable configurations
type BackendConfig struct {
	LlamaCpp BackendSettings `yaml:"llama-cpp" json:"llama-cpp"`
	VLLM     BackendSettings `yaml:"vllm" json:"vllm"`
	MLX      BackendSettings `yaml:"mlx" json:"mlx"`
}

// AppConfig represents the configuration for llamactl
type AppConfig struct {
	Server    ServerConfig          `yaml:"server" json:"server"`
	Backends  BackendConfig         `yaml:"backends" json:"backends"`
	Instances InstancesConfig       `yaml:"instances" json:"instances"`
	Database  DatabaseConfig        `yaml:"database" json:"database"`
	Auth      AuthConfig            `yaml:"auth" json:"auth"`
	LocalNode string                `yaml:"local_node,omitempty" json:"local_node,omitempty"`
	Nodes     map[string]NodeConfig `yaml:"nodes,omitempty" json:"nodes,omitempty"`

	// Directory where all llamactl data will be stored (database, instances, logs, etc.)
	DataDir string `yaml:"data_dir" json:"data_dir"`

	Version    string `yaml:"-" json:"version"`
	CommitHash string `yaml:"-" json:"commit_hash"`
	BuildTime  string `yaml:"-" json:"build_time"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	// Server host to bind to
	Host string `yaml:"host" json:"host"`

	// Server port to bind to
	Port int `yaml:"port" json:"port"`

	// Allowed origins for CORS (e.g., "http://localhost:3000")
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`

	// Allowed headers for CORS (e.g., "Accept", "Authorization", "Content-Type", "X-CSRF-Token")
	AllowedHeaders []string `yaml:"allowed_headers" json:"allowed_headers"`

	// Enable Swagger UI for API documentation
	EnableSwagger bool `yaml:"enable_swagger" json:"enable_swagger"`

	// Response headers to send with responses
	ResponseHeaders map[string]string `yaml:"response_headers,omitempty" json:"response_headers,omitempty"`
}

// DatabaseConfig contains database configuration settings
type DatabaseConfig struct {
	// Database file path (relative to the top-level data_dir or absolute)
	Path string `yaml:"path" json:"path"`

	// Connection settings
	MaxOpenConnections int           `yaml:"max_open_connections" json:"max_open_connections"`
	MaxIdleConnections int           `yaml:"max_idle_connections" json:"max_idle_connections"`
	ConnMaxLifetime    time.Duration `yaml:"connection_max_lifetime" json:"connection_max_lifetime" swaggertype:"string" example:"1h"`
}

// InstancesConfig contains instance management configuration
type InstancesConfig struct {
	// Port range for instances (e.g., 8000,9000)
	PortRange [2]int `yaml:"port_range" json:"port_range"`

	// Automatically create the data directory if it doesn't exist
	AutoCreateDirs bool `yaml:"auto_create_dirs" json:"auto_create_dirs"`

	// Maximum number of instances that can be created
	MaxInstances int `yaml:"max_instances" json:"max_instances"`

	// Maximum number of instances that can be running at the same time
	MaxRunningInstances int `yaml:"max_running_instances,omitempty" json:"max_running_instances,omitempty"`

	// Enable LRU eviction for instance logs
	EnableLRUEviction bool `yaml:"enable_lru_eviction" json:"enable_lru_eviction"`

	// Default auto-restart setting for new instances
	DefaultAutoRestart bool `yaml:"default_auto_restart" json:"default_auto_restart"`

	// Default max restarts for new instances
	DefaultMaxRestarts int `yaml:"default_max_restarts" json:"default_max_restarts"`

	// Default restart delay for new instances (in seconds)
	DefaultRestartDelay int `yaml:"default_restart_delay" json:"default_restart_delay"`

	// Default on-demand start setting for new instances
	DefaultOnDemandStart bool `yaml:"default_on_demand_start" json:"default_on_demand_start"`

	// How long to wait for an instance to start on demand (in seconds)
	OnDemandStartTimeout int `yaml:"on_demand_start_timeout,omitempty" json:"on_demand_start_timeout,omitempty"`

	// Interval for checking instance timeouts (in minutes)
	TimeoutCheckInterval int `yaml:"timeout_check_interval" json:"timeout_check_interval"`

	// Logs directory override (relative to data_dir if not absolute)
	LogsDir string `yaml:"logs_dir" json:"logs_dir"`

	// Log rotation enabled
	LogRotationEnabled bool `yaml:"log_rotation_enabled" default:"true"`

	// Maximum log file size in MB before rotation
	LogRotationMaxSize int `yaml:"log_rotation_max_size" default:"100"`

	// Whether to compress rotated log files
	LogRotationCompress bool `yaml:"log_rotation_compress" default:"false"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {

	// Require authentication for OpenAI compatible inference endpoints
	RequireInferenceAuth bool `yaml:"require_inference_auth" json:"require_inference_auth"`

	// Require authentication for management endpoints
	RequireManagementAuth bool `yaml:"require_management_auth" json:"require_management_auth"`

	// List of keys for management endpoints
	ManagementKeys []string `yaml:"management_keys" json:"management_keys"`
}

type NodeConfig struct {
	Address string `yaml:"address" json:"address"`
	APIKey  string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
}
