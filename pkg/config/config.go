package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// BackendSettings contains structured backend configuration
type BackendSettings struct {
	Command         string            `yaml:"command"`
	Args            []string          `yaml:"args"`
	Environment     map[string]string `yaml:"environment,omitempty"`
	Docker          *DockerSettings   `yaml:"docker,omitempty"`
	ResponseHeaders map[string]string `yaml:"response_headers,omitempty"`
}

// DockerSettings contains Docker-specific configuration
type DockerSettings struct {
	Enabled     bool              `yaml:"enabled"`
	Image       string            `yaml:"image"`
	Args        []string          `yaml:"args"`
	Environment map[string]string `yaml:"environment,omitempty"`
}

// BackendConfig contains backend executable configurations
type BackendConfig struct {
	LlamaCpp BackendSettings `yaml:"llama-cpp"`
	VLLM     BackendSettings `yaml:"vllm"`
	MLX      BackendSettings `yaml:"mlx"`
}

// AppConfig represents the configuration for llamactl
type AppConfig struct {
	Server     ServerConfig    `yaml:"server"`
	Backends   BackendConfig   `yaml:"backends"`
	Instances  InstancesConfig `yaml:"instances"`
	Auth       AuthConfig      `yaml:"auth"`
	Version    string          `yaml:"-"`
	CommitHash string          `yaml:"-"`
	BuildTime  string          `yaml:"-"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	// Server host to bind to
	Host string `yaml:"host"`

	// Server port to bind to
	Port int `yaml:"port"`

	// Allowed origins for CORS (e.g., "http://localhost:3000")
	AllowedOrigins []string `yaml:"allowed_origins"`

	// Allowed headers for CORS (e.g., "Accept", "Authorization", "Content-Type", "X-CSRF-Token")
	AllowedHeaders []string `yaml:"allowed_headers"`

	// Enable Swagger UI for API documentation
	EnableSwagger bool `yaml:"enable_swagger"`

	// Response headers to send with responses
	ResponseHeaders map[string]string `yaml:"response_headers,omitempty"`
}

// InstancesConfig contains instance management configuration
type InstancesConfig struct {
	// Port range for instances (e.g., 8000,9000)
	PortRange [2]int `yaml:"port_range"`

	// Directory where all llamactl data will be stored (instances.json, logs, etc.)
	DataDir string `yaml:"data_dir"`

	// Instance config directory override
	InstancesDir string `yaml:"configs_dir"`

	// Logs directory override
	LogsDir string `yaml:"logs_dir"`

	// Automatically create the data directory if it doesn't exist
	AutoCreateDirs bool `yaml:"auto_create_dirs"`

	// Maximum number of instances that can be created
	MaxInstances int `yaml:"max_instances"`

	// Maximum number of instances that can be running at the same time
	MaxRunningInstances int `yaml:"max_running_instances,omitempty"`

	// Enable LRU eviction for instance logs
	EnableLRUEviction bool `yaml:"enable_lru_eviction"`

	// Default auto-restart setting for new instances
	DefaultAutoRestart bool `yaml:"default_auto_restart"`

	// Default max restarts for new instances
	DefaultMaxRestarts int `yaml:"default_max_restarts"`

	// Default restart delay for new instances (in seconds)
	DefaultRestartDelay int `yaml:"default_restart_delay"`

	// Default on-demand start setting for new instances
	DefaultOnDemandStart bool `yaml:"default_on_demand_start"`

	// How long to wait for an instance to start on demand (in seconds)
	OnDemandStartTimeout int `yaml:"on_demand_start_timeout,omitempty"`

	// Interval for checking instance timeouts (in minutes)
	TimeoutCheckInterval int `yaml:"timeout_check_interval"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {

	// Require authentication for OpenAI compatible inference endpoints
	RequireInferenceAuth bool `yaml:"require_inference_auth"`

	// List of keys for OpenAI compatible inference endpoints
	InferenceKeys []string `yaml:"inference_keys"`

	// Require authentication for management endpoints
	RequireManagementAuth bool `yaml:"require_management_auth"`

	// List of keys for management endpoints
	ManagementKeys []string `yaml:"management_keys"`
}

// LoadConfig loads configuration with the following precedence:
// 1. Hardcoded defaults
// 2. Config file
// 3. Environment variables
func LoadConfig(configPath string) (AppConfig, error) {
	// 1. Start with defaults
	cfg := AppConfig{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			AllowedOrigins: []string{"*"}, // Default to allow all origins
			AllowedHeaders: []string{"*"}, // Default to allow all headers
			EnableSwagger:  false,
		},
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
						"-v", filepath.Join(getDefaultDataDirectory(), "llama.cpp") + ":/root/.cache/llama.cpp"},
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
						"-v", filepath.Join(getDefaultDataDirectory(), "huggingface") + ":/root/.cache/huggingface",
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
			DataDir:   getDefaultDataDirectory(),
			// NOTE: empty strings are set as placeholder values since InstancesDir and LogsDir
			// should be relative path to DataDir if not explicitly set.
			InstancesDir:         "",
			LogsDir:              "",
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

	// 3. Override with environment variables
	loadEnvVars(&cfg)

	// If InstancesDir or LogsDir is not set, set it to relative path of DataDir
	if cfg.Instances.InstancesDir == "" {
		cfg.Instances.InstancesDir = filepath.Join(cfg.Instances.DataDir, "instances")
	}
	if cfg.Instances.LogsDir == "" {
		cfg.Instances.LogsDir = filepath.Join(cfg.Instances.DataDir, "logs")
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
		cfg.Instances.DataDir = dataDir
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

// GetBackendSettings resolves backend settings
func (bc *BackendConfig) GetBackendSettings(backendType string) BackendSettings {
	switch backendType {
	case "llama-cpp":
		return bc.LlamaCpp
	case "vllm":
		return bc.VLLM
	case "mlx":
		return bc.MLX
	default:
		return BackendSettings{}
	}
}
