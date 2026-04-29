package config

import (
	"encoding/json"
	"fmt"
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
	loadDotEnv(configPath)

	// 3. Load from config file with env var expansion
	data, err := readConfigFile(configPath)
	if err != nil {
		return cfg, err
	}
	if data != nil {
		data = expandEnvVars(data)
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	}

	// If local node is not defined in nodes, add it with default config
	if _, ok := cfg.Nodes[cfg.LocalNode]; !ok {
		cfg.Nodes[cfg.LocalNode] = NodeConfig{}
	}

	// 4. Override with environment variables
	loadEnvVars(&cfg)

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

// readConfigFile attempts to read config from file with fallback locations.
// Returns nil data if no config file is found (not an error).
func readConfigFile(configPath string) ([]byte, error) {
	var configLocations []string

	if configPath != "" {
		configLocations = []string{configPath}
	} else {
		configLocations = getDefaultConfigLocations()
	}

	for _, path := range configLocations {
		if data, err := os.ReadFile(path); err == nil {
			log.Printf("Read config at %s", path)
			return data, nil
		}
	}

	return nil, nil
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
	sanitized.Auth.ManagementKeys = []string{}

	// Clear API keys from nodes
	for nodeName, node := range sanitized.Nodes {
		node.APIKey = ""
		sanitized.Nodes[nodeName] = node
	}

	return sanitized, nil
}
