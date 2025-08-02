package llamactl_test

import (
	"os"
	"path/filepath"
	"testing"

	llamactl "llamactl/pkg"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Test loading config when no file exists and no env vars set
	cfg, err := llamactl.LoadConfig("nonexistent-file.yaml")
	if err != nil {
		t.Fatalf("LoadConfig should not error with defaults: %v", err)
	}

	// Verify default values
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected default host to be 0.0.0.0, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected default port to be 8080, got %d", cfg.Server.Port)
	}
	if cfg.Data.Directory != "/var/lib/llamactl" {
		t.Errorf("Expected default data directory '/var/lib/llamactl', got %q", cfg.Data.Directory)
	}
	if !cfg.Data.AutoCreate {
		t.Error("Expected default data auto-create to be true")
	}
	if cfg.Instances.PortRange != [2]int{8000, 9000} {
		t.Errorf("Expected default port range [8000, 9000], got %v", cfg.Instances.PortRange)
	}
	if cfg.Instances.LogDirectory != "/tmp/llamactl" {
		t.Errorf("Expected default log directory '/tmp/llamactl', got %q", cfg.Instances.LogDirectory)
	}
	if cfg.Instances.MaxInstances != -1 {
		t.Errorf("Expected default max instances -1, got %d", cfg.Instances.MaxInstances)
	}
	if cfg.Instances.LlamaExecutable != "llama-server" {
		t.Errorf("Expected default executable 'llama-server', got %q", cfg.Instances.LlamaExecutable)
	}
	if !cfg.Instances.DefaultAutoRestart {
		t.Error("Expected default auto restart to be true")
	}
	if cfg.Instances.DefaultMaxRestarts != 3 {
		t.Errorf("Expected default max restarts 3, got %d", cfg.Instances.DefaultMaxRestarts)
	}
	if cfg.Instances.DefaultRestartDelay != 5 {
		t.Errorf("Expected default restart delay 5, got %d", cfg.Instances.DefaultRestartDelay)
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
server:
  host: "localhost"
  port: 9090
instances:
  port_range: [7000, 8000]
  log_directory: "/custom/logs"
  max_instances: 5
  llama_executable: "/usr/bin/llama-server"
  default_auto_restart: false
  default_max_restarts: 10
  default_restart_delay: 30
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := llamactl.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify values from file
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Instances.PortRange != [2]int{7000, 8000} {
		t.Errorf("Expected port range [7000, 8000], got %v", cfg.Instances.PortRange)
	}
	if cfg.Instances.LogDirectory != "/custom/logs" {
		t.Errorf("Expected log directory '/custom/logs', got %q", cfg.Instances.LogDirectory)
	}
	if cfg.Instances.MaxInstances != 5 {
		t.Errorf("Expected max instances 5, got %d", cfg.Instances.MaxInstances)
	}
	if cfg.Instances.LlamaExecutable != "/usr/bin/llama-server" {
		t.Errorf("Expected executable '/usr/bin/llama-server', got %q", cfg.Instances.LlamaExecutable)
	}
	if cfg.Instances.DefaultAutoRestart {
		t.Error("Expected auto restart to be false")
	}
	if cfg.Instances.DefaultMaxRestarts != 10 {
		t.Errorf("Expected max restarts 10, got %d", cfg.Instances.DefaultMaxRestarts)
	}
	if cfg.Instances.DefaultRestartDelay != 30 {
		t.Errorf("Expected restart delay 30, got %d", cfg.Instances.DefaultRestartDelay)
	}
}

func TestLoadConfig_EnvironmentOverrides(t *testing.T) {
	// Set environment variables
	envVars := map[string]string{
		"LLAMACTL_HOST":                  "0.0.0.0",
		"LLAMACTL_PORT":                  "3000",
		"LLAMACTL_INSTANCE_PORT_RANGE":   "5000-6000",
		"LLAMACTL_LOG_DIR":               "/env/logs",
		"LLAMACTL_MAX_INSTANCES":         "20",
		"LLAMACTL_LLAMA_EXECUTABLE":      "/env/llama-server",
		"LLAMACTL_DEFAULT_AUTO_RESTART":  "false",
		"LLAMACTL_DEFAULT_MAX_RESTARTS":  "7",
		"LLAMACTL_DEFAULT_RESTART_DELAY": "15",
	}

	// Set env vars and ensure cleanup
	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	cfg, err := llamactl.LoadConfig("nonexistent-file.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify environment overrides
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Expected host '0.0.0.0', got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", cfg.Server.Port)
	}
	if cfg.Instances.PortRange != [2]int{5000, 6000} {
		t.Errorf("Expected port range [5000, 6000], got %v", cfg.Instances.PortRange)
	}
	if cfg.Instances.LogDirectory != "/env/logs" {
		t.Errorf("Expected log directory '/env/logs', got %q", cfg.Instances.LogDirectory)
	}
	if cfg.Instances.MaxInstances != 20 {
		t.Errorf("Expected max instances 20, got %d", cfg.Instances.MaxInstances)
	}
	if cfg.Instances.LlamaExecutable != "/env/llama-server" {
		t.Errorf("Expected executable '/env/llama-server', got %q", cfg.Instances.LlamaExecutable)
	}
	if cfg.Instances.DefaultAutoRestart {
		t.Error("Expected auto restart to be false")
	}
	if cfg.Instances.DefaultMaxRestarts != 7 {
		t.Errorf("Expected max restarts 7, got %d", cfg.Instances.DefaultMaxRestarts)
	}
	if cfg.Instances.DefaultRestartDelay != 15 {
		t.Errorf("Expected restart delay 15, got %d", cfg.Instances.DefaultRestartDelay)
	}
}

