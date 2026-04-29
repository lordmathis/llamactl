package config

import (
	"os"
	"regexp"
)

var envVarPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::-([^}]*))?\}`)

func expandEnvVars(data []byte) []byte {
	return envVarPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		submatch := envVarPattern.FindSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		varName := string(submatch[1])
		value, isSet := os.LookupEnv(varName)

		if len(submatch) >= 3 && submatch[2] != nil {
			defaultVal := string(submatch[2])
			if !isSet || value == "" {
				return []byte(defaultVal)
			}
			return []byte(value)
		}

		if isSet {
			return []byte(value)
		}
		return match
	})
}

func expandString(s string) string {
	return string(expandEnvVars([]byte(s)))
}
