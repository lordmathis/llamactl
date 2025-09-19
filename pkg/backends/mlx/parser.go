package mlx

import (
	"llamactl/pkg/backends"
)

// ParseMlxCommand parses a mlx_lm.server command string into MlxServerOptions
// Supports multiple formats:
// 1. Full command: "mlx_lm.server --model model/path"
// 2. Full path: "/usr/local/bin/mlx_lm.server --model model/path"
// 3. Args only: "--model model/path --host 0.0.0.0"
// 4. Multiline commands with backslashes
func ParseMlxCommand(command string) (*MlxServerOptions, error) {
	config := backends.CommandParserConfig{
		ExecutableNames:  []string{"mlx_lm.server"},
		MultiValuedFlags: map[string]struct{}{}, // MLX has no multi-valued flags
	}

	var mlxOptions MlxServerOptions
	if err := backends.ParseCommand(command, config, &mlxOptions); err != nil {
		return nil, err
	}

	return &mlxOptions, nil
}

// isValidLogLevel validates MLX log levels
func isValidLogLevel(level string) bool {
	validLevels := []string{"DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"}
	for _, valid := range validLevels {
		if level == valid {
			return true
		}
	}
	return false
}

