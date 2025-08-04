package manager_test

import (
	"encoding/json"
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewInstanceManager(t *testing.T) {
	cfg := config.InstancesConfig{
		PortRange:           [2]int{8000, 9000},
		LogsDir:             "/tmp/test",
		MaxInstances:        5,
		LlamaExecutable:     "llama-server",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	manager := manager.NewInstanceManager(cfg)
	if manager == nil {
		t.Fatal("NewInstanceManager returned nil")
	}

	// Test initial state
	instances, err := manager.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("Expected empty instance list, got %d instances", len(instances))
	}
}

func TestCreateInstance_Success(t *testing.T) {
	manager := createTestManager()

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	inst, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	if inst.Name != "test-instance" {
		t.Errorf("Expected instance name 'test-instance', got %q", inst.Name)
	}
	if inst.Running {
		t.Error("New instance should not be running")
	}
	if inst.GetOptions().Port != 8080 {
		t.Errorf("Expected port 8080, got %d", inst.GetOptions().Port)
	}
}

func TestCreateInstance_DuplicateName(t *testing.T) {
	manager := createTestManager()

	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Create first instance
	_, err := manager.CreateInstance("test-instance", options1)
	if err != nil {
		t.Fatalf("First CreateInstance failed: %v", err)
	}

	// Try to create duplicate
	_, err = manager.CreateInstance("test-instance", options2)
	if err == nil {
		t.Error("Expected error for duplicate instance name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected duplicate name error, got: %v", err)
	}
}

func TestCreateInstance_MaxInstancesLimit(t *testing.T) {
	// Create manager with low max instances limit
	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		MaxInstances: 2, // Very low limit for testing
	}
	manager := manager.NewInstanceManager(cfg)

	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options3 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Create instances up to the limit
	_, err := manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance 1 failed: %v", err)
	}

	_, err = manager.CreateInstance("instance2", options2)
	if err != nil {
		t.Fatalf("CreateInstance 2 failed: %v", err)
	}

	// This should fail due to max instances limit
	_, err = manager.CreateInstance("instance3", options3)
	if err == nil {
		t.Error("Expected error when exceeding max instances limit")
	}
	if !strings.Contains(err.Error(), "maximum number of instances") && !strings.Contains(err.Error(), "limit") {
		t.Errorf("Expected max instances error, got: %v", err)
	}
}

func TestCreateInstance_PortAssignment(t *testing.T) {
	manager := createTestManager()

	// Create instance without specifying port
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Should auto-assign a port in the range
	port := inst.GetOptions().Port
	if port < 8000 || port > 9000 {
		t.Errorf("Expected port in range 8000-9000, got %d", port)
	}
}

func TestCreateInstance_PortConflictDetection(t *testing.T) {
	manager := createTestManager()

	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080, // Explicit port
		},
	}

	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model2.gguf",
			Port:  8080, // Same port - should conflict
		},
	}

	// Create first instance
	_, err := manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance 1 failed: %v", err)
	}

	// Try to create second instance with same port
	_, err = manager.CreateInstance("instance2", options2)
	if err == nil {
		t.Error("Expected error for port conflict")
	}
	if !strings.Contains(err.Error(), "port") && !strings.Contains(err.Error(), "conflict") && !strings.Contains(err.Error(), "in use") {
		t.Errorf("Expected port conflict error, got: %v", err)
	}
}

func TestCreateInstance_MultiplePortAssignment(t *testing.T) {
	manager := createTestManager()

	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Create multiple instances and verify they get different ports
	instance1, err := manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance 1 failed: %v", err)
	}

	instance2, err := manager.CreateInstance("instance2", options2)
	if err != nil {
		t.Fatalf("CreateInstance 2 failed: %v", err)
	}

	port1 := instance1.GetOptions().Port
	port2 := instance2.GetOptions().Port

	if port1 == port2 {
		t.Errorf("Expected different ports, both got %d", port1)
	}
}

func TestCreateInstance_PortExhaustion(t *testing.T) {
	// Create manager with very small port range
	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 8001}, // Only 2 ports available
		MaxInstances: 10,                 // Higher than available ports
	}
	manager := manager.NewInstanceManager(cfg)

	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options3 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Create instances to exhaust all ports
	_, err := manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance 1 failed: %v", err)
	}

	_, err = manager.CreateInstance("instance2", options2)
	if err != nil {
		t.Fatalf("CreateInstance 2 failed: %v", err)
	}

	// This should fail due to port exhaustion
	_, err = manager.CreateInstance("instance3", options3)
	if err == nil {
		t.Error("Expected error when ports are exhausted")
	}
	if !strings.Contains(err.Error(), "port") && !strings.Contains(err.Error(), "available") {
		t.Errorf("Expected port exhaustion error, got: %v", err)
	}
}

