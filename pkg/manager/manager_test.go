package manager_test

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewInstanceManager(t *testing.T) {
	backendConfig := config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		LogsDir:              "/tmp/test",
		MaxInstances:         5,
		DefaultAutoRestart:   true,
		DefaultMaxRestarts:   3,
		DefaultRestartDelay:  5,
		TimeoutCheckInterval: 5,
	}

	mgr := manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")
	if mgr == nil {
		t.Fatal("NewInstanceManager returned nil")
	}

	// Test initial state
	instances, err := mgr.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("Expected empty instance list, got %d instances", len(instances))
	}
}

func TestPersistence(t *testing.T) {
	tempDir := t.TempDir()

	backendConfig := config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		InstancesDir:         tempDir,
		MaxInstances:         10,
		TimeoutCheckInterval: 5,
	}

	// Test instance persistence on creation
	manager1 := manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err := manager1.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Check that JSON file was created
	expectedPath := filepath.Join(tempDir, "test-instance.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected persistence file %s to exist", expectedPath)
	}

	// Test loading instances from disk
	manager2 := manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")
	instances, err := manager2.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("Expected 1 loaded instance, got %d", len(instances))
	}
	if instances[0].Name != "test-instance" {
		t.Errorf("Expected loaded instance name 'test-instance', got %q", instances[0].Name)
	}

	// Test port map populated from loaded instances (port conflict should be detected)
	_, err = manager2.CreateInstance("new-instance", options) // Same port
	if err == nil || !strings.Contains(err.Error(), "port") {
		t.Errorf("Expected port conflict error, got: %v", err)
	}

	// Test file deletion on instance deletion
	err = manager2.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("Expected persistence file to be deleted")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := createTestManager()
	defer mgr.Shutdown()

	// Test concurrent operations
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Concurrent instance creation
	for i := range 5 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			options := &instance.Options{
				BackendOptions: backends.Options{
					BackendType: backends.BackendTypeLlamaCpp,
					LlamaServerOptions: &backends.LlamaServerOptions{
						Model: "/path/to/model.gguf",
					},
				},
			}
			instanceName := fmt.Sprintf("concurrent-test-%d", index)
			if _, err := mgr.CreateInstance(instanceName, options); err != nil {
				errChan <- err
			}
		}(i)
	}

	// Concurrent list operations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := mgr.ListInstances(); err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Check for any errors during concurrent access
	for err := range errChan {
		t.Errorf("Concurrent access error: %v", err)
	}
}

func TestShutdown(t *testing.T) {
	mgr := createTestManager()

	// Create test instance
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}
	_, err := mgr.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Shutdown should not panic
	mgr.Shutdown()

	// Multiple shutdowns should not panic
	mgr.Shutdown()
}

// Helper function to create a test manager with standard config
func createTestManager() manager.InstanceManager {
	backendConfig := config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		LogsDir:              "/tmp/test",
		MaxInstances:         10,
		DefaultAutoRestart:   true,
		DefaultMaxRestarts:   3,
		DefaultRestartDelay:  5,
		TimeoutCheckInterval: 5,
	}
	return manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")
}

func TestAutoRestartDisabledInstanceStatus(t *testing.T) {
	tempDir := t.TempDir()

	backendConfig := config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
	}

	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		InstancesDir:         tempDir,
		MaxInstances:         10,
		TimeoutCheckInterval: 5,
	}

	// Create first manager and instance with auto-restart disabled
	manager1 := manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")

	autoRestart := false
	options := &instance.Options{
		AutoRestart: &autoRestart,
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	inst, err := manager1.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Simulate instance being in running state when persisted
	// (this would happen if the instance was running when llamactl was stopped)
	inst.SetStatus(instance.Running)

	// Shutdown first manager
	manager1.Shutdown()

	// Create second manager (simulating restart of llamactl)
	manager2 := manager.New(backendConfig, cfg, map[string]config.NodeConfig{}, "main")

	// Get the loaded instance
	loadedInst, err := manager2.GetInstance("test-instance")
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}

	// The instance should be marked as Stopped, not Running
	// because auto-restart is disabled
	if loadedInst.IsRunning() {
		t.Errorf("Expected instance with auto-restart disabled to be stopped after manager restart, but it was running")
	}

	if loadedInst.GetStatus() != instance.Stopped {
		t.Errorf("Expected instance status to be Stopped, got %v", loadedInst.GetStatus())
	}

	manager2.Shutdown()
}
