package instance_test

import (
	"encoding/json"
	"llamactl/pkg/backends"
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/testutil"
	"testing"
)

func TestNewInstance(t *testing.T) {
	backendConfig := &config.BackendConfig{
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
	}

	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	options := &instance.CreateInstanceOptions{
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.InstanceStatus) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, "main", mockOnStatusChange)

	if inst.Name != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %q", inst.Name)
	}
	if inst.IsRunning() {
		t.Error("New instance should not be running")
	}

	// Check that options were properly set with defaults applied
	opts := inst.GetOptions()
	if opts.LlamaServerOptions.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.LlamaServerOptions.Model)
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
}

func TestNewInstance_WithRestartOptions(t *testing.T) {
	backendConfig := &config.BackendConfig{
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
	}

	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	// Override some defaults
	autoRestart := false
	maxRestarts := 10
	restartDelay := 15

	options := &instance.CreateInstanceOptions{
		AutoRestart:  &autoRestart,
		MaxRestarts:  &maxRestarts,
		RestartDelay: &restartDelay,
		BackendType:  backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.InstanceStatus) {}

	instance := instance.NewInstance("test-instance", backendConfig, globalSettings, options, "main", mockOnStatusChange)
	opts := instance.GetOptions()

	// Check that explicit values override defaults
	if opts.AutoRestart == nil || *opts.AutoRestart {
		t.Error("Expected AutoRestart to be false (overridden)")
	}
	if opts.MaxRestarts == nil || *opts.MaxRestarts != 10 {
		t.Errorf("Expected MaxRestarts to be 10 (overridden), got %v", opts.MaxRestarts)
	}
	if opts.RestartDelay == nil || *opts.RestartDelay != 15 {
		t.Errorf("Expected RestartDelay to be 15 (overridden), got %v", opts.RestartDelay)
	}
}

func TestSetOptions(t *testing.T) {
	backendConfig := &config.BackendConfig{
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
	}

	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	initialOptions := &instance.CreateInstanceOptions{
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.InstanceStatus) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, initialOptions, "main", mockOnStatusChange)

	// Update options
	newOptions := &instance.CreateInstanceOptions{
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/new-model.gguf",
			Port:  8081,
		},
	}

	inst.SetOptions(newOptions)
	opts := inst.GetOptions()

	if opts.LlamaServerOptions.Model != "/path/to/new-model.gguf" {
		t.Errorf("Expected updated model '/path/to/new-model.gguf', got %q", opts.LlamaServerOptions.Model)
	}
	if inst.GetPort() != 8081 {
		t.Errorf("Expected updated port 8081, got %d", inst.GetPort())
	}

	// Check that defaults are still applied
	if opts.AutoRestart == nil || !*opts.AutoRestart {
		t.Error("Expected AutoRestart to be true (default)")
	}
}

func TestGetProxy(t *testing.T) {
	backendConfig := &config.BackendConfig{
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
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	options := &instance.CreateInstanceOptions{
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Host: "localhost",
			Port: 8080,
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.InstanceStatus) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, "main", mockOnStatusChange)

	// Get proxy for the first time
	proxy1, err := inst.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}
	if proxy1 == nil {
		t.Error("Expected proxy to be created")
	}

	// Get proxy again - should return cached version
	proxy2, err := inst.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}
	if proxy1 != proxy2 {
		t.Error("Expected cached proxy to be returned")
	}
}

func TestMarshalJSON(t *testing.T) {
	backendConfig := &config.BackendConfig{
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
	}

	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	options := &instance.CreateInstanceOptions{
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.InstanceStatus) {}

	instance := instance.NewInstance("test-instance", backendConfig, globalSettings, options, "main", mockOnStatusChange)

	data, err := json.Marshal(instance)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Check that JSON contains expected fields
	var result map[string]any
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if result["name"] != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %v", result["name"])
	}
	if result["status"] != "stopped" {
		t.Errorf("Expected status 'stopped', got %v", result["status"])
	}

	// Check that options are included
	options_data, ok := result["options"]
	if !ok {
		t.Error("Expected options to be included in JSON")
	}
	options_map, ok := options_data.(map[string]interface{})
	if !ok {
		t.Error("Expected options to be a map")
	}

	// Check backend type
	if options_map["backend_type"] != string(backends.BackendTypeLlamaCpp) {
		t.Errorf("Expected backend_type '%s', got %v", backends.BackendTypeLlamaCpp, options_map["backend_type"])
	}

	// Check backend options
	backend_options_data, ok := options_map["backend_options"]
	if !ok {
		t.Error("Expected backend_options to be included in JSON")
	}
	backend_options_map, ok := backend_options_data.(map[string]any)
	if !ok {
		t.Error("Expected backend_options to be a map")
	}
	if backend_options_map["model"] != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %v", backend_options_map["model"])
	}
	if backend_options_map["port"] != float64(8080) {
		t.Errorf("Expected port 8080, got %v", backend_options_map["port"])
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

	var inst instance.Process
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
	if opts.BackendType != backends.BackendTypeLlamaCpp {
		t.Errorf("Expected backend_type '%s', got %s", backends.BackendTypeLlamaCpp, opts.BackendType)
	}
	if opts.LlamaServerOptions == nil {
		t.Fatal("Expected LlamaServerOptions to be set")
	}
	if opts.LlamaServerOptions.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.LlamaServerOptions.Model)
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

func TestCreateInstanceOptionsValidation(t *testing.T) {
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

	backendConfig := &config.BackendConfig{
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
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.CreateInstanceOptions{
				MaxRestarts:  tt.maxRestarts,
				RestartDelay: tt.restartDelay,
				BackendType:  backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			// Mock onStatusChange function
			mockOnStatusChange := func(oldStatus, newStatus instance.InstanceStatus) {}

			instance := instance.NewInstance("test", backendConfig, globalSettings, options, "main", mockOnStatusChange)
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
