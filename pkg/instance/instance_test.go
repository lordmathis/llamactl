package instance_test

import (
	"encoding/json"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/testutil"
	"os"
	"testing"
	"time"
)

func TestNewInstance(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command: "llama-server",
				Args:    []string{},
			},
			MLX: config.BackendSettings{
				Command: "mlx_lm.server",
				Args:    []string{},
			},
			VLLM: config.BackendSettings{
				Command: "vllm",
				Args:    []string{"serve"},
			},
		},
		Instances: config.InstancesConfig{
			DefaultAutoRestart:  true,
			LogsDir:             "/tmp/test",
			DefaultMaxRestarts:  3,
			DefaultRestartDelay: 5,
		},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	inst := instance.New("test-instance", globalConfig, options, mockOnStatusChange)

	if inst.Name != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %q", inst.Name)
	}
	if inst.IsRunning() {
		t.Error("New instance should not be running")
	}

	// Check that options were properly set with defaults applied
	opts := inst.GetOptions()
	if opts.BackendOptions.LlamaServerOptions.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.BackendOptions.LlamaServerOptions.Model)
	}
	if inst.GetPort() != 8080 {
		t.Errorf("Expected port 8080, got %d", inst.GetPort())
	}

	// Check that defaults were applied
	if opts.AutoRestart == nil || !*opts.AutoRestart {
		t.Error("Expected AutoRestart to be true (default)")
	}
	if opts.MaxRestarts == nil || *opts.MaxRestarts != 3 {
		t.Errorf("Expected MaxRestarts to be 3 (default), got %v", opts.MaxRestarts)
	}
	if opts.RestartDelay == nil || *opts.RestartDelay != 5 {
		t.Errorf("Expected RestartDelay to be 5 (default), got %v", opts.RestartDelay)
	}

	// Test that explicit values override defaults
	autoRestart := false
	maxRestarts := 10
	optionsWithOverrides := &instance.Options{
		AutoRestart: &autoRestart,
		MaxRestarts: &maxRestarts,
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	inst2 := instance.New("test-override", globalConfig, optionsWithOverrides, mockOnStatusChange)
	opts2 := inst2.GetOptions()

	if opts2.AutoRestart == nil || *opts2.AutoRestart {
		t.Error("Expected AutoRestart to be false (overridden)")
	}
	if opts2.MaxRestarts == nil || *opts2.MaxRestarts != 10 {
		t.Errorf("Expected MaxRestarts to be 10 (overridden), got %v", opts2.MaxRestarts)
	}
}

func TestSetOptions(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command: "llama-server",
				Args:    []string{},
			},
			MLX: config.BackendSettings{
				Command: "mlx_lm.server",
				Args:    []string{},
			},
			VLLM: config.BackendSettings{
				Command: "vllm",
				Args:    []string{"serve"},
			},
		},
		Instances: config.InstancesConfig{
			DefaultAutoRestart:  true,
			LogsDir:             "/tmp/test",
			DefaultMaxRestarts:  3,
			DefaultRestartDelay: 5,
		},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	initialOptions := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	inst := instance.New("test-instance", globalConfig, initialOptions, mockOnStatusChange)

	// Update options
	newOptions := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/new-model.gguf",
				Port:  8081,
			},
		},
	}

	inst.SetOptions(newOptions)
	opts := inst.GetOptions()

	if opts.BackendOptions.LlamaServerOptions.Model != "/path/to/new-model.gguf" {
		t.Errorf("Expected updated model '/path/to/new-model.gguf', got %q", opts.BackendOptions.LlamaServerOptions.Model)
	}
	if inst.GetPort() != 8081 {
		t.Errorf("Expected updated port 8081, got %d", inst.GetPort())
	}

	// Check that defaults are still applied
	if opts.AutoRestart == nil || !*opts.AutoRestart {
		t.Error("Expected AutoRestart to be true (default)")
	}
}

func TestMarshalJSON(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server"},
		},
		Instances: config.InstancesConfig{LogsDir: "/tmp/test"},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	inst := instance.New("test-instance", globalConfig, options, nil)

	data, err := json.Marshal(inst)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Verify by unmarshaling and checking key fields
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if result["name"] != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %v", result["name"])
	}
	if result["status"] != "stopped" {
		t.Errorf("Expected status 'stopped', got %v", result["status"])
	}
	if result["options"] == nil {
		t.Error("Expected options to be included in JSON")
	}
}

