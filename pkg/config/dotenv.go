package config

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func loadDotEnv(explicitConfigPath string) string {
	var searchPaths []string

	if explicitConfigPath != "" {
		absPath, err := filepath.Abs(explicitConfigPath)
		if err == nil {
			searchPaths = append(searchPaths, filepath.Join(filepath.Dir(absPath), ".env"))
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		searchPaths = append(searchPaths, filepath.Join(cwd, ".env"))
	}

	for _, dir := range getPlatformConfigDirs() {
		searchPaths = append(searchPaths, filepath.Join(dir, ".env"))
	}

	for _, p := range searchPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		vars := parseDotEnv(data)
		for key, value := range vars {
			if _, exists := os.LookupEnv(key); !exists {
				os.Setenv(key, value)
			}
		}
		log.Printf("Loaded .env from %s", p)
		return p
	}

	return ""
}

func parseDotEnv(data []byte) map[string]string {
	result := make(map[string]string)

	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		lineStr := string(line)

		lineStr = strings.TrimPrefix(lineStr, "export ")
		lineStr = strings.TrimSpace(lineStr)

		eqIdx := strings.Index(lineStr, "=")
		if eqIdx < 0 {
			continue
		}

		key := strings.TrimSpace(lineStr[:eqIdx])
		value := strings.TrimSpace(lineStr[eqIdx+1:])

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		result[key] = value
	}

	return result
}
