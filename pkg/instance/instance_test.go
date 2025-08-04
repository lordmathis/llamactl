package instance_test

import (
	"encoding/json"
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/testutil"
	"testing"
)

func TestNewInstance(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	instance := instance.NewInstance("test-instance", globalSettings, options)

	if instance.Name != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %q", instance.Name)
	}
	if instance.Running {
		t.Error("New instance should not be running")
	}

	// Check that options were properly set with defaults applied
	opts := instance.GetOptions()
	if opts.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.Model)
	}
	if opts.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", opts.Port)
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
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	instance := instance.NewInstance("test-instance", globalSettings, options)
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

func TestNewInstance_ValidationAndDefaults(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	// Test with invalid negative values
	invalidMaxRestarts := -5
	invalidRestartDelay := -10

	options := &instance.CreateInstanceOptions{
		MaxRestarts:  &invalidMaxRestarts,
		RestartDelay: &invalidRestartDelay,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	instance := instance.NewInstance("test-instance", globalSettings, options)
	opts := instance.GetOptions()

	// Check that negative values were corrected to 0
	if opts.MaxRestarts == nil || *opts.MaxRestarts != 0 {
		t.Errorf("Expected MaxRestarts to be corrected to 0, got %v", opts.MaxRestarts)
	}
	if opts.RestartDelay == nil || *opts.RestartDelay != 0 {
		t.Errorf("Expected RestartDelay to be corrected to 0, got %v", opts.RestartDelay)
	}
}

func TestSetOptions(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	initialOptions := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, initialOptions)

	// Update options
	newOptions := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/new-model.gguf",
			Port:  8081,
		},
	}

	inst.SetOptions(newOptions)
	opts := inst.GetOptions()

	if opts.Model != "/path/to/new-model.gguf" {
		t.Errorf("Expected updated model '/path/to/new-model.gguf', got %q", opts.Model)
	}
	if opts.Port != 8081 {
		t.Errorf("Expected updated port 8081, got %d", opts.Port)
	}

	// Check that defaults are still applied
	if opts.AutoRestart == nil || !*opts.AutoRestart {
		t.Error("Expected AutoRestart to be true (default)")
	}
}

func TestSetOptions_NilOptions(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	instance := instance.NewInstance("test-instance", globalSettings, options)
	originalOptions := instance.GetOptions()

	// Try to set nil options
	instance.SetOptions(nil)

	// Options should remain unchanged
	currentOptions := instance.GetOptions()
	if currentOptions.Model != originalOptions.Model {
		t.Error("Options should not change when setting nil options")
	}
}

func TestGetProxy(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Host: "localhost",
			Port: 8080,
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)

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
	globalSettings := &config.InstancesConfig{
		LogsDir:             "/tmp/test",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	instance := instance.NewInstance("test-instance", globalSettings, options)

	data, err := json.Marshal(instance)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	// Check that JSON contains expected fields
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if result["name"] != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %v", result["name"])
	}
	if result["running"] != false {
		t.Errorf("Expected running false, got %v", result["running"])
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
	if options_map["model"] != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %v", options_map["model"])
	}
}

func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"name": "test-instance",
		"running": true,
		"options": {
			"model": "/path/to/model.gguf",
			"port": 8080,
			"auto_restart": false,
			"max_restarts": 5
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
	if !inst.Running {
		t.Error("Expected running to be true")
	}

	opts := inst.GetOptions()
	if opts == nil {
		t.Fatal("Expected options to be set")
	}
	if opts.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.Model)
	}
	if opts.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", opts.Port)
	}
	if opts.AutoRestart == nil || *opts.AutoRestart {
		t.Error("Expected AutoRestart to be false")
	}
	if opts.MaxRestarts == nil || *opts.MaxRestarts != 5 {
		t.Errorf("Expected MaxRestarts to be 5, got %v", opts.MaxRestarts)
	}
}

func TestUnmarshalJSON_PartialOptions(t *testing.T) {
	jsonData := `{
		"name": "test-instance",
		"running": false,
		"options": {
			"model": "/path/to/model.gguf"
		}
	}`

	var inst instance.Process
	err := json.Unmarshal([]byte(jsonData), &inst)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	opts := inst.GetOptions()
	if opts.Model != "/path/to/model.gguf" {
		t.Errorf("Expected model '/path/to/model.gguf', got %q", opts.Model)
	}

	// Note: Defaults are NOT applied during unmarshaling
	// They should only be applied by NewInstance or SetOptions
	if opts.AutoRestart != nil {
		t.Error("Expected AutoRestart to be nil (no defaults applied during unmarshal)")
	}
}

func TestUnmarshalJSON_NoOptions(t *testing.T) {
	jsonData := `{
		"name": "test-instance",
		"running": false
	}`

	var inst instance.Process
	err := json.Unmarshal([]byte(jsonData), &inst)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if inst.Name != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %q", inst.Name)
	}
	if inst.Running {
		t.Error("Expected running to be false")
	}

	opts := inst.GetOptions()
	if opts != nil {
		t.Error("Expected options to be nil when not provided in JSON")
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
			name:          "nil values",
			maxRestarts:   nil,
			restartDelay:  nil,
			expectedMax:   0, // Should remain nil, but we can't easily test nil in this structure
			expectedDelay: 0,
		},
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

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.CreateInstanceOptions{
				MaxRestarts:  tt.maxRestarts,
				RestartDelay: tt.restartDelay,
				LlamaServerOptions: llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			instance := instance.NewInstance("test", globalSettings, options)
			opts := instance.GetOptions()

			if tt.maxRestarts != nil {
				if opts.MaxRestarts == nil {
					t.Error("Expected MaxRestarts to be set")
				} else if *opts.MaxRestarts != tt.expectedMax {
					t.Errorf("Expected MaxRestarts %d, got %d", tt.expectedMax, *opts.MaxRestarts)
				}
			}

			if tt.restartDelay != nil {
				if opts.RestartDelay == nil {
					t.Error("Expected RestartDelay to be set")
				} else if *opts.RestartDelay != tt.expectedDelay {
					t.Errorf("Expected RestartDelay %d, got %d", tt.expectedDelay, *opts.RestartDelay)
				}
			}
		})
	}
}