func TestDeleteInstance_PortRelease(t *testing.T) {
	manager := createTestManager()

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	// Create instance with specific port
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Delete the instance
	err = manager.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Should be able to create new instance with same port
	_, err = manager.CreateInstance("new-instance", options)
	if err != nil {
		t.Errorf("Expected to reuse port after deletion, got error: %v", err)
	}
}

func TestGetInstance_Success(t *testing.T) {
	manager := createTestManager()

	// Create an instance first
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}
	created, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Retrieve it
	retrieved, err := manager.GetInstance("test-instance")
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}

	if retrieved.Name != created.Name {
		t.Errorf("Expected name %q, got %q", created.Name, retrieved.Name)
	}
}

func TestGetInstance_NotFound(t *testing.T) {
	manager := createTestManager()

	_, err := manager.GetInstance("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent instance")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestListInstances(t *testing.T) {
	manager := createTestManager()

	// Initially empty
	instances, err := manager.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(instances))
	}

	// Create some instances
	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	_, err = manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance 1 failed: %v", err)
	}

	_, err = manager.CreateInstance("instance2", options2)
	if err != nil {
		t.Fatalf("CreateInstance 2 failed: %v", err)
	}

	// List should return both
	instances, err = manager.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}

	// Check names are present
	names := make(map[string]bool)
	for _, inst := range instances {
		names[inst.Name] = true
	}
	if !names["instance1"] || !names["instance2"] {
		t.Error("Expected both instance1 and instance2 in list")
	}
}

func TestDeleteInstance_Success(t *testing.T) {
	manager := createTestManager()

	// Create an instance
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Delete it
	err = manager.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Should no longer exist
	_, err = manager.GetInstance("test-instance")
	if err == nil {
		t.Error("Instance should not exist after deletion")
	}
}

func TestDeleteInstance_NotFound(t *testing.T) {
	manager := createTestManager()

	err := manager.DeleteInstance("nonexistent")
	if err == nil {
		t.Error("Expected error for deleting nonexistent instance")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestUpdateInstance_Success(t *testing.T) {
	manager := createTestManager()

	// Create an instance
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Update it
	newOptions := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/new-model.gguf",
			Port:  8081,
		},
	}

	updated, err := manager.UpdateInstance("test-instance", newOptions)
	if err != nil {
		t.Fatalf("UpdateInstance failed: %v", err)
	}

	if updated.GetOptions().Model != "/path/to/new-model.gguf" {
		t.Errorf("Expected model '/path/to/new-model.gguf', got %q", updated.GetOptions().Model)
	}
	if updated.GetOptions().Port != 8081 {
		t.Errorf("Expected port 8081, got %d", updated.GetOptions().Port)
	}
}

func TestUpdateInstance_NotFound(t *testing.T) {
	manager := createTestManager()

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	_, err := manager.UpdateInstance("nonexistent", options)
	if err == nil {
		t.Error("Expected error for updating nonexistent instance")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestPersistence_InstancePersistedOnCreation(t *testing.T) {
	// Create temporary directory for persistence
	tempDir := t.TempDir()

	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		InstancesDir: tempDir,
		MaxInstances: 10,
	}
	manager := manager.NewInstanceManager(cfg)

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	// Create instance
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Check that JSON file was created
	expectedPath := filepath.Join(tempDir, "test-instance.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected persistence file %s to exist", expectedPath)
	}

	// Verify file contains correct data
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read persistence file: %v", err)
	}

	var persistedInstance map[string]interface{}
	if err := json.Unmarshal(data, &persistedInstance); err != nil {
		t.Fatalf("Failed to unmarshal persisted data: %v", err)
	}

	if persistedInstance["name"] != "test-instance" {
		t.Errorf("Expected name 'test-instance', got %v", persistedInstance["name"])
	}
}

func TestPersistence_InstancePersistedOnUpdate(t *testing.T) {
	tempDir := t.TempDir()

	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		InstancesDir: tempDir,
		MaxInstances: 10,
	}
	manager := manager.NewInstanceManager(cfg)

	// Create instance
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Update instance
	newOptions := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/new-model.gguf",
			Port:  8081,
		},
	}
	_, err = manager.UpdateInstance("test-instance", newOptions)
	if err != nil {
		t.Fatalf("UpdateInstance failed: %v", err)
	}

	// Verify persistence file was updated
	expectedPath := filepath.Join(tempDir, "test-instance.json")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read persistence file: %v", err)
	}

	var persistedInstance map[string]interface{}
	if err := json.Unmarshal(data, &persistedInstance); err != nil {
		t.Fatalf("Failed to unmarshal persisted data: %v", err)
	}

	// Check that the options were updated
	options_data, ok := persistedInstance["options"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected options to be present in persisted data")
	}

	if options_data["model"] != "/path/to/new-model.gguf" {
		t.Errorf("Expected updated model '/path/to/new-model.gguf', got %v", options_data["model"])
	}
}

