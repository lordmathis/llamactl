package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func loadDotEnv(explicitConfigPath string) (string, error) {
	var searchPaths []string

	if explicitConfigPath != "" {
		if absPath, err := filepath.Abs(explicitConfigPath); err == nil {
			searchPaths = append(searchPaths, filepath.Join(filepath.Dir(absPath), ".env"))
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(cwd, ".env"))
	}
	for _, dir := range getPlatformConfigDirs() {
		searchPaths = append(searchPaths, filepath.Join(dir, ".env"))
	}

	for _, p := range searchPaths {
		info, err := os.Lstat(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("inspecting .env file %s: %w", p, err)
		}
		if err := verifyDotEnvInfo(p, info); err != nil {
			return "", err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("reading .env file %s: %w", p, err)
		}
		vars := parseDotEnv(data)
		for key, value := range vars {
			if _, exists := os.LookupEnv(key); !exists {
				os.Setenv(key, value)
			}
		}
		log.Printf("Loaded .env from %s", p)
		return p, nil
	}

	return "", nil
}

func verifyDotEnvInfo(path string, info fs.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("rejecting .env file %s: not a regular file (mode %s)", path, info.Mode())
	}
	// Windows reports 0666 for every writable file
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("rejecting .env file %s: writable by group or others (%s)", path, info.Mode().Perm())
	}
	return nil
}

func parseDotEnv(data []byte) map[string]string {
	result := make(map[string]string)

	for line := range bytes.SplitSeq(data, []byte("\n")) {
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
