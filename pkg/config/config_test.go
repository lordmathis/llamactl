package config_test

import (
	"llamactl/pkg/config"
	"os"
	"path/filepath"
	"testing"
)

// GetBackendSettings resolves backend settings
func getBackendSettings(bc *config.BackendConfig, backendType string) config.BackendSettings {
	switch backendType {
	case "llama-cpp":
		return bc.LlamaCpp
	case "vllm":
		return bc.VLLM
	case "mlx":
		return bc.MLX
	default:
		return config.BackendSettings{}
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Test loading config when no file exists and no env vars set
	cfg, err := config.LoadConfig("nonexistent-file.yaml")
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

	homedir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get user home directory: %v", err)
	}

	if cfg.Instances.InstancesDir != filepath.Join(homedir, ".local", "share", "llamactl", "instances") {
		t.Errorf("Expected default instances directory '%s', got %q", filepath.Join(homedir, ".local", "share", "llamactl", "instances"), cfg.Instances.InstancesDir)
	}
	if cfg.Instances.Logging.LogsDir != filepath.Join(homedir, ".local", "share", "llamactl", "logs") {
		t.Errorf("Expected default logs directory '%s', got %q", filepath.Join(homedir, ".local", "share", "llamactl", "logs"), cfg.Instances.Logging.LogsDir)
	}
	if !cfg.Instances.AutoCreateDirs {
		t.Error("Expected default instances auto-create to be true")
	}
	if cfg.Instances.PortRange != [2]int{8000, 9000} {
		t.Errorf("Expected default port range [8000, 9000], got %v", cfg.Instances.PortRange)
	}
	if cfg.Instances.MaxInstances != -1 {
		t.Errorf("Expected default max instances -1, got %d", cfg.Instances.MaxInstances)
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
  max_instances: 5
  logging:
    logs_dir: "/custom/logs"
  llama_executable: "/usr/bin/llama-server"
  default_auto_restart: false
  default_max_restarts: 10
  default_restart_delay: 30
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := config.LoadConfig(configFile)
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
	if cfg.Instances.Logging.LogsDir != "/custom/logs" {
		t.Errorf("Expected logs directory '/custom/logs', got %q", cfg.Instances.Logging.LogsDir)
	}
	if cfg.Instances.MaxInstances != 5 {
		t.Errorf("Expected max instances 5, got %d", cfg.Instances.MaxInstances)
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
		"LLAMACTL_LOGS_DIR":              "/env/logs",
		"LLAMACTL_MAX_INSTANCES":         "20",
		"LLAMACTL_DEFAULT_AUTO_RESTART":  "false",
		"LLAMACTL_DEFAULT_MAX_RESTARTS":  "7",
		"LLAMACTL_DEFAULT_RESTART_DELAY": "15",
	}

	// Set env vars and ensure cleanup
	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	cfg, err := config.LoadConfig("nonexistent-file.yaml")
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
	if cfg.Instances.Logging.LogsDir != "/env/logs" {
		t.Errorf("Expected logs directory '/env/logs', got %q", cfg.Instances.Logging.LogsDir)
	}
	if cfg.Instances.MaxInstances != 20 {
		t.Errorf("Expected max instances 20, got %d", cfg.Instances.MaxInstances)
	}
	if cfg.Backends.LlamaCpp.Command != "llama-server" {
		t.Errorf("Expected default llama command 'llama-server', got %q", cfg.Backends.LlamaCpp.Command)
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

	cfg, err := config.LoadConfig(configFile)
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
			result := config.ParsePortRange(tt.input)
			if result != tt.expected {
				t.Errorf("ParsePortRange(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetBackendSettings_NewStructuredConfig(t *testing.T) {
	bc := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "custom-llama",
			Args:    []string{"--verbose"},
			Docker: &config.DockerSettings{
				Enabled:     true,
				Image:       "custom-llama:latest",
				Args:        []string{"--gpus", "all"},
				Environment: map[string]string{"CUDA_VISIBLE_DEVICES": "1"},
			},
		},
		VLLM: config.BackendSettings{
			Command: "custom-vllm",
			Args:    []string{"serve", "--debug"},
		},
		MLX: config.BackendSettings{
			Command: "custom-mlx",
			Args:    []string{},
		},
	}

	// Test llama-cpp with Docker
	settings := getBackendSettings(bc, "llama-cpp")
	if settings.Command != "custom-llama" {
		t.Errorf("Expected command 'custom-llama', got %q", settings.Command)
	}
	if len(settings.Args) != 1 || settings.Args[0] != "--verbose" {
		t.Errorf("Expected args ['--verbose'], got %v", settings.Args)
	}
	if settings.Docker == nil || !settings.Docker.Enabled {
		t.Error("Expected Docker to be enabled")
	}
	if settings.Docker.Image != "custom-llama:latest" {
		t.Errorf("Expected Docker image 'custom-llama:latest', got %q", settings.Docker.Image)
	}

	// Test vLLM without Docker
	settings = getBackendSettings(bc, "vllm")
	if settings.Command != "custom-vllm" {
		t.Errorf("Expected command 'custom-vllm', got %q", settings.Command)
	}
	if len(settings.Args) != 2 || settings.Args[0] != "serve" || settings.Args[1] != "--debug" {
		t.Errorf("Expected args ['serve', '--debug'], got %v", settings.Args)
	}
	if settings.Docker != nil && settings.Docker.Enabled {
		t.Error("Expected Docker to be disabled or nil")
	}

	// Test MLX
	settings = getBackendSettings(bc, "mlx")
	if settings.Command != "custom-mlx" {
		t.Errorf("Expected command 'custom-mlx', got %q", settings.Command)
	}
}

func TestLoadConfig_BackendEnvironmentVariables(t *testing.T) {
	// Test that backend environment variables work correctly
	envVars := map[string]string{
		"LLAMACTL_LLAMACPP_COMMAND":        "env-llama",
		"LLAMACTL_LLAMACPP_ARGS":           "--verbose --threads 4",
		"LLAMACTL_LLAMACPP_DOCKER_ENABLED": "true",
		"LLAMACTL_LLAMACPP_DOCKER_IMAGE":   "env-llama:latest",
		"LLAMACTL_LLAMACPP_DOCKER_ARGS":    "run --rm --network host --gpus all",
		"LLAMACTL_LLAMACPP_DOCKER_ENV":     "CUDA_VISIBLE_DEVICES=0,OMP_NUM_THREADS=4",
		"LLAMACTL_VLLM_COMMAND":            "env-vllm",
		"LLAMACTL_VLLM_DOCKER_ENABLED":     "false",
		"LLAMACTL_VLLM_DOCKER_IMAGE":       "env-vllm:latest",
		"LLAMACTL_VLLM_DOCKER_ENV":         "PYTORCH_CUDA_ALLOC_CONF=max_split_size_mb:512,CUDA_VISIBLE_DEVICES=1",
		"LLAMACTL_MLX_COMMAND":             "env-mlx",
	}

	// Set env vars and ensure cleanup
	for key, value := range envVars {
		os.Setenv(key, value)
		defer os.Unsetenv(key)
	}

	cfg, err := config.LoadConfig("nonexistent-file.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify llama-cpp environment overrides
	if cfg.Backends.LlamaCpp.Command != "env-llama" {
		t.Errorf("Expected llama command 'env-llama', got %q", cfg.Backends.LlamaCpp.Command)
	}
	expectedArgs := []string{"--verbose", "--threads", "4"}
	if len(cfg.Backends.LlamaCpp.Args) != len(expectedArgs) {
		t.Errorf("Expected llama args %v, got %v", expectedArgs, cfg.Backends.LlamaCpp.Args)
	}
	if !cfg.Backends.LlamaCpp.Docker.Enabled {
		t.Error("Expected llama Docker to be enabled")
	}
	if cfg.Backends.LlamaCpp.Docker.Image != "env-llama:latest" {
		t.Errorf("Expected llama Docker image 'env-llama:latest', got %q", cfg.Backends.LlamaCpp.Docker.Image)
	}
	expectedDockerArgs := []string{"run", "--rm", "--network", "host", "--gpus", "all"}
	if len(cfg.Backends.LlamaCpp.Docker.Args) != len(expectedDockerArgs) {
		t.Errorf("Expected llama Docker args %v, got %v", expectedDockerArgs, cfg.Backends.LlamaCpp.Docker.Args)
	}
	if cfg.Backends.LlamaCpp.Docker.Environment["CUDA_VISIBLE_DEVICES"] != "0" {
		t.Errorf("Expected CUDA_VISIBLE_DEVICES=0, got %q", cfg.Backends.LlamaCpp.Docker.Environment["CUDA_VISIBLE_DEVICES"])
	}
	if cfg.Backends.LlamaCpp.Docker.Environment["OMP_NUM_THREADS"] != "4" {
		t.Errorf("Expected OMP_NUM_THREADS=4, got %q", cfg.Backends.LlamaCpp.Docker.Environment["OMP_NUM_THREADS"])
	}

	// Verify vLLM environment overrides
	if cfg.Backends.VLLM.Command != "env-vllm" {
		t.Errorf("Expected vLLM command 'env-vllm', got %q", cfg.Backends.VLLM.Command)
	}
	if cfg.Backends.VLLM.Docker.Enabled {
		t.Error("Expected vLLM Docker to be disabled")
	}
	if cfg.Backends.VLLM.Docker.Environment["PYTORCH_CUDA_ALLOC_CONF"] != "max_split_size_mb:512" {
		t.Errorf("Expected PYTORCH_CUDA_ALLOC_CONF=max_split_size_mb:512, got %q", cfg.Backends.VLLM.Docker.Environment["PYTORCH_CUDA_ALLOC_CONF"])
	}

	// Verify MLX environment overrides
	if cfg.Backends.MLX.Command != "env-mlx" {
		t.Errorf("Expected MLX command 'env-mlx', got %q", cfg.Backends.MLX.Command)
	}
}

func TestLoadConfig_LocalNode(t *testing.T) {
	t.Run("default local node", func(t *testing.T) {
		cfg, err := config.LoadConfig("nonexistent-file.yaml")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.LocalNode != "main" {
			t.Errorf("Expected default local node 'main', got %q", cfg.LocalNode)
		}
	})

	t.Run("local node from file", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test-config.yaml")

		configContent := `
local_node: "worker1"
nodes:
  worker1:
    address: ""
  worker2:
    address: "http://192.168.1.10:8080"
    api_key: "test-key"
`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write test config file: %v", err)
		}

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.LocalNode != "worker1" {
			t.Errorf("Expected local node 'worker1', got %q", cfg.LocalNode)
		}

		// Verify nodes map (includes default "main" + worker1 + worker2)
		if len(cfg.Nodes) != 2 {
			t.Errorf("Expected 2 nodes (default worker1 + worker2), got %d", len(cfg.Nodes))
		}

		// Verify local node exists and is empty
		localNode, exists := cfg.Nodes["worker1"]
		if !exists {
			t.Error("Expected local node 'worker1' to exist in nodes map")
		}
		if localNode.Address != "" {
			t.Errorf("Expected local node address to be empty, got %q", localNode.Address)
		}
		if localNode.APIKey != "" {
			t.Errorf("Expected local node api_key to be empty, got %q", localNode.APIKey)
		}

		// Verify remote node
		remoteNode, exists := cfg.Nodes["worker2"]
		if !exists {
			t.Error("Expected remote node 'worker2' to exist in nodes map")
		}
		if remoteNode.Address != "http://192.168.1.10:8080" {
			t.Errorf("Expected remote node address 'http://192.168.1.10:8080', got %q", remoteNode.Address)
		}

		// Verify default main node still exists
		_, exists = cfg.Nodes["main"]
		if exists {
			t.Error("Default 'main' node should not exist when local_node is overridden")
		}
	})

	t.Run("custom local node name in config", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test-config.yaml")

		configContent := `
local_node: "primary"
nodes:
  primary:
    address: ""
  worker1:
    address: "http://192.168.1.10:8080"
`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write test config file: %v", err)
		}

		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.LocalNode != "primary" {
			t.Errorf("Expected local node 'primary', got %q", cfg.LocalNode)
		}

		// Verify nodes map includes default "main" + primary + worker1
		if len(cfg.Nodes) != 2 {
			t.Errorf("Expected 2 nodes (primary + worker1), got %d", len(cfg.Nodes))
		}

		localNode, exists := cfg.Nodes["primary"]
		if !exists {
			t.Error("Expected local node 'primary' to exist in nodes map")
		}
		if localNode.Address != "" {
			t.Errorf("Expected local node address to be empty, got %q", localNode.Address)
		}
	})

	t.Run("local node from environment variable", func(t *testing.T) {
		os.Setenv("LLAMACTL_LOCAL_NODE", "custom-node")
		defer os.Unsetenv("LLAMACTL_LOCAL_NODE")

		cfg, err := config.LoadConfig("nonexistent-file.yaml")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.LocalNode != "custom-node" {
			t.Errorf("Expected local node 'custom-node' from env var, got %q", cfg.LocalNode)
		}
	})
}