func TestPersistence_InstanceFileDeletedOnDeletion(t *testing.T) {
	tempDir := t.TempDir()

	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		InstancesDir: tempDir,
		MaxInstances: 10,
	}
	manager := manager.NewInstanceManager(cfg)

	// Create instance
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "test-instance.json")

	// Verify file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatal("Expected persistence file to exist before deletion")
	}

	// Delete instance
	err = manager.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("Expected persistence file to be deleted")
	}
}

func TestPersistence_InstancesLoadedFromDisk(t *testing.T) {
	tempDir := t.TempDir()

	// Create JSON files manually (simulating previous run)
	instance1JSON := `{
		"name": "instance1",
		"running": false,
		"options": {
			"model": "/path/to/model1.gguf",
			"port": 8080
		}
	}`

	instance2JSON := `{
		"name": "instance2", 
		"running": false,
		"options": {
			"model": "/path/to/model2.gguf",
			"port": 8081
		}
	}`

	// Write JSON files
	err := os.WriteFile(filepath.Join(tempDir, "instance1.json"), []byte(instance1JSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "instance2.json"), []byte(instance2JSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	// Create manager - should load instances from disk
	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		InstancesDir: tempDir,
		MaxInstances: 10,
	}
	manager := manager.NewInstanceManager(cfg)

	// Verify instances were loaded
	instances, err := manager.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}

	if len(instances) != 2 {
		t.Fatalf("Expected 2 loaded instances, got %d", len(instances))
	}

	// Check instances by name
	instancesByName := make(map[string]*instance.Process)
	for _, inst := range instances {
		instancesByName[inst.Name] = inst
	}

	instance1, exists := instancesByName["instance1"]
	if !exists {
		t.Error("Expected instance1 to be loaded")
	} else {
		if instance1.GetOptions().Model != "/path/to/model1.gguf" {
			t.Errorf("Expected model '/path/to/model1.gguf', got %q", instance1.GetOptions().Model)
		}
		if instance1.GetOptions().Port != 8080 {
			t.Errorf("Expected port 8080, got %d", instance1.GetOptions().Port)
		}
	}

	instance2, exists := instancesByName["instance2"]
	if !exists {
		t.Error("Expected instance2 to be loaded")
	} else {
		if instance2.GetOptions().Model != "/path/to/model2.gguf" {
			t.Errorf("Expected model '/path/to/model2.gguf', got %q", instance2.GetOptions().Model)
		}
		if instance2.GetOptions().Port != 8081 {
			t.Errorf("Expected port 8081, got %d", instance2.GetOptions().Port)
		}
	}
}

func TestPersistence_PortMapPopulatedFromLoadedInstances(t *testing.T) {
	tempDir := t.TempDir()

	// Create JSON file with specific port
	instanceJSON := `{
		"name": "test-instance",
		"running": false,
		"options": {
			"model": "/path/to/model.gguf",
			"port": 8080
		}
	}`

	err := os.WriteFile(filepath.Join(tempDir, "test-instance.json"), []byte(instanceJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test JSON file: %v", err)
	}

	// Create manager - should load instance and register port
	cfg := config.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		InstancesDir: tempDir,
		MaxInstances: 10,
	}
	manager := manager.NewInstanceManager(cfg)

	// Try to create new instance with same port - should fail due to conflict
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model2.gguf",
			Port:  8080, // Same port as loaded instance
		},
	}

	_, err = manager.CreateInstance("new-instance", options)
	if err == nil {
		t.Error("Expected error for port conflict with loaded instance")
	}
	if !strings.Contains(err.Error(), "port") || !strings.Contains(err.Error(), "in use") {
		t.Errorf("Expected port conflict error, got: %v", err)
	}
}

