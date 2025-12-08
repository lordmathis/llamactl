package manager_test

import (
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"sync"
	"testing"
	"time"
)

func TestManager_PersistsAndLoadsInstances(t *testing.T) {
	tempDir := t.TempDir()
	appConfig := createTestAppConfig(tempDir)
	// Use file-based database for this test since we need to persist across connections
	appConfig.Database.Path = tempDir + "/test.db"

	// Create instance and check database was created
	db1, err := database.Open(&database.Config{
		Path:               appConfig.Database.Path,
		MaxOpenConnections: appConfig.Database.MaxOpenConnections,
		MaxIdleConnections: appConfig.Database.MaxIdleConnections,
		ConnMaxLifetime:    appConfig.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := database.RunMigrations(db1); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	manager1 := manager.New(appConfig, db1)
	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err = manager1.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Shutdown first manager to close database connection
	manager1.Shutdown()

	// Load instances from database
	db2, err := database.Open(&database.Config{
		Path:               appConfig.Database.Path,
		MaxOpenConnections: appConfig.Database.MaxOpenConnections,
		MaxIdleConnections: appConfig.Database.MaxIdleConnections,
		ConnMaxLifetime:    appConfig.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := database.RunMigrations(db2); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	manager2 := manager.New(appConfig, db2)
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

	manager2.Shutdown()
}

func TestDeleteInstance_RemovesFromDatabase(t *testing.T) {
	tempDir := t.TempDir()
	appConfig := createTestAppConfig(tempDir)

	db, err := database.Open(&database.Config{
		Path:               appConfig.Database.Path,
		MaxOpenConnections: appConfig.Database.MaxOpenConnections,
		MaxIdleConnections: appConfig.Database.MaxIdleConnections,
		ConnMaxLifetime:    appConfig.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	mgr := manager.New(appConfig, db)
	defer mgr.Shutdown()

	options := &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	_, err = mgr.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Verify instance exists
	instances, err := mgr.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(instances))
	}

	// Delete instance
	err = mgr.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Verify instance was deleted from database
	instances, err = mgr.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 0 {
		t.Errorf("Expected 0 instances after deletion, got %d", len(instances))
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
			PortRange:           [2]int{8000, 9000},
			InstancesDir:        instancesDir,
			MaxInstances:        10,
			MaxRunningInstances: 10,
			DefaultAutoRestart:  true,
			DefaultMaxRestarts:  3,
			Logging: config.LoggingConfig{
				LogsDir: instancesDir,
			},
			DefaultRestartDelay:  5,
			TimeoutCheckInterval: 5,
		},
		Database: config.DatabaseConfig{
			Path:               ":memory:",
			MaxOpenConnections: 25,
			MaxIdleConnections: 5,
			ConnMaxLifetime:    5 * time.Minute,
		},
		LocalNode: "main",
		Nodes:     map[string]config.NodeConfig{},
	}
}

func createTestManager(t *testing.T) manager.InstanceManager {
	tempDir := t.TempDir()
	appConfig := createTestAppConfig(tempDir)
	db, err := database.Open(&database.Config{
		Path:               appConfig.Database.Path,
		MaxOpenConnections: appConfig.Database.MaxOpenConnections,
		MaxIdleConnections: appConfig.Database.MaxIdleConnections,
		ConnMaxLifetime:    appConfig.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	return manager.New(appConfig, db)
}
