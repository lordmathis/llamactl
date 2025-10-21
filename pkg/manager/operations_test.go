package manager_test

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"strings"
	"testing"
)

func TestCreateInstance_Success(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	inst, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	if inst.Name != "test-instance" {
		t.Errorf("Expected instance name 'test-instance', got %q", inst.Name)
	}
	if inst.GetStatus() != instance.Stopped {
		t.Error("New instance should not be running")
	}
	if inst.GetPort() != 8080 {
		t.Errorf("Expected port 8080, got %d", inst.GetPort())
	}
}

func TestCreateInstance_DuplicateName(t *testing.T) {
	mngr := createTestManager()
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	_, err := mngr.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("First CreateInstance failed: %v", err)
	}

	// Try to create duplicate
	_, err = mngr.CreateInstance("test-instance", options)
	if err == nil {
		t.Error("Expected error for duplicate instance name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected duplicate name error, got: %v", err)
	}
}

func TestCreateInstance_MaxInstancesLimit(t *testing.T) {
	backendConfig := config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
	}
	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		MaxInstances:         1, // Very low limit for testing
		TimeoutCheckInterval: 5,
	}
	limitedManager := manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	_, err := limitedManager.CreateInstance("instance1", options)
	if err != nil {
		t.Fatalf("CreateInstance 1 failed: %v", err)
	}

	// This should fail due to max instances limit
	_, err = limitedManager.CreateInstance("instance2", options)
	if err == nil {
		t.Error("Expected error when exceeding max instances limit")
	}
	if !strings.Contains(err.Error(), "maximum number of instances") {
		t.Errorf("Expected max instances error, got: %v", err)
	}
}

func TestPort_AutoAssignment(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	inst, err := manager.CreateInstance("instance1", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	port := inst.GetPort()
	if port < 8000 || port > 9000 {
		t.Errorf("Expected port in range 8000-9000, got %d", port)
	}
}

func TestPort_ConflictDetection(t *testing.T) {
	manager := createTestManager()

	options1 := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err := manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Try to create instance with same port
	options2 := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model2.gguf",
				Port:  8080, // Same port - should conflict
			},
		},
	}

	_, err = manager.CreateInstance("instance2", options2)
	if err == nil {
		t.Error("Expected error for port conflict")
	}
	if !strings.Contains(err.Error(), "port") && !strings.Contains(err.Error(), "in use") {
		t.Errorf("Expected port conflict error, got: %v", err)
	}
}

func TestPort_ReleaseOnDeletion(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err := manager.CreateInstance("port-test", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	err = manager.DeleteInstance("port-test")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Should be able to create new instance with same port
	_, err = manager.CreateInstance("new-port-test", options)
	if err != nil {
		t.Errorf("Expected to reuse port after deletion, got error: %v", err)
	}
}

func TestGetInstance(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	created, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	retrieved, err := manager.GetInstance("test-instance")
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}
	if retrieved.Name != created.Name {
		t.Errorf("Expected name %q, got %q", created.Name, retrieved.Name)
	}
}

func TestUpdateInstance(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	newOptions := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/new-model.gguf",
				Port:  8081,
			},
		},
	}

	updated, err := manager.UpdateInstance("test-instance", newOptions)
	if err != nil {
		t.Fatalf("UpdateInstance failed: %v", err)
	}
	if updated.GetOptions().BackendOptions.LlamaServerOptions.Model != "/path/to/new-model.gguf" {
		t.Errorf("Expected model '/path/to/new-model.gguf', got %q", updated.GetOptions().BackendOptions.LlamaServerOptions.Model)
	}
}

func TestListInstances(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	instances, err := manager.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}
}

func TestDeleteInstance(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	err = manager.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	_, err = manager.GetInstance("test-instance")
	if err == nil {
		t.Error("Instance should not exist after deletion")
	}
}

func TestInstanceOperations_NonExistentInstance(t *testing.T) {
	manager := createTestManager()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}

	_, err := manager.GetInstance("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	err = manager.DeleteInstance("nonexistent")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	_, err = manager.UpdateInstance("nonexistent", options)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}
