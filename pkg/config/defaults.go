package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"
)

func getDefaultConfig(dataDir string) AppConfig {
	return AppConfig{
		Server: ServerConfig{
			Host:           "0.0.0.0",
			Port:           8080,
			AllowedOrigins: []string{"*"}, // Default to allow all origins
			AllowedHeaders: []string{"*"}, // Default to allow all headers
			EnableSwagger:  false,
		},
		LocalNode: "main",
		Nodes:     map[string]NodeConfig{},
		DataDir:   dataDir,
		Backends: BackendConfig{
			LlamaCpp: BackendSettings{
				Command:     "llama-server",
				Args:        []string{},
				Environment: map[string]string{},
				CacheDir:    getDefaultLlamaCacheDir(),
				Docker: &DockerSettings{
					Enabled: false,
					Image:   "ghcr.io/ggml-org/llama.cpp:server",
					Args: []string{
						"run", "--rm", "--network", "host", "--gpus", "all",
						"-v", getDefaultLlamaCacheDir() + ":/root/.cache/llama.cpp"},
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
						"-v", filepath.Join(dataDir, "huggingface") + ":/root/.cache/huggingface",
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
			PortRange:            [2]int{8000, 9000},
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
			LogRotationMaxSize:   100,
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
			RequireManagementAuth: true,
			ManagementKeys:        []string{},
		},
	}
}

// getDefaultLlamaCacheDir returns the default platform specific cache directory for llama.cpp models
func getDefaultLlamaCacheDir() string {
	homeDir, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "llama.cpp", "cache")
		}
		return filepath.Join("AppData", "Local", "llama.cpp", "cache")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Caches", "llama.cpp")
	default:
		if xdgCacheHome := os.Getenv("XDG_CACHE_HOME"); xdgCacheHome != "" {
			return filepath.Join(xdgCacheHome, "llama.cpp")
		}
		return filepath.Join(homeDir, ".cache", "llama.cpp")
	}
}

// getDefaultDataDir returns platform-specific default data directory
func getDefaultDataDir() string {
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