func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"name": "test-instance",
		"status": "running",
		"options": {
			"auto_restart": false,
			"max_restarts": 5,
			"backend_type": "llama_cpp",
			"backend_options": {
				"model": "/path/to/model.gguf",
				"port": 8080
			}
		}
	}`

	var inst instance.Instance
	err := json.Unmarshal([]byte(jsonData), &inst)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if inst.Name != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %q", inst.Name)
	}
	if !inst.IsRunning() {
		t.Error("Expected status to be running")
	}

	opts := inst.GetOptions()
	if opts == nil {
		t.Fatal("Expected options to be set")
	}
	if opts.BackendOptions.BackendType != backends.BackendTypeLlamaCpp {
		t.Errorf("Expected backend_type '%s', got %s", backends.BackendTypeLlamaCpp, opts.BackendOptions.BackendType)
	}
	if opts.BackendOptions.LlamaServerOptions == nil {
		t.Fatal("Expected LlamaServerOptions to be set")
	}
	if opts.BackendOptions.LlamaServerOptions.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.BackendOptions.LlamaServerOptions.Model)
	}
	if inst.GetPort() != 8080 {
		t.Errorf("Expected port 8080, got %d", inst.GetPort())
	}
	if opts.AutoRestart == nil || *opts.AutoRestart {
		t.Error("Expected AutoRestart to be false")
	}
	if opts.MaxRestarts == nil || *opts.MaxRestarts != 5 {
		t.Errorf("Expected MaxRestarts to be 5, got %v", opts.MaxRestarts)
	}
}

func TestCreateOptionsValidation(t *testing.T) {
	tests := []struct {
		name          string
		maxRestarts   *int
		restartDelay  *int
		expectedMax   int
		expectedDelay int
	}{
		{
			name:          "valid positive values",
			maxRestarts:   testutil.IntPtr(10),
			restartDelay:  testutil.IntPtr(30),
			expectedMax:   10,
			expectedDelay: 30,
		},
		{
			name:          "zero values",
			maxRestarts:   testutil.IntPtr(0),
			restartDelay:  testutil.IntPtr(0),
			expectedMax:   0,
			expectedDelay: 0,
		},
		{
			name:          "negative values should be corrected",
			maxRestarts:   testutil.IntPtr(-5),
			restartDelay:  testutil.IntPtr(-10),
			expectedMax:   0,
			expectedDelay: 0,
		},
	}

	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command: "llama-server",
				Args:    []string{},
			},
			MLX: config.BackendSettings{
				Command: "mlx_lm.server",
				Args:    []string{},
			},
			VLLM: config.BackendSettings{
				Command: "vllm",
				Args:    []string{"serve"},
			},
		},
		Instances: config.InstancesConfig{
			LogsDir: "/tmp/test",
		},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.Options{
				MaxRestarts:  tt.maxRestarts,
				RestartDelay: tt.restartDelay,
				BackendOptions: backends.Options{
					BackendType: backends.BackendTypeLlamaCpp,
					LlamaServerOptions: &backends.LlamaServerOptions{
						Model: "/path/to/model.gguf",
					},
				},
			}

			// Mock onStatusChange function
			mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

			instance := instance.New("test", globalConfig, options, mockOnStatusChange)
			opts := instance.GetOptions()

			if opts.MaxRestarts == nil {
				t.Error("Expected MaxRestarts to be set")
			} else if *opts.MaxRestarts != tt.expectedMax {
				t.Errorf("Expected MaxRestarts %d, got %d", tt.expectedMax, *opts.MaxRestarts)
			}

			if opts.RestartDelay == nil {
				t.Error("Expected RestartDelay to be set")
			} else if *opts.RestartDelay != tt.expectedDelay {
				t.Errorf("Expected RestartDelay %d, got %d", tt.expectedDelay, *opts.RestartDelay)
			}
		})
	}
}

func TestStatusChangeCallback(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server"},
		},
		Instances: config.InstancesConfig{LogsDir: "/tmp/test"},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	var callbackOldStatus, callbackNewStatus instance.Status
	callbackCalled := false

	onStatusChange := func(oldStatus, newStatus instance.Status) {
		callbackOldStatus = oldStatus
		callbackNewStatus = newStatus
		callbackCalled = true
	}

	inst := instance.New("test", globalConfig, options, onStatusChange)

	inst.SetStatus(instance.Running)

	if !callbackCalled {
		t.Error("Expected status change callback to be called")
	}
	if callbackOldStatus != instance.Stopped {
		t.Errorf("Expected old status Stopped, got %v", callbackOldStatus)
	}
	if callbackNewStatus != instance.Running {
		t.Errorf("Expected new status Running, got %v", callbackNewStatus)
	}
}

func TestSetOptions_NodesPreserved(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server"},
		},
		Instances: config.InstancesConfig{LogsDir: "/tmp/test"},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	tests := []struct {
		name          string
		initialNodes  map[string]struct{}
		updateNodes   map[string]struct{}
		expectedNodes map[string]struct{}
	}{
		{
			name:          "nil nodes preserved as nil",
			initialNodes:  nil,
			updateNodes:   map[string]struct{}{"worker1": {}},
			expectedNodes: nil,
		},
		{
			name:          "empty nodes preserved as empty",
			initialNodes:  map[string]struct{}{},
			updateNodes:   map[string]struct{}{"worker1": {}},
			expectedNodes: map[string]struct{}{},
		},
		{
			name:          "existing nodes preserved",
			initialNodes:  map[string]struct{}{"worker1": {}, "worker2": {}},
			updateNodes:   map[string]struct{}{"worker3": {}},
			expectedNodes: map[string]struct{}{"worker1": {}, "worker2": {}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.Options{
				Nodes: tt.initialNodes,
				BackendOptions: backends.Options{
					BackendType: backends.BackendTypeLlamaCpp,
					LlamaServerOptions: &backends.LlamaServerOptions{
						Model: "/path/to/model.gguf",
					},
				},
			}

			inst := instance.New("test", globalConfig, options, nil)

			// Attempt to update nodes (should be ignored)
			updateOptions := &instance.Options{
				Nodes: tt.updateNodes,
				BackendOptions: backends.Options{
					BackendType: backends.BackendTypeLlamaCpp,
					LlamaServerOptions: &backends.LlamaServerOptions{
						Model: "/path/to/new-model.gguf",
					},
				},
			}
			inst.SetOptions(updateOptions)

			opts := inst.GetOptions()

			// Verify nodes are preserved
			if len(opts.Nodes) != len(tt.expectedNodes) {
				t.Errorf("Expected %d nodes, got %d", len(tt.expectedNodes), len(opts.Nodes))
			}
			for node := range tt.expectedNodes {
				if _, exists := opts.Nodes[node]; !exists {
					t.Errorf("Expected node %s to exist", node)
				}
			}

			// Verify other options were updated
			if opts.BackendOptions.LlamaServerOptions.Model != "/path/to/new-model.gguf" {
				t.Errorf("Expected model to be updated to '/path/to/new-model.gguf', got %q", opts.BackendOptions.LlamaServerOptions.Model)
			}
		})
	}
}

func TestProcessErrorCases(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server"},
		},
		Instances: config.InstancesConfig{LogsDir: "/tmp/test"},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	inst := instance.New("test", globalConfig, options, nil)

	// Stop when not running should return error
	err := inst.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running instance")
	}

	// Simulate running state
	inst.SetStatus(instance.Running)

	// Start when already running should return error
	err = inst.Start()
	if err == nil {
		t.Error("Expected error when starting already running instance")
	}
}

func TestRemoteInstanceOperations(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server"},
		},
		Instances: config.InstancesConfig{LogsDir: "/tmp/test"},
		Nodes: map[string]config.NodeConfig{
			"remote-node": {Address: "http://remote-node:8080"},
		},
		LocalNode: "main",
	}
	options := &instance.Options{
		Nodes: map[string]struct{}{"remote-node": {}}, // Remote instance
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	inst := instance.New("remote-test", globalConfig, options, nil)

	if !inst.IsRemote() {
		t.Error("Expected instance to be remote")
	}

	// Start should fail for remote instance
	if err := inst.Start(); err == nil {
		t.Error("Expected error when starting remote instance")
	}

	// Stop should fail for remote instance
	if err := inst.Stop(); err == nil {
		t.Error("Expected error when stopping remote instance")
	}

	// Restart should fail for remote instance
	if err := inst.Restart(); err == nil {
		t.Error("Expected error when restarting remote instance")
	}

	// GetLogs should fail for remote instance
	if _, err := inst.GetLogs(10); err == nil {
		t.Error("Expected error when getting logs for remote instance")
	}
}

func TestIdleTimeout(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server"},
		},
		Instances: config.InstancesConfig{LogsDir: "/tmp/test"},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	t.Run("not running never times out", func(t *testing.T) {
		timeout := 1
		inst := instance.New("test", globalConfig, &instance.Options{
			IdleTimeout: &timeout,
			BackendOptions: backends.Options{
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &backends.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			},
		}, nil)

		if inst.ShouldTimeout() {
			t.Error("Non-running instance should never timeout")
		}
	})

	t.Run("no timeout configured", func(t *testing.T) {
		inst := instance.New("test", globalConfig, &instance.Options{
			IdleTimeout: nil, // No timeout
			BackendOptions: backends.Options{
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &backends.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			},
		}, nil)
		inst.SetStatus(instance.Running)

		if inst.ShouldTimeout() {
			t.Error("Instance with no timeout configured should not timeout")
		}
	})

	t.Run("timeout exceeded", func(t *testing.T) {
		timeout := 1 // 1 minute
		inst := instance.New("test", globalConfig, &instance.Options{
			IdleTimeout: &timeout,
			BackendOptions: backends.Options{
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &backends.LlamaServerOptions{
					Model: "/path/to/model.gguf",
					Host:  "localhost",
					Port:  8080,
				},
			},
		}, nil)
		inst.SetStatus(instance.Running)

		// Use mock time provider
		mockTime := &mockTimeProvider{currentTime: time.Now().Unix()}
		inst.SetTimeProvider(mockTime)

		// Set last request time to now
		inst.UpdateLastRequestTime()

		// Advance time by 2 minutes (exceeds 1 minute timeout)
		mockTime.currentTime = time.Now().Add(2 * time.Minute).Unix()

		if !inst.ShouldTimeout() {
			t.Error("Instance should timeout when idle time exceeds configured timeout")
		}
	})
}

// mockTimeProvider for timeout testing
type mockTimeProvider struct {
	currentTime int64 // Unix timestamp
}

func (m *mockTimeProvider) Now() time.Time {
	return time.Unix(m.currentTime, 0)
}

func TestWritePresetIni(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command: "llama-server",
				Args:    []string{},
			},
			MLX: config.BackendSettings{
				Command: "mlx_lm.server",
				Args:    []string{},
			},
			VLLM: config.BackendSettings{
				Command: "vllm",
				Args:    []string{"serve"},
			},
		},
		Instances: config.InstancesConfig{
			LogsDir:             "/tmp/test-logs",
			InstancesDir:        "/tmp/test-instances",
			DefaultAutoRestart:  true,
			DefaultMaxRestarts:  3,
			DefaultRestartDelay: 5,
		},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	t.Run("preset_ini with content creates file", func(t *testing.T) {
		presetContent := "[model1]\nmodel = /path/to/model1.gguf\ngpu-layers = 35\n"
		options := &instance.Options{
			PresetIni: &presetContent,
			BackendOptions: backends.Options{
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &backends.LlamaServerOptions{
					Model: "/path/to/model.gguf",
					Port:  8080,
				},
			},
		}

		_ = instance.New("test-preset", globalConfig, options, mockOnStatusChange)

		// Verify preset.ini file was created
		presetPath := "/tmp/test-instances/test-preset/preset.ini"
		content, err := os.ReadFile(presetPath)
		if err != nil {
			t.Fatalf("Failed to read preset.ini file: %v", err)
		}
		if string(content) != presetContent {
			t.Errorf("Expected preset content '%s', got '%s'", presetContent, string(content))
		}
	})

	t.Run("empty preset_ini does not create file", func(t *testing.T) {
		emptyPreset := ""
		options := &instance.Options{
			PresetIni: &emptyPreset,
			BackendOptions: backends.Options{
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &backends.LlamaServerOptions{
					Model: "/path/to/model.gguf",
					Port:  8080,
				},
			},
		}

		_ = instance.New("test-empty", globalConfig, options, mockOnStatusChange)

		// Verify preset.ini file was NOT created
		presetPath := "/tmp/test-instances/test-empty/preset.ini"
		if _, err := os.Stat(presetPath); err == nil {
			t.Error("Expected preset.ini file to NOT exist when preset_ini is empty, but it does")
		}
	})
}
