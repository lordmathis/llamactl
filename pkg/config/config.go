package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration with the following precedence:
// 1. Hardcoded defaults
// 2. .env file
// 3. Config file (with env var expansion)
// 4. LLAMACTL_* environment variable overrides
func LoadConfig(configPath string) (AppConfig, error) {
	// 1. Start with defaults
	defaultDataDir := getDefaultDataDir()
	cfg := getDefaultConfig(defaultDataDir)

	// 2. Load .env file
	if _, err := loadDotEnv(configPath); err != nil {
		return AppConfig{}, err
	}

	// 3. Load from config file with env var expansion
	data, err := readConfigFile(configPath)
	if err != nil {
		return AppConfig{}, err
	}
	if data != nil {
		data = expandEnvVars(data)
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return AppConfig{}, err
		}
	}

	// If local node is not defined in nodes, add it with default config
	if _, ok := cfg.Nodes[cfg.LocalNode]; !ok {
		cfg.Nodes[cfg.LocalNode] = NodeConfig{}
	}

	// 4. Override with environment variables
	loadEnvVars(&cfg)

	if cfg.Auth.OIDC.Enabled() && !cfg.Auth.RequireManagementAuth {
		log.Printf("Warning: auth.oidc is enabled but require_management_auth is false; " +
			"management endpoints stay unauthenticated and OIDC login only adds session support.")
	}

	// Set default directories if not specified
	if cfg.Instances.LogsDir == "" {
		cfg.Instances.LogsDir = filepath.Join(cfg.DataDir, "logs")
	}
	if cfg.Instances.InstancesDir == "" {
		cfg.Instances.InstancesDir = filepath.Join(cfg.DataDir, "instances")
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

// An explicit path must be readable: any error fails. Fallback locations
// are skipped only when they do not exist.
func readConfigFile(configPath string) ([]byte, error) {
	var configLocations []string

	if configPath != "" {
		configLocations = []string{configPath}
	} else {
		configLocations = getDefaultConfigLocations()
	}

	for _, path := range configLocations {
		data, err := os.ReadFile(path)
		if err == nil {
			log.Printf("Read config at %s", path)
			return data, nil
		}
		if configPath != "" || !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("reading config file %s: %w", path, err)
		}
	}

	return nil, nil
}
