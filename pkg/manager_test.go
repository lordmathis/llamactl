package llamactl_test

import (
	"strings"
	"testing"

	llamactl "llamactl/pkg"
)

func TestNewInstanceManager(t *testing.T) {
	config := llamactl.InstancesConfig{
		PortRange:           [2]int{8000, 9000},
		LogDir:              "/tmp/test",
		MaxInstances:        5,
		LlamaExecutable:     "llama-server",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}

	manager := llamactl.NewInstanceManager(config)
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

	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}

	instance, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	if instance.Name != "test-instance" {
		t.Errorf("Expected instance name 'test-instance', got %q", instance.Name)
	}
	if instance.Running {
		t.Error("New instance should not be running")
	}
	if instance.GetOptions().Port != 8080 {
		t.Errorf("Expected port 8080, got %d", instance.GetOptions().Port)
	}
}

func TestCreateInstance_DuplicateName(t *testing.T) {
	manager := createTestManager()

	options1 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	config := llamactl.InstancesConfig{
		PortRange:    [2]int{8000, 9000},
		MaxInstances: 2, // Very low limit for testing
	}
	manager := llamactl.NewInstanceManager(config)

	options1 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options3 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	instance, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Should auto-assign a port in the range
	port := instance.GetOptions().Port
	if port < 8000 || port > 9000 {
		t.Errorf("Expected port in range 8000-9000, got %d", port)
	}
}

func TestCreateInstance_PortConflictDetection(t *testing.T) {
	manager := createTestManager()

	options1 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080, // Explicit port
		},
	}

	options2 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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

	options1 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	config := llamactl.InstancesConfig{
		PortRange:    [2]int{8000, 8001}, // Only 2 ports available
		MaxInstances: 10,                 // Higher than available ports
	}
	manager := llamactl.NewInstanceManager(config)

	options1 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options3 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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

	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	options1 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	options2 := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	for _, instance := range instances {
		names[instance.Name] = true
	}
	if !names["instance1"] || !names["instance2"] {
		t.Error("Expected both instance1 and instance2 in list")
	}
}

func TestDeleteInstance_Success(t *testing.T) {
	manager := createTestManager()

	// Create an instance
	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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
	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
		},
	}
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Update it
	newOptions := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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

	options := &llamactl.CreateInstanceOptions{
		LlamaServerOptions: llamactl.LlamaServerOptions{
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

// Helper function to create a test manager with standard config
func createTestManager() llamactl.InstanceManager {
	config := llamactl.InstancesConfig{
		PortRange:           [2]int{8000, 9000},
		LogDir:              "/tmp/test",
		MaxInstances:        10,
		LlamaExecutable:     "llama-server",
		DefaultAutoRestart:  true,
		DefaultMaxRestarts:  3,
		DefaultRestartDelay: 5,
	}
	return llamactl.NewInstanceManager(config)
}