func TestPersistence_CompleteInstanceDataRoundTrip(t *testing.T) {
	tempDir := t.TempDir()

	cfg := config.InstancesConfig{
		PortRange:           [2]int{8000, 9000},
		InstancesDir:        tempDir,
		MaxInstances:        10,
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	// Create first manager and instance with comprehensive options
	manager1 := manager.NewInstanceManager(cfg)

	autoRestart := false
	maxRestarts := 10
	restartDelay := 30

	originalOptions := &instance.CreateInstanceOptions{
		AutoRestart:  &autoRestart,
		MaxRestarts:  &maxRestarts,
		RestartDelay: &restartDelay,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model:       "/path/to/model.gguf",
			Port:        8080,
			Host:        "localhost",
			CtxSize:     4096,
			GPULayers:   32,
			Temperature: 0.7,
			TopK:        40,
			TopP:        0.9,
			Verbose:     true,
			FlashAttn:   false,
			Lora:        []string{"adapter1.bin", "adapter2.bin"},
			HFRepo:      "microsoft/DialoGPT-medium",
		},
	}

	originalInstance, err := manager1.CreateInstance("roundtrip-test", originalOptions)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Create second manager (simulating restart) - should load the instance
	manager2 := manager.NewInstanceManager(cfg)

	loadedInstance, err := manager2.GetInstance("roundtrip-test")
	if err != nil {
		t.Fatalf("GetInstance failed after reload: %v", err)
	}

	// Compare all data
	if loadedInstance.Name != originalInstance.Name {
		t.Errorf("Name mismatch: original=%q, loaded=%q", originalInstance.Name, loadedInstance.Name)
	}

	originalOpts := originalInstance.GetOptions()
	loadedOpts := loadedInstance.GetOptions()

	// Compare restart options
	if *loadedOpts.AutoRestart != *originalOpts.AutoRestart {
		t.Errorf("AutoRestart mismatch: original=%v, loaded=%v", *originalOpts.AutoRestart, *loadedOpts.AutoRestart)
	}
	if *loadedOpts.MaxRestarts != *originalOpts.MaxRestarts {
		t.Errorf("MaxRestarts mismatch: original=%v, loaded=%v", *originalOpts.MaxRestarts, *loadedOpts.MaxRestarts)
	}
	if *loadedOpts.RestartDelay != *originalOpts.RestartDelay {
		t.Errorf("RestartDelay mismatch: original=%v, loaded=%v", *originalOpts.RestartDelay, *loadedOpts.RestartDelay)
	}

	// Compare llama server options
	if loadedOpts.Model != originalOpts.Model {
		t.Errorf("Model mismatch: original=%q, loaded=%q", originalOpts.Model, loadedOpts.Model)
	}
	if loadedOpts.Port != originalOpts.Port {
		t.Errorf("Port mismatch: original=%d, loaded=%d", originalOpts.Port, loadedOpts.Port)
	}
	if loadedOpts.Host != originalOpts.Host {
		t.Errorf("Host mismatch: original=%q, loaded=%q", originalOpts.Host, loadedOpts.Host)
	}
	if loadedOpts.CtxSize != originalOpts.CtxSize {
		t.Errorf("CtxSize mismatch: original=%d, loaded=%d", originalOpts.CtxSize, loadedOpts.CtxSize)
	}
	if loadedOpts.GPULayers != originalOpts.GPULayers {
		t.Errorf("GPULayers mismatch: original=%d, loaded=%d", originalOpts.GPULayers, loadedOpts.GPULayers)
	}
	if loadedOpts.Temperature != originalOpts.Temperature {
		t.Errorf("Temperature mismatch: original=%f, loaded=%f", originalOpts.Temperature, loadedOpts.Temperature)
	}
	if loadedOpts.TopK != originalOpts.TopK {
		t.Errorf("TopK mismatch: original=%d, loaded=%d", originalOpts.TopK, loadedOpts.TopK)
	}
	if loadedOpts.TopP != originalOpts.TopP {
		t.Errorf("TopP mismatch: original=%f, loaded=%f", originalOpts.TopP, loadedOpts.TopP)
	}
	if loadedOpts.Verbose != originalOpts.Verbose {
		t.Errorf("Verbose mismatch: original=%v, loaded=%v", originalOpts.Verbose, loadedOpts.Verbose)
	}
	if loadedOpts.FlashAttn != originalOpts.FlashAttn {
		t.Errorf("FlashAttn mismatch: original=%v, loaded=%v", originalOpts.FlashAttn, loadedOpts.FlashAttn)
	}
	if loadedOpts.HFRepo != originalOpts.HFRepo {
		t.Errorf("HFRepo mismatch: original=%q, loaded=%q", originalOpts.HFRepo, loadedOpts.HFRepo)
	}

	// Compare slice fields
	if !reflect.DeepEqual(loadedOpts.Lora, originalOpts.Lora) {
		t.Errorf("Lora mismatch: original=%v, loaded=%v", originalOpts.Lora, loadedOpts.Lora)
	}

	// Verify created timestamp is preserved
	if loadedInstance.Created != originalInstance.Created {
		t.Errorf("Created timestamp mismatch: original=%d, loaded=%d", originalInstance.Created, loadedInstance.Created)
	}
}

// Helper function to create a test manager with standard config
func createTestManager() manager.InstanceManager {
	cfg := config.InstancesConfig{
		PortRange:           [2]int{8000, 9000},
		LogsDir:             "/tmp/test",
		MaxInstances:        10,
		LlamaExecutable:     "llama-server",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}
	return manager.NewInstanceManager(cfg)
}
