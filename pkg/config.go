package llamactl

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the configuration for llamactl
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Instances InstancesConfig `yaml:"instances"`
	Auth      AuthConfig      `yaml:"auth"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	// Server host to bind to
	Host string `yaml:"host"`

	// Server port to bind to
	Port int `yaml:"port"`

	// Allowed origins for CORS (e.g., "http://localhost:3000")
	AllowedOrigins []string `yaml:"allowed_origins"`

	// Enable Swagger UI for API documentation
	EnableSwagger bool `yaml:"enable_swagger"`
}

// InstancesConfig contains instance management configuration
type InstancesConfig struct {
	// Port range for instances (e.g., 8000,9000)
	PortRange [2]int `yaml:"port_range"`

	// Directory where all llamactl data will be stored (instances.json, logs, etc.)
	DataDir string `yaml:"data_dir"`

	// Instance config directory override
	ConfigDir string `yaml:"config_dir"`

	// Logs directory override
	LogDir string `yaml:"logs_dir"`

	// Automatically create the data directory if it doesn't exist
	AutoCreateDirs bool `yaml:"auto_create_dirs"`

	// Maximum number of instances that can be created
	MaxInstances int `yaml:"max_instances"`

	// Path to llama-server executable
	LlamaExecutable string `yaml:"llama_executable"`

	// Default auto-restart setting for new instances
	DefaultAutoRestart bool `yaml:"default_auto_restart"`

	// Default max restarts for new instances
	DefaultMaxRestarts int `yaml:"default_max_restarts"`

	// Default restart delay for new instances (in seconds)
	DefaultRestartDelay int `yaml:"default_restart_delay"`
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
func LoadConfig(configPath string) (Config, error) {
	// 1. Start with defaults
	cfg := Config{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			AllowedOrigins: []string{"*"}, // Default to allow all origins
			EnableSwagger:  false,
		},
		Instances: InstancesConfig{
			PortRange:           [2]int{8000, 9000},
			DataDir:             getDefaultDataDirectory(),
			ConfigDir:           filepath.Join(getDefaultDataDirectory(), "instances"),
			LogDir:              filepath.Join(getDefaultDataDirectory(), "logs"),
			AutoCreateDirs:      true,
			MaxInstances:        -1, // -1 means unlimited
			LlamaExecutable:     "llama-server",
			DefaultAutoRestart:  true,
			DefaultMaxRestarts:  3,
			DefaultRestartDelay: 5,
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

	return cfg, nil
}

// loadConfigFile attempts to load config from file with fallback locations
func loadConfigFile(cfg *Config, configPath string) error {
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
			return nil
		}
	}

	return nil
}

// loadEnvVars overrides config with environment variables
func loadEnvVars(cfg *Config) {
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
	if instancesDir := os.Getenv("LLAMACTL_INSTANCES_DIRECTORY"); instancesDir != "" {
		cfg.Instances.ConfigDir = instancesDir
	}
	if logsDir := os.Getenv("LLAMACTL_LOGS_DIRECTORY"); logsDir != "" {
		cfg.Instances.LogDir = logsDir
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
	if llamaExec := os.Getenv("LLAMACTL_LLAMA_EXECUTABLE"); llamaExec != "" {
		cfg.Instances.LlamaExecutable = llamaExec
	}
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
		// System-wide would be /usr/local/var/llamactl but requires sudo
		homeDir, _ := os.UserHomeDir()
		if homeDir != "" {
			return filepath.Join(homeDir, "Library", "Application Support", "llamactl")
		}
		return "/usr/local/var/llamactl" // Fallback

	default:
		// Linux and other Unix-like systems
		// Try system directory first, fallback to user directory
		if os.Geteuid() == 0 { // Running as root
			return "/var/lib/llamactl"
		}
		// For non-root users, use XDG data home
		if xdgDataHome := os.Getenv("XDG_DATA_HOME"); xdgDataHome != "" {
			return filepath.Join(xdgDataHome, "llamactl")
		}
		if homeDir, _ := os.UserHomeDir(); homeDir != "" {
			return filepath.Join(homeDir, ".local", "share", "llamactl")
		}
		return "/var/lib/llamactl" // Final fallback
	}
}

// getDefaultConfigLocations returns platform-specific config file locations
func getDefaultConfigLocations() []string {
	var locations []string

	// Current directory (cross-platform)
	locations = append(locations,
		"./llamactl.yaml",
		"./config.yaml",
	)

	homeDir, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		// Windows: Use APPDATA and ProgramData
		if appData := os.Getenv("APPDATA"); appData != "" {
			locations = append(locations, filepath.Join(appData, "llamactl", "config.yaml"))
		}
		if programData := os.Getenv("PROGRAMDATA"); programData != "" {
			locations = append(locations, filepath.Join(programData, "llamactl", "config.yaml"))
		}
		// Fallback to user home
		if homeDir != "" {
			locations = append(locations, filepath.Join(homeDir, "llamactl", "config.yaml"))
		}

	case "darwin":
		// macOS: Use proper Application Support directories
		if homeDir != "" {
			locations = append(locations,
				filepath.Join(homeDir, "Library", "Application Support", "llamactl", "config.yaml"),
				filepath.Join(homeDir, ".config", "llamactl", "config.yaml"), // XDG fallback
			)
		}
		locations = append(locations, "/Library/Application Support/llamactl/config.yaml")
		locations = append(locations, "/etc/llamactl/config.yaml") // Unix fallback

	default:
		// User config: $XDG_CONFIG_HOME/llamactl/config.yaml or ~/.config/llamactl/config.yaml
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" && homeDir != "" {
			configHome = filepath.Join(homeDir, ".config")
		}
		if configHome != "" {
			locations = append(locations, filepath.Join(configHome, "llamactl", "config.yaml"))
		}

		// System config: /etc/llamactl/config.yaml
		locations = append(locations, "/etc/llamactl/config.yaml")

		// Additional system locations
		if xdgConfigDirs := os.Getenv("XDG_CONFIG_DIRS"); xdgConfigDirs != "" {
			for dir := range strings.SplitSeq(xdgConfigDirs, ":") {
				if dir != "" {
					locations = append(locations, filepath.Join(dir, "llamactl", "config.yaml"))
				}
			}
		}
	}

	return locations
}