func TestLoadConfig_FileAndEnvironmentPrecedence(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	configContent := `
server:
  host: "file-host"
  port: 8888
instances:
  max_instances: 5
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Set some environment variables (should override file)
	os.Setenv("LLAMACTL_HOST", "env-host")
	os.Setenv("LLAMACTL_MAX_INSTANCES", "15")
	defer os.Unsetenv("LLAMACTL_HOST")
	defer os.Unsetenv("LLAMACTL_MAX_INSTANCES")

	cfg, err := llamactl.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Environment should override file
	if cfg.Server.Host != "env-host" {
		t.Errorf("Expected env override 'env-host', got %q", cfg.Server.Host)
	}
	if cfg.Instances.MaxInstances != 15 {
		t.Errorf("Expected env override 15, got %d", cfg.Instances.MaxInstances)
	}
	// File should override defaults
	if cfg.Server.Port != 8888 {
		t.Errorf("Expected file value 8888, got %d", cfg.Server.Port)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "invalid-config.yaml")

	invalidContent := `
server:
  host: "localhost"
  port: not-a-number
instances:
  [invalid yaml structure
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err = llamactl.LoadConfig(configFile)
	if err == nil {
		t.Error("Expected LoadConfig to return error for invalid YAML")
	}
}

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected [2]int
	}{
		{"hyphen format", "8000-9000", [2]int{8000, 9000}},
		{"comma format", "8000,9000", [2]int{8000, 9000}},
		{"with spaces", "8000 - 9000", [2]int{8000, 9000}},
		{"comma with spaces", "8000 , 9000", [2]int{8000, 9000}},
		{"single number", "8000", [2]int{0, 0}},
		{"invalid format", "not-a-range", [2]int{0, 0}},
		{"non-numeric", "start-end", [2]int{0, 0}},
		{"empty string", "", [2]int{0, 0}},
		{"too many parts", "8000-9000-10000", [2]int{0, 0}},
		{"negative numbers", "-1000--500", [2]int{0, 0}}, // Invalid parsing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := llamactl.ParsePortRange(tt.input)
			if result != tt.expected {
				t.Errorf("ParsePortRange(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Remove the getDefaultConfigLocations test entirely

func TestLoadConfig_EnvironmentVariableTypes(t *testing.T) {
	// Test that environment variables are properly converted to correct types
	testCases := []struct {
		envVar   string
		envValue string
		checkFn  func(*llamactl.Config) bool
		desc     string
	}{
		{
			envVar:   "LLAMACTL_PORT",
			envValue: "invalid-port",
			checkFn:  func(c *llamactl.Config) bool { return c.Server.Port == 8080 }, // Should keep default
			desc:     "invalid port number should keep default",
		},
		{
			envVar:   "LLAMACTL_MAX_INSTANCES",
			envValue: "not-a-number",
			checkFn:  func(c *llamactl.Config) bool { return c.Instances.MaxInstances == -1 }, // Should keep default
			desc:     "invalid max instances should keep default",
		},
		{
			envVar:   "LLAMACTL_DEFAULT_AUTO_RESTART",
			envValue: "invalid-bool",
			checkFn:  func(c *llamactl.Config) bool { return c.Instances.DefaultAutoRestart == true }, // Should keep default
			desc:     "invalid boolean should keep default",
		},
		{
			envVar:   "LLAMACTL_INSTANCE_PORT_RANGE",
			envValue: "invalid-range",
			checkFn:  func(c *llamactl.Config) bool { return c.Instances.PortRange == [2]int{8000, 9000} }, // Should keep default
			desc:     "invalid port range should keep default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			os.Setenv(tc.envVar, tc.envValue)
			defer os.Unsetenv(tc.envVar)

			cfg, err := llamactl.LoadConfig("nonexistent-file.yaml")
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			if !tc.checkFn(&cfg) {
				t.Errorf("Test failed: %s", tc.desc)
			}
		})
	}
}

func TestLoadConfig_PartialFile(t *testing.T) {
	// Test that partial config files work correctly (missing sections should use defaults)
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "partial-config.yaml")

	// Only specify server config, instances should use defaults
	configContent := `
server:
  host: "partial-host"
  port: 7777
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := llamactl.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Server config should be from file
	if cfg.Server.Host != "partial-host" {
		t.Errorf("Expected host 'partial-host', got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("Expected port 7777, got %d", cfg.Server.Port)
	}

	// Instances config should be defaults
	if cfg.Instances.PortRange != [2]int{8000, 9000} {
		t.Errorf("Expected default port range [8000, 9000], got %v", cfg.Instances.PortRange)
	}
	if cfg.Instances.MaxInstances != -1 {
		t.Errorf("Expected default max instances -1, got %d", cfg.Instances.MaxInstances)
	}
}
