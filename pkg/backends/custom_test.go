package backends_test

import (
	"encoding/json"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"reflect"
	"testing"
)

func TestCustomBuildCommandArgs(t *testing.T) {
	tests := []struct {
		name     string
		options  backends.CustomServerOptions
		expected []string
	}{
		{
			name:     "port substitution",
			options:  backends.CustomServerOptions{Port: 8123, Args: []string{"serve", "--port", "{port}"}},
			expected: []string{"serve", "--port", "8123"},
		},
		{
			name:     "both placeholders in one arg",
			options:  backends.CustomServerOptions{Port: 9000, Model: "m1", Args: []string{"--run={model}-{port}"}},
			expected: []string{"--run=m1-9000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.options.BuildCommandArgs()
			if !reflect.DeepEqual(args, tt.expected) {
				t.Errorf("BuildCommandArgs() = %v, want %v", args, tt.expected)
			}
		})
	}
}

// Args are persisted; mutation would corrupt subsequent launches.
func TestCustomBuildCommandArgs_DoesNotMutateArgs(t *testing.T) {
	options := backends.CustomServerOptions{
		Port: 8123,
		Args: []string{"--port", "{port}"},
	}

	options.BuildCommandArgs()

	if options.Args[1] != "{port}" {
		t.Errorf("original args must not be mutated, got %v", options.Args)
	}
}

func TestOptionsGetHealthPath(t *testing.T) {
	backendConfig := &config.BackendConfig{
		Custom: map[string]config.BackendSettings{
			"x":    {HealthPath: "/ready"},
			"bare": {HealthPath: "ready"},
		},
		LlamaCpp: config.BackendSettings{HealthPath: "/llm-ready"},
	}

	tests := []struct {
		name     string
		options  backends.Options
		expected string
	}{
		{
			name:     "custom configured path",
			options:  backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "x"}},
			expected: "/ready",
		},
		{
			name:     "missing leading slash is prefixed",
			options:  backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "bare"}},
			expected: "/ready",
		},
		{
			name:     "entry without path falls back to default",
			options:  backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "other"}},
			expected: "/health",
		},
		{
			name:     "built-in backends read the same field",
			options:  backends.Options{BackendType: backends.BackendTypeLlamaCpp, LlamaServerOptions: &backends.LlamaServerOptions{}},
			expected: "/llm-ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.options.GetHealthPath(backendConfig); got != tt.expected {
				t.Errorf("GetHealthPath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCustomValidate(t *testing.T) {
	tests := []struct {
		name      string
		options   *backends.CustomServerOptions
		expectErr bool
	}{
		{name: "nil options", options: nil, expectErr: true},
		{name: "missing name", options: &backends.CustomServerOptions{}, expectErr: true},
		{
			name: "port out of range",
			options: &backends.CustomServerOptions{
				Name: "x",
				Port: 70000,
			},
			expectErr: true,
		},
		{
			name: "valid options",
			options: &backends.CustomServerOptions{
				Name:  "x",
				Model: "org/model",
				Args:  []string{"serve", "--model", "{model}", "--port", "{port}"},
				Port:  8123,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Validate()
			if (err != nil) != tt.expectErr {
				t.Errorf("Validate() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}

func TestCustomParseCommandUnsupported(t *testing.T) {
	var opts backends.CustomServerOptions
	if _, err := opts.ParseCommand("anything"); err == nil {
		t.Error("expected ParseCommand to return an error for custom backends")
	}
}

func TestCustomOptionsJSONRoundTrip(t *testing.T) {
	opts := backends.Options{
		BackendType: backends.BackendTypeCustom,
		CustomServerOptions: &backends.CustomServerOptions{
			Name:  "my-engine",
			Model: "org/model",
			Port:  8123,
			Args:  []string{"--no-webui", "--port", "{port}"},
		},
	}

	// Marshal a pointer: MarshalJSON has a pointer receiver, so marshaling
	// the value would silently skip it and drop backend_options.
	data, err := json.Marshal(&opts)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var unmarshaled backends.Options
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if unmarshaled.BackendType != backends.BackendTypeCustom {
		t.Errorf("expected backend_type custom, got %s", unmarshaled.BackendType)
	}
	if got := unmarshaled.CustomServerOptions; !reflect.DeepEqual(got, opts.CustomServerOptions) {
		t.Errorf("round trip mismatch: got %+v, want %+v", got, opts.CustomServerOptions)
	}
}

func TestCustomGetCommandAndBuildCommandArgs(t *testing.T) {
	backendConfig := &config.BackendConfig{
		Custom: map[string]config.BackendSettings{
			"my-engine": {
				Command: "my-engine",
				Args:    []string{"serve"},
			},
		},
	}

	t.Run("known name resolves command and merges args", func(t *testing.T) {
		opts := backends.Options{
			BackendType: backends.BackendTypeCustom,
			CustomServerOptions: &backends.CustomServerOptions{
				Name: "my-engine",
				Port: 8456,
				Args: []string{"--port", "{port}"},
			},
		}

		if cmd := opts.GetCommand(backendConfig, nil, ""); cmd != "my-engine" {
			t.Errorf("GetCommand() = %q, want %q", cmd, "my-engine")
		}

		args := opts.BuildCommandArgs(backendConfig, nil)
		expected := []string{"serve", "--port", "8456"}
		if !reflect.DeepEqual(args, expected) {
			t.Errorf("BuildCommandArgs() = %v, want %v", args, expected)
		}
	})

	t.Run("unknown name yields empty command without panic", func(t *testing.T) {
		opts := backends.Options{
			BackendType: backends.BackendTypeCustom,
			CustomServerOptions: &backends.CustomServerOptions{
				Name: "gone",
				Args: []string{"--port", "{port}"},
			},
		}

		if cmd := opts.GetCommand(backendConfig, nil, ""); cmd != "" {
			t.Errorf("GetCommand() = %q, want empty", cmd)
		}
	})

	t.Run("nil custom options yields empty command without panic", func(t *testing.T) {
		opts := backends.Options{BackendType: backends.BackendTypeCustom}

		if cmd := opts.GetCommand(backendConfig, nil, ""); cmd != "" {
			t.Errorf("GetCommand() = %q, want empty", cmd)
		}
	})
}
