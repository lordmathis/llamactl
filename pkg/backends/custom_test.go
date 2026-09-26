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
			name:     "literal passthrough",
			options:  backends.CustomServerOptions{Args: []string{"serve", "--no-webui"}},
			expected: []string{"serve", "--no-webui"},
		},
		{
			name:     "port substitution",
			options:  backends.CustomServerOptions{Port: 8123, Args: []string{"serve", "--port", "{port}"}},
			expected: []string{"serve", "--port", "8123"},
		},
		{
			name:     "model substitution",
			options:  backends.CustomServerOptions{Model: "org/model", Args: []string{"--model", "{model}"}},
			expected: []string{"--model", "org/model"},
		},
		{
			name:     "mixed placeholders in one arg",
			options:  backends.CustomServerOptions{Port: 9000, Model: "m1", Args: []string{"--run={model}-{port}"}},
			expected: []string{"--run=m1-9000"},
		},
		{
			name:     "no args",
			options:  backends.CustomServerOptions{Name: "x"},
			expected: []string{},
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
		LlamaCpp: config.BackendSettings{HealthPath: "/health"},
	}

	tests := []struct {
		name      string
		options   backends.Options
		nilConfig bool
		expected  string
	}{
		{
			name:     "custom configured path",
			options:  backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "x"}},
			expected: "/ready",
		},
		{
			name:     "custom entry without path falls back to default",
			options:  backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "other"}},
			expected: "/health",
		},
		{
			name:     "missing leading slash is normalized",
			options:  backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "bare"}},
			expected: "/ready",
		},
		{
			name:     "custom nil options falls back to default",
			options:  backends.Options{BackendType: backends.BackendTypeCustom},
			expected: "/health",
		},
		{
			name:     "llama cpp configured path",
			options:  backends.Options{BackendType: backends.BackendTypeLlamaCpp, LlamaServerOptions: &backends.LlamaServerOptions{}},
			expected: "/health",
		},
		{
			name:     "unknown backend type falls back to default",
			options:  backends.Options{BackendType: backends.BackendTypeUnknown},
			expected: "/health",
		},
		{
			name:      "nil backend config falls back to default",
			options:   backends.Options{BackendType: backends.BackendTypeCustom, CustomServerOptions: &backends.CustomServerOptions{Name: "x"}},
			nilConfig: true,
			expected:  "/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := backendConfig
			if tt.nilConfig {
				cfg = nil
			}
			if got := tt.options.GetHealthPath(cfg); got != tt.expected {
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
			name: "model placeholder is validated by the manager against merged args",
			options: &backends.CustomServerOptions{
				Name: "x",
				Args: []string{"--model", "{model}"},
			},
			expectErr: false,
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
			Name:  "splash",
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

	got := unmarshaled.CustomServerOptions
	if got == nil {
		t.Fatal("expected CustomServerOptions to be populated")
	}
	if got.Name != "splash" {
		t.Errorf("expected name 'splash', got %q", got.Name)
	}
	if got.Model != "org/model" {
		t.Errorf("expected model 'org/model', got %q", got.Model)
	}
	if got.Port != 8123 {
		t.Errorf("expected port 8123, got %d", got.Port)
	}
	if !reflect.DeepEqual(got.Args, opts.CustomServerOptions.Args) {
		t.Errorf("expected args %v, got %v", opts.CustomServerOptions.Args, got.Args)
	}
}

func TestCustomGetCommandAndBuildCommandArgs(t *testing.T) {
	backendConfig := &config.BackendConfig{
		Custom: map[string]config.BackendSettings{
			"splash": {
				Command: "splash",
				Args:    []string{"serve"},
			},
		},
	}

	t.Run("known name resolves command and merges args", func(t *testing.T) {
		opts := backends.Options{
			BackendType: backends.BackendTypeCustom,
			CustomServerOptions: &backends.CustomServerOptions{
				Name: "splash",
				Port: 8456,
				Args: []string{"--port", "{port}"},
			},
		}

		if cmd := opts.GetCommand(backendConfig, nil, ""); cmd != "splash" {
			t.Errorf("GetCommand() = %q, want %q", cmd, "splash")
		}

		args := opts.BuildCommandArgs(backendConfig, nil)
		expected := []string{"serve", "--port", "8456"}
		if !reflect.DeepEqual(args, expected) {
			t.Errorf("BuildCommandArgs() = %v, want %v", args, expected)
		}
	})

	t.Run("command override wins", func(t *testing.T) {
		opts := backends.Options{
			BackendType: backends.BackendTypeCustom,
			CustomServerOptions: &backends.CustomServerOptions{
				Name: "splash",
				Args: []string{"--port", "{port}"},
			},
		}

		if cmd := opts.GetCommand(backendConfig, nil, "/opt/splash/bin/splash"); cmd != "/opt/splash/bin/splash" {
			t.Errorf("GetCommand() = %q, want override", cmd)
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
