package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// BackendSettings contains structured backend configuration
type BackendSettings struct {
	Command         string            `yaml:"command" json:"command"`
	Args            []string          `yaml:"args" json:"args"`
	Environment     map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Docker          *DockerSettings   `yaml:"docker,omitempty" json:"docker,omitempty"`
	ResponseHeaders map[string]string `yaml:"response_headers,omitempty" json:"response_headers,omitempty"`
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

// LogRotationConfig contains log rotation settings for instances
type LogRotationConfig struct {
	Enabled   bool `yaml:"enabled" default:"true"`
	MaxSizeMB int  `yaml:"max_size_mb" default:"100"` // MB
	Compress  bool `yaml:"compress" default:"false"`
}

// InstancesConfig contains instance management configuration
type InstancesConfig struct {
	// Port range for instances (e.g., 8000,9000)
	PortRange [2]int `yaml:"port_range" json:"port_range"`

	// Instance config directory override (relative to data_dir if not absolute)
	InstancesDir string `yaml:"configs_dir" json:"configs_dir"`

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
	LogRotationMaxSizeMB int `yaml:"log_rotation_max_size_mb" default:"100"`

	// Whether to compress rotated log files
	LogRotationCompress bool `yaml:"log_rotation_compress" default:"false"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {

	// Require authentication for OpenAI compatible inference endpoints
	RequireInferenceAuth bool `yaml:"require_inference_auth" json:"require_inference_auth"`

	// List of keys for OpenAI compatible inference endpoints
	InferenceKeys []string `yaml:"inference_keys" json:"inference_keys"`

	// Require authentication for management endpoints
	RequireManagementAuth bool `yaml:"require_management_auth" json:"require_management_auth"`

	// List of keys for management endpoints
	ManagementKeys []string `yaml:"management_keys" json:"management_keys"`
}

type NodeConfig struct {
	Address string `yaml:"address" json:"address"`
	APIKey  string `yaml:"api_key,omitempty" json:"api_key,omitempty"`
}

// LoadConfig loads configuration with the following precedence:
// 1. Hardcoded defaults
// 2. Config file
// 3. Environment variables
func LoadConfig(configPath string) (AppConfig, error) {
	// 1. Start with defaults
	defaultDataDir := getDefaultDataDirectory()

	cfg := AppConfig{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			AllowedOrigins: []string{"*"}, // Default to allow all origins
			AllowedHeaders: []string{"*"}, // Default to allow all headers
			EnableSwagger:  false,
		},
		LocalNode: "main",
		Nodes:     map[string]NodeConfig{},
		DataDir:   defaultDataDir,
		Backends: BackendConfig{
			LlamaCpp: BackendSettings{
				Command:     "llama-server",
				Args:        []string{},
				Environment: map[string]string{},
				Docker: &DockerSettings{
					Enabled: false,
					Image:   "ghcr.io/ggml-org/llama.cpp:server",
					Args: []string{
						"run", "--rm", "--network", "host", "--gpus", "all",
						"-v", filepath.Join(defaultDataDir, "llama.cpp") + ":/root/.cache/llama.cpp"},
					Environment: map[string]string{},
				},
			},
			VLLM: BackendSettings{
				Command: "vllm",
				Args:    []string{"serve"},
				Docker: &DockerSettings{
					Enabled: false,
					Image:   "vllm/vllm-openai:latest",
					Args: []string{
						"run", "--rm", "--network", "host", "--gpus", "all", "--shm-size", "1g",
						"-v", filepath.Join(defaultDataDir, "huggingface") + ":/root/.cache/huggingface",
					},
					Environment: map[string]string{},
				},
			},
			MLX: BackendSettings{
				Command: "mlx_lm.server",
				Args:    []string{},
				// No Docker section for MLX - not supported
			},
		},
		Instances: InstancesConfig{
			PortRange: [2]int{8000, 9000},
			// NOTE: empty string is set as placeholder value since InstancesDir
			// should be relative path to DataDir if not explicitly set.
			InstancesDir:         "",
			AutoCreateDirs:       true,
			MaxInstances:         -1, // -1 means unlimited
			MaxRunningInstances:  -1, // -1 means unlimited
			EnableLRUEviction:    true,
			DefaultAutoRestart:   true,
			DefaultMaxRestarts:   3,
			DefaultRestartDelay:  5,
			DefaultOnDemandStart: true,
			OnDemandStartTimeout: 120, // 2 minutes
			TimeoutCheckInterval: 5,   // Check timeouts every 5 minutes
			LogsDir:              "",  // Will be set to data_dir/logs if empty
			LogRotationEnabled:   true,
			LogRotationMaxSizeMB: 100,
			LogRotationCompress:  false,
		},
		Database: DatabaseConfig{
			Path:               "", // Will be set to data_dir/llamactl.db if empty
			MaxOpenConnections: 25,
			MaxIdleConnections: 5,
			ConnMaxLifetime:    5 * time.Minute,
		},
		Auth: AuthConfig{
			RequireInferenceAuth:  true,
			InferenceKeys:         []string{},
			RequireManagementAuth: true,
			ManagementKeys:        []string{},
		},
	}

	// 2. Load from config file
	if err := loadConfigFile(&cfg, configPath); err != nil {
		return cfg, err
	}

	// If local node is not defined in nodes, add it with default config
	if _, ok := cfg.Nodes[cfg.LocalNode]; !ok {
		cfg.Nodes[cfg.LocalNode] = NodeConfig{}
	}

	// 3. Override with environment variables
	loadEnvVars(&cfg)

	// Log warning if deprecated inference keys are present
	if len(cfg.Auth.InferenceKeys) > 0 {
		log.Println("⚠️ Config-based inference keys are no longer supported and will be ignored.")
		log.Println("    Please create inference keys in web UI or via management API.")
	}

	// Set default directories if not specified
	if cfg.Instances.InstancesDir == "" {
		cfg.Instances.InstancesDir = filepath.Join(cfg.DataDir, "instances")
	} else {
		// Log deprecation warning if using custom instances dir
		log.Println("⚠️ Instances directory is deprecated and will be removed in future versions. Instances are persisted in the database.")
	}
	if cfg.Instances.LogsDir == "" {
		cfg.Instances.LogsDir = filepath.Join(cfg.DataDir, "logs")
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = filepath.Join(cfg.DataDir, "llamactl.db")
	}

	// Validate port range
	if cfg.Instances.PortRange[0] <= 0 || cfg.Instances.PortRange[1] <= 0 || cfg.Instances.PortRange[0] >= cfg.Instances.PortRange[1] {
		return AppConfig{}, fmt.Errorf("invalid port range: %v", cfg.Instances.PortRange)
	}

	return cfg, nil
}

// loadConfigFile attempts to load config from file with fallback locations
func loadConfigFile(cfg *AppConfig, configPath string) error {
	var configLocations []string

	// If specific config path provided, use only that
	if configPath != "" {
		configLocations = []string{configPath}
	} else {
		// Default config file locations (in order of precedence)
		configLocations = getDefaultConfigLocations()
	}

	for _, path := range configLocations {
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return err
			}
			log.Printf("Read config at %s", path)
			return nil
		}
	}

	return nil
}

// loadEnvVars overrides config with environment variables
func loadEnvVars(cfg *AppConfig) {
	// Server config
	if host := os.Getenv("LLAMACTL_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if port := os.Getenv("LLAMACTL_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if allowedOrigins := os.Getenv("LLAMACTL_ALLOWED_ORIGINS"); allowedOrigins != "" {
		cfg.Server.AllowedOrigins = strings.Split(allowedOrigins, ",")
	}
	if enableSwagger := os.Getenv("LLAMACTL_ENABLE_SWAGGER"); enableSwagger != "" {
		if b, err := strconv.ParseBool(enableSwagger); err == nil {
			cfg.Server.EnableSwagger = b
		}
	}

	// Data config
	if dataDir := os.Getenv("LLAMACTL_DATA_DIRECTORY"); dataDir != "" {
		cfg.DataDir = dataDir
	}
	if instancesDir := os.Getenv("LLAMACTL_INSTANCES_DIR"); instancesDir != "" {
		cfg.Instances.InstancesDir = instancesDir
	}
	if logsDir := os.Getenv("LLAMACTL_LOGS_DIR"); logsDir != "" {
		cfg.Instances.LogsDir = logsDir
	}
	if autoCreate := os.Getenv("LLAMACTL_AUTO_CREATE_DATA_DIR"); autoCreate != "" {
		if b, err := strconv.ParseBool(autoCreate); err == nil {
			cfg.Instances.AutoCreateDirs = b
		}
	}

	// Instance config
	if portRange := os.Getenv("LLAMACTL_INSTANCE_PORT_RANGE"); portRange != "" {
		if ports := ParsePortRange(portRange); ports != [2]int{0, 0} {
			cfg.Instances.PortRange = ports
		}
	}
	if maxInstances := os.Getenv("LLAMACTL_MAX_INSTANCES"); maxInstances != "" {
		if m, err := strconv.Atoi(maxInstances); err == nil {
			cfg.Instances.MaxInstances = m
		}
	}
	if maxRunning := os.Getenv("LLAMACTL_MAX_RUNNING_INSTANCES"); maxRunning != "" {
		if m, err := strconv.Atoi(maxRunning); err == nil {
			cfg.Instances.MaxRunningInstances = m
		}
	}
	if enableLRUEviction := os.Getenv("LLAMACTL_ENABLE_LRU_EVICTION"); enableLRUEviction != "" {
		if b, err := strconv.ParseBool(enableLRUEviction); err == nil {
			cfg.Instances.EnableLRUEviction = b
		}
	}
	// Backend config
	// LlamaCpp backend
	if llamaCmd := os.Getenv("LLAMACTL_LLAMACPP_COMMAND"); llamaCmd != "" {
		cfg.Backends.LlamaCpp.Command = llamaCmd
	}
	if llamaArgs := os.Getenv("LLAMACTL_LLAMACPP_ARGS"); llamaArgs != "" {
		cfg.Backends.LlamaCpp.Args = strings.Split(llamaArgs, " ")
	}
	if llamaEnv := os.Getenv("LLAMACTL_LLAMACPP_ENV"); llamaEnv != "" {
		if cfg.Backends.LlamaCpp.Environment == nil {
			cfg.Backends.LlamaCpp.Environment = make(map[string]string)
		}
		parseEnvVars(llamaEnv, cfg.Backends.LlamaCpp.Environment)
	}
	if llamaDockerEnabled := os.Getenv("LLAMACTL_LLAMACPP_DOCKER_ENABLED"); llamaDockerEnabled != "" {
		if b, err := strconv.ParseBool(llamaDockerEnabled); err == nil {
			if cfg.Backends.LlamaCpp.Docker == nil {
				cfg.Backends.LlamaCpp.Docker = &DockerSettings{}
			}
			cfg.Backends.LlamaCpp.Docker.Enabled = b
		}
	}
	if llamaDockerImage := os.Getenv("LLAMACTL_LLAMACPP_DOCKER_IMAGE"); llamaDockerImage != "" {
		if cfg.Backends.LlamaCpp.Docker == nil {
			cfg.Backends.LlamaCpp.Docker = &DockerSettings{}
		}
		cfg.Backends.LlamaCpp.Docker.Image = llamaDockerImage
	}
	if llamaDockerArgs := os.Getenv("LLAMACTL_LLAMACPP_DOCKER_ARGS"); llamaDockerArgs != "" {
		if cfg.Backends.LlamaCpp.Docker == nil {
			cfg.Backends.LlamaCpp.Docker = &DockerSettings{}
		}
		cfg.Backends.LlamaCpp.Docker.Args = strings.Split(llamaDockerArgs, " ")
	}
	if llamaDockerEnv := os.Getenv("LLAMACTL_LLAMACPP_DOCKER_ENV"); llamaDockerEnv != "" {
		if cfg.Backends.LlamaCpp.Docker == nil {
			cfg.Backends.LlamaCpp.Docker = &DockerSettings{}
		}
		if cfg.Backends.LlamaCpp.Docker.Environment == nil {
			cfg.Backends.LlamaCpp.Docker.Environment = make(map[string]string)
		}
		parseEnvVars(llamaDockerEnv, cfg.Backends.LlamaCpp.Docker.Environment)
	}
	if llamaEnv := os.Getenv("LLAMACTL_LLAMACPP_RESPONSE_HEADERS"); llamaEnv != "" {
		if cfg.Backends.LlamaCpp.ResponseHeaders == nil {
			cfg.Backends.LlamaCpp.ResponseHeaders = make(map[string]string)
		}
		parseHeaders(llamaEnv, cfg.Backends.LlamaCpp.ResponseHeaders)
	}

	// vLLM backend
	if vllmCmd := os.Getenv("LLAMACTL_VLLM_COMMAND"); vllmCmd != "" {
		cfg.Backends.VLLM.Command = vllmCmd
	}
	if vllmArgs := os.Getenv("LLAMACTL_VLLM_ARGS"); vllmArgs != "" {
		cfg.Backends.VLLM.Args = strings.Split(vllmArgs, " ")
	}
	if vllmEnv := os.Getenv("LLAMACTL_VLLM_ENV"); vllmEnv != "" {
		if cfg.Backends.VLLM.Environment == nil {
			cfg.Backends.VLLM.Environment = make(map[string]string)
		}
		parseEnvVars(vllmEnv, cfg.Backends.VLLM.Environment)
	}
	if vllmDockerEnabled := os.Getenv("LLAMACTL_VLLM_DOCKER_ENABLED"); vllmDockerEnabled != "" {
		if b, err := strconv.ParseBool(vllmDockerEnabled); err == nil {
			if cfg.Backends.VLLM.Docker == nil {
				cfg.Backends.VLLM.Docker = &DockerSettings{}
			}
			cfg.Backends.VLLM.Docker.Enabled = b
		}
	}
	if vllmDockerImage := os.Getenv("LLAMACTL_VLLM_DOCKER_IMAGE"); vllmDockerImage != "" {
		if cfg.Backends.VLLM.Docker == nil {
			cfg.Backends.VLLM.Docker = &DockerSettings{}
		}
		cfg.Backends.VLLM.Docker.Image = vllmDockerImage
	}
	if vllmDockerArgs := os.Getenv("LLAMACTL_VLLM_DOCKER_ARGS"); vllmDockerArgs != "" {
		if cfg.Backends.VLLM.Docker == nil {
			cfg.Backends.VLLM.Docker = &DockerSettings{}
		}
		cfg.Backends.VLLM.Docker.Args = strings.Split(vllmDockerArgs, " ")
	}
	if vllmDockerEnv := os.Getenv("LLAMACTL_VLLM_DOCKER_ENV"); vllmDockerEnv != "" {
		if cfg.Backends.VLLM.Docker == nil {
			cfg.Backends.VLLM.Docker = &DockerSettings{}
		}
		if cfg.Backends.VLLM.Docker.Environment == nil {
			cfg.Backends.VLLM.Docker.Environment = make(map[string]string)
		}
		parseEnvVars(vllmDockerEnv, cfg.Backends.VLLM.Docker.Environment)
	}
	if llamaEnv := os.Getenv("LLAMACTL_VLLM_RESPONSE_HEADERS"); llamaEnv != "" {
		if cfg.Backends.VLLM.ResponseHeaders == nil {
			cfg.Backends.VLLM.ResponseHeaders = make(map[string]string)
		}
		parseHeaders(llamaEnv, cfg.Backends.VLLM.ResponseHeaders)
	}

	// MLX backend
	if mlxCmd := os.Getenv("LLAMACTL_MLX_COMMAND"); mlxCmd != "" {
		cfg.Backends.MLX.Command = mlxCmd
	}
	if mlxArgs := os.Getenv("LLAMACTL_MLX_ARGS"); mlxArgs != "" {
		cfg.Backends.MLX.Args = strings.Split(mlxArgs, " ")
	}
	if mlxEnv := os.Getenv("LLAMACTL_MLX_ENV"); mlxEnv != "" {
		if cfg.Backends.MLX.Environment == nil {
			cfg.Backends.MLX.Environment = make(map[string]string)
		}
		parseEnvVars(mlxEnv, cfg.Backends.MLX.Environment)
	}
	if llamaEnv := os.Getenv("LLAMACTL_MLX_RESPONSE_HEADERS"); llamaEnv != "" {
		if cfg.Backends.MLX.ResponseHeaders == nil {
			cfg.Backends.MLX.ResponseHeaders = make(map[string]string)
		}
		parseHeaders(llamaEnv, cfg.Backends.MLX.ResponseHeaders)
	}

	// Instance defaults
	if autoRestart := os.Getenv("LLAMACTL_DEFAULT_AUTO_RESTART"); autoRestart != "" {
		if b, err := strconv.ParseBool(autoRestart); err == nil {
			cfg.Instances.DefaultAutoRestart = b
		}
	}
	if maxRestarts := os.Getenv("LLAMACTL_DEFAULT_MAX_RESTARTS"); maxRestarts != "" {
		if m, err := strconv.Atoi(maxRestarts); err == nil {
			cfg.Instances.DefaultMaxRestarts = m
		}
	}
	if restartDelay := os.Getenv("LLAMACTL_DEFAULT_RESTART_DELAY"); restartDelay != "" {
		if seconds, err := strconv.Atoi(restartDelay); err == nil {
			cfg.Instances.DefaultRestartDelay = seconds
		}
	}
	if onDemandStart := os.Getenv("LLAMACTL_DEFAULT_ON_DEMAND_START"); onDemandStart != "" {
		if b, err := strconv.ParseBool(onDemandStart); err == nil {
			cfg.Instances.DefaultOnDemandStart = b
		}
	}
	if onDemandTimeout := os.Getenv("LLAMACTL_ON_DEMAND_START_TIMEOUT"); onDemandTimeout != "" {
		if seconds, err := strconv.Atoi(onDemandTimeout); err == nil {
			cfg.Instances.OnDemandStartTimeout = seconds
		}
	}
	if timeoutCheckInterval := os.Getenv("LLAMACTL_TIMEOUT_CHECK_INTERVAL"); timeoutCheckInterval != "" {
		if minutes, err := strconv.Atoi(timeoutCheckInterval); err == nil {
			cfg.Instances.TimeoutCheckInterval = minutes
		}
	}
	// Auth config
	if requireInferenceAuth := os.Getenv("LLAMACTL_REQUIRE_INFERENCE_AUTH"); requireInferenceAuth != "" {
		if b, err := strconv.ParseBool(requireInferenceAuth); err == nil {
			cfg.Auth.RequireInferenceAuth = b
		}
	}
	if inferenceKeys := os.Getenv("LLAMACTL_INFERENCE_KEYS"); inferenceKeys != "" {
		cfg.Auth.InferenceKeys = strings.Split(inferenceKeys, ",")
	}
	if requireManagementAuth := os.Getenv("LLAMACTL_REQUIRE_MANAGEMENT_AUTH"); requireManagementAuth != "" {
		if b, err := strconv.ParseBool(requireManagementAuth); err == nil {
			cfg.Auth.RequireManagementAuth = b
		}
	}
	if managementKeys := os.Getenv("LLAMACTL_MANAGEMENT_KEYS"); managementKeys != "" {
		cfg.Auth.ManagementKeys = strings.Split(managementKeys, ",")
	}

	// Local node config
	if localNode := os.Getenv("LLAMACTL_LOCAL_NODE"); localNode != "" {
		cfg.LocalNode = localNode
	}

	// Database config
	if dbPath := os.Getenv("LLAMACTL_DATABASE_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if maxOpenConns := os.Getenv("LLAMACTL_DATABASE_MAX_OPEN_CONNECTIONS"); maxOpenConns != "" {
		if m, err := strconv.Atoi(maxOpenConns); err == nil {
			cfg.Database.MaxOpenConnections = m
		}
	}
	if maxIdleConns := os.Getenv("LLAMACTL_DATABASE_MAX_IDLE_CONNECTIONS"); maxIdleConns != "" {
		if m, err := strconv.Atoi(maxIdleConns); err == nil {
			cfg.Database.MaxIdleConnections = m
		}
	}
	if connMaxLifetime := os.Getenv("LLAMACTL_DATABASE_CONN_MAX_LIFETIME"); connMaxLifetime != "" {
		if d, err := time.ParseDuration(connMaxLifetime); err == nil {
			cfg.Database.ConnMaxLifetime = d
		}
	}

	// Log rotation config
	if logRotationEnabled := os.Getenv("LLAMACTL_LOG_ROTATION_ENABLED"); logRotationEnabled != "" {
		if b, err := strconv.ParseBool(logRotationEnabled); err == nil {
			cfg.Instances.LogRotationEnabled = b
		}
	}
	if logRotationMaxSizeMB := os.Getenv("LLAMACTL_LOG_ROTATION_MAX_SIZE_MB"); logRotationMaxSizeMB != "" {
		if m, err := strconv.Atoi(logRotationMaxSizeMB); err == nil {
			cfg.Instances.LogRotationMaxSizeMB = m
		}
	}
	if logRotationCompress := os.Getenv("LLAMACTL_LOG_ROTATION_COMPRESS"); logRotationCompress != "" {
		if b, err := strconv.ParseBool(logRotationCompress); err == nil {
			cfg.Instances.LogRotationCompress = b
		}
	}
}

// ParsePortRange parses port range from string formats like "8000-9000" or "8000,9000"
func ParsePortRange(s string) [2]int {
	var parts []string

	// Try both separators
	if strings.Contains(s, "-") {
		parts = strings.Split(s, "-")
	} else if strings.Contains(s, ",") {
		parts = strings.Split(s, ",")
	}

	// Parse the two parts
	if len(parts) == 2 {
		start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err1 == nil && err2 == nil {
			return [2]int{start, end}
		}
	}

	return [2]int{0, 0} // Invalid format
}

// parseEnvVars parses environment variables in format "KEY1=value1,KEY2=value2"
// and populates the provided environment map
func parseEnvVars(envString string, envMap map[string]string) {
	if envString == "" {
		return
	}
	for _, envPair := range strings.Split(envString, ",") {
		if parts := strings.SplitN(strings.TrimSpace(envPair), "=", 2); len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
}

// parseHeaders parses HTTP headers in format "KEY1=value1;KEY2=value2"
// and populates the provided environment map
func parseHeaders(envString string, envMap map[string]string) {
	if envString == "" {
		return
	}
	for _, envPair := range strings.Split(envString, ";") {
		if parts := strings.SplitN(strings.TrimSpace(envPair), "=", 2); len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
}

// getDefaultDataDirectory returns platform-specific default data directory
func getDefaultDataDirectory() string {
	switch runtime.GOOS {
	case "windows":
		// Try PROGRAMDATA first (system-wide), fallback to LOCALAPPDATA (user)
		if programData := os.Getenv("PROGRAMDATA"); programData != "" {
			return filepath.Join(programData, "llamactl")
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "llamactl")
		}
		return "C:\\ProgramData\\llamactl" // Final fallback

	case "darwin":
		// For macOS, use user's Application Support directory
		if homeDir, _ := os.UserHomeDir(); homeDir != "" {
			return filepath.Join(homeDir, "Library", "Application Support", "llamactl")
		}
		return "/usr/local/var/llamactl" // Fallback

	default:
		// Linux and other Unix-like systems
		if homeDir, _ := os.UserHomeDir(); homeDir != "" {
			return filepath.Join(homeDir, ".local", "share", "llamactl")
		}
		return "/var/lib/llamactl" // Final fallback
	}
}

// getDefaultConfigLocations returns platform-specific config file locations
func getDefaultConfigLocations() []string {
	var locations []string
	// Use ./llamactl.yaml and ./config.yaml as the default config file
	locations = append(locations, "llamactl.yaml")
	locations = append(locations, "config.yaml")

	homeDir, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		// Windows: Use APPDATA if available, else user home, fallback to ProgramData
		if appData := os.Getenv("APPDATA"); appData != "" {
			locations = append(locations, filepath.Join(appData, "llamactl", "config.yaml"))
		} else if homeDir != "" {
			locations = append(locations, filepath.Join(homeDir, "llamactl", "config.yaml"))
		}
		locations = append(locations, filepath.Join(os.Getenv("PROGRAMDATA"), "llamactl", "config.yaml"))

	case "darwin":
		// macOS: Use Application Support in user home, fallback to /Library/Application Support
		if homeDir != "" {
			locations = append(locations, filepath.Join(homeDir, "Library", "Application Support", "llamactl", "config.yaml"))
		}
		locations = append(locations, "/Library/Application Support/llamactl/config.yaml")

	default:
		// Linux/Unix: Use ~/.config/llamactl/config.yaml, fallback to /etc/llamactl/config.yaml
		if homeDir != "" {
			locations = append(locations, filepath.Join(homeDir, ".config", "llamactl", "config.yaml"))
		}
		locations = append(locations, "/etc/llamactl/config.yaml")
	}

	return locations
}

// SanitizedCopy returns a copy of the AppConfig with sensitive information removed
func (cfg *AppConfig) SanitizedCopy() (AppConfig, error) {
	// Deep copy via JSON marshal/unmarshal to avoid concurrent map access
	data, err := json.Marshal(cfg)
	if err != nil {
		log.Printf("Failed to marshal config for sanitization: %v", err)
		return AppConfig{}, err
	}

	var sanitized AppConfig
	if err := json.Unmarshal(data, &sanitized); err != nil {
		log.Printf("Failed to unmarshal config for sanitization: %v", err)
		return AppConfig{}, err
	}

	// Clear sensitive information
	sanitized.Auth.InferenceKeys = []string{}
	sanitized.Auth.ManagementKeys = []string{}

	// Clear API keys from nodes
	for nodeName, node := range sanitized.Nodes {
		node.APIKey = ""
		sanitized.Nodes[nodeName] = node
	}

	return sanitized, nil
}
