package manager_test

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestManager_PersistsAndLoadsInstances(t *testing.T) {
	tempDir := t.TempDir()
	appConfig := createTestAppConfig(tempDir)

	// Create instance and check file was created
	manager1 := manager.New(appConfig)
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

	expectedPath := filepath.Join(tempDir, "test-instance.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected persistence file %s to exist", expectedPath)
	}

	// Load instances from disk
	manager2 := manager.New(appConfig)
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
}

func TestDeleteInstance_RemovesPersistenceFile(t *testing.T) {
	tempDir := t.TempDir()
	appConfig := createTestAppConfig(tempDir)

	mgr := manager.New(appConfig)
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err := mgr.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	expectedPath := filepath.Join(tempDir, "test-instance.json")

	err = mgr.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Error("Expected persistence file to be deleted")
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := createTestManager(t)
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
	for range 3 {
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

// Helper functions for test configuration
func createTestAppConfig(instancesDir string) *config.AppConfig {
	// Use 'sh -c "sleep 999999"' as a test command instead of 'llama-server'
	// The shell ignores all additional arguments passed after the command
	return &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command: "sh",
				Args:    []string{"-c", "sleep 999999"},
			},
			MLX: config.BackendSettings{
				Command: "sh",
				Args:    []string{"-c", "sleep 999999"},
			},
		},
		Instances: config.InstancesConfig{
			PortRange:            [2]int{8000, 9000},
			InstancesDir:         instancesDir,
			LogsDir:              instancesDir,
			MaxInstances:         10,
			MaxRunningInstances:  10,
			DefaultAutoRestart:   true,
			DefaultMaxRestarts:   3,
			DefaultRestartDelay:  5,
			TimeoutCheckInterval: 5,
		},
		LocalNode: "main",
		Nodes:     map[string]config.NodeConfig{},
	}
}

func createTestManager(t *testing.T) manager.InstanceManager {
	tempDir := t.TempDir()
	appConfig := createTestAppConfig(tempDir)
	return manager.New(appConfig)
}
