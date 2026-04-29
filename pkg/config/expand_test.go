package config

import (
	"os"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "simple variable",
			input:    "host: ${MY_HOST}",
			envVars:  map[string]string{"MY_HOST": "localhost"},
			expected: "host: localhost",
		},
		{
			name:     "unset variable left as-is",
			input:    "host: ${UNSET_VAR}",
			envVars:  nil,
			expected: "host: ${UNSET_VAR}",
		},
		{
			name:     "variable with default when unset",
			input:    "port: ${MY_PORT:-8080}",
			envVars:  nil,
			expected: "port: 8080",
		},
		{
			name:     "variable with default when set",
			input:    "port: ${MY_PORT:-8080}",
			envVars:  map[string]string{"MY_PORT": "9090"},
			expected: "port: 9090",
		},
		{
			name:     "variable with empty default",
			input:    "val: ${MY_VAR:-}",
			envVars:  nil,
			expected: "val: ",
		},
		{
			name:     "variable with default when empty",
			input:    "val: ${MY_VAR:-fallback}",
			envVars:  map[string]string{"MY_VAR": ""},
			expected: "val: fallback",
		},
		{
			name:     "no braces not expanded",
			input:    "host: $MY_HOST",
			envVars:  map[string]string{"MY_HOST": "localhost"},
			expected: "host: $MY_HOST",
		},
		{
			name:     "multiple substitutions",
			input:    "${HOST}:${PORT}",
			envVars:  map[string]string{"HOST": "localhost", "PORT": "8080"},
			expected: "localhost:8080",
		},
		{
			name:     "inline in longer string",
			input:    "url: http://${HOST}:${PORT}/api",
			envVars:  map[string]string{"HOST": "example.com", "PORT": "443"},
			expected: "url: http://example.com:443/api",
		},
		{
			name:     "default with special characters",
			input:    "val: ${VAR:-/usr/local/bin}",
			envVars:  nil,
			expected: "val: /usr/local/bin",
		},
		{
			name:     "empty input",
			input:    "",
			envVars:  nil,
			expected: "",
		},
		{
			name:     "no placeholders",
			input:    "just a normal string",
			envVars:  nil,
			expected: "just a normal string",
		},
		{
			name:     "underscore in variable name",
			input:    "${MY_LONG_VAR_NAME:-default}",
			envVars:  map[string]string{"MY_LONG_VAR_NAME": "value"},
			expected: "value",
		},
		{
			name:     "variable starting with underscore",
			input:    "${_PRIVATE_VAR}",
			envVars:  map[string]string{"_PRIVATE_VAR": "secret"},
			expected: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k := range tt.envVars {
				os.Unsetenv(k)
			}
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := expandString(tt.input)
			if result != tt.expected {
				t.Errorf("expandString(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDotEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "basic key-value pairs",
			input: "API_KEY=sk-abc123\nDB_HOST=localhost\nDB_PORT=5432\n",
			expected: map[string]string{
				"API_KEY": "sk-abc123",
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
		},
		{
			name: "comments and blank lines",
			input: "# comment\n\nAPI_KEY=value\n# another comment\n",
			expected: map[string]string{
				"API_KEY": "value",
			},
		},
		{
			name:     "empty value",
			input:    "EMPTY_VAR=\n",
			expected: map[string]string{"EMPTY_VAR": ""},
		},
		{
			name:     "double quoted value",
			input:    `QUOTED_VAR="hello world"` + "\n",
			expected: map[string]string{"QUOTED_VAR": "hello world"},
		},
		{
			name:     "single quoted value",
			input:    "SINGLE_QUOTED='literal text'\n",
			expected: map[string]string{"SINGLE_QUOTED": "literal text"},
		},
		{
			name:     "export prefix",
			input:    "export EXPORTED_VAR=value\n",
			expected: map[string]string{"EXPORTED_VAR": "value"},
		},
		{
			name: "leading whitespace on lines",
			input: "  KEY1=val1\n  KEY2=val2\n",
			expected: map[string]string{
				"KEY1": "val1",
				"KEY2": "val2",
			},
		},
		{
			name: "inline comments not supported",
			input: "KEY=val # comment\n",
			expected: map[string]string{
				"KEY": "val # comment",
			},
		},
		{
			name:     "whitespace trimming",
			input:    "  KEY  =  value  \n",
			expected: map[string]string{"KEY": "value"},
		},
		{
			name:     "line with no equals sign is skipped",
			input:    "NOEQUALS\nKEY=val\n",
			expected: map[string]string{"KEY": "val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDotEnv([]byte(tt.input))
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d vars, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("parseDotEnv: key %q = %q, expected %q", k, result[k], v)
				}
			}
		})
	}
}
