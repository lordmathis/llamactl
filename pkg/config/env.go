package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

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
	if logRotationMaxSize := os.Getenv("LLAMACTL_LOG_ROTATION_MAX_SIZE"); logRotationMaxSize != "" {
		if m, err := strconv.Atoi(logRotationMaxSize); err == nil {
			cfg.Instances.LogRotationMaxSize = m
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
