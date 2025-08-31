package manager_test

import (
	"fmt"
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewInstanceManager(t *testing.T) {
	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		LogsDir:              "/tmp/test",
		MaxInstances:         5,
		LlamaExecutable:      "llama-server",
		DefaultAutoRestart:   true,
		DefaultMaxRestarts:   3,
		DefaultRestartDelay:  5,
		TimeoutCheckInterval: 5,
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
	if inst.GetStatus() != instance.Stopped {
		t.Error("New instance should not be running")
	}
	if inst.GetOptions().Port != 8080 {
		t.Errorf("Expected port 8080, got %d", inst.GetOptions().Port)
	}
}

func TestCreateInstance_ValidationAndLimits(t *testing.T) {
	// Test duplicate names
	mngr := createTestManager()
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
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

	// Test max instances limit
	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		MaxInstances:         1, // Very low limit for testing
		TimeoutCheckInterval: 5,
	}
	limitedManager := manager.NewInstanceManager(cfg)

	_, err = limitedManager.CreateInstance("instance1", options)
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

func TestPortManagement(t *testing.T) {
	manager := createTestManager()

	// Test auto port assignment
	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst1, err := manager.CreateInstance("instance1", options1)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	port1 := inst1.GetOptions().Port
	if port1 < 8000 || port1 > 9000 {
		t.Errorf("Expected port in range 8000-9000, got %d", port1)
	}

	// Test port conflict detection
	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model2.gguf",
			Port:  port1, // Same port - should conflict
		},
	}

	_, err = manager.CreateInstance("instance2", options2)
	if err == nil {
		t.Error("Expected error for port conflict")
	}
	if !strings.Contains(err.Error(), "port") && !strings.Contains(err.Error(), "in use") {
		t.Errorf("Expected port conflict error, got: %v", err)
	}

	// Test port release on deletion
	specificPort := 8080
	options3 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  specificPort,
		},
	}

	_, err = manager.CreateInstance("port-test", options3)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	err = manager.DeleteInstance("port-test")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	// Should be able to create new instance with same port
	_, err = manager.CreateInstance("new-port-test", options3)
	if err != nil {
		t.Errorf("Expected to reuse port after deletion, got error: %v", err)
	}
}

func TestInstanceOperations(t *testing.T) {
	manager := createTestManager()

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Create instance
	created, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Get instance
	retrieved, err := manager.GetInstance("test-instance")
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}
	if retrieved.Name != created.Name {
		t.Errorf("Expected name %q, got %q", created.Name, retrieved.Name)
	}

	// Update instance
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

	// List instances
	instances, err := manager.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}

	// Delete instance
	err = manager.DeleteInstance("test-instance")
	if err != nil {
		t.Fatalf("DeleteInstance failed: %v", err)
	}

	_, err = manager.GetInstance("test-instance")
	if err == nil {
		t.Error("Instance should not exist after deletion")
	}

	// Test operations on non-existent instances
	_, err = manager.GetInstance("nonexistent")
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

func TestPersistence(t *testing.T) {
	tempDir := t.TempDir()

	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		InstancesDir:         tempDir,
		MaxInstances:         10,
		TimeoutCheckInterval: 5,
	}

	// Test instance persistence on creation
	manager1 := manager.NewInstanceManager(cfg)
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
			Port:  8080,
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
	manager2 := manager.NewInstanceManager(cfg)
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

func TestTimeoutFunctionality(t *testing.T) {
	// Test timeout checker initialization
	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		TimeoutCheckInterval: 10,
		MaxInstances:         5,
	}

	manager := manager.NewInstanceManager(cfg)
	if manager == nil {
		t.Fatal("Manager should be initialized with timeout checker")
	}
	manager.Shutdown() // Clean up

	// Test timeout configuration and logic without starting the actual process
	testManager := createTestManager()
	defer testManager.Shutdown()

	idleTimeout := 1 // 1 minute
	options := &instance.CreateInstanceOptions{
		IdleTimeout: &idleTimeout,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst, err := testManager.CreateInstance("timeout-test", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Test timeout configuration is properly set
	if inst.GetOptions().IdleTimeout == nil {
		t.Fatal("Instance should have idle timeout configured")
	}
	if *inst.GetOptions().IdleTimeout != 1 {
		t.Errorf("Expected idle timeout 1 minute, got %d", *inst.GetOptions().IdleTimeout)
	}

	// Test timeout logic without actually starting the process
	// Create a mock time provider to simulate timeout
	mockTime := NewMockTimeProvider(time.Now())
	inst.SetTimeProvider(mockTime)

	// Set instance to running state so timeout logic can work
	inst.SetStatus(instance.Running)

	// Simulate instance being "running" for timeout check (without actual process)
	// We'll test the ShouldTimeout logic directly
	inst.UpdateLastRequestTime()

	// Initially should not timeout (just updated)
	if inst.ShouldTimeout() {
		t.Error("Instance should not timeout immediately after request")
	}

	// Advance time to trigger timeout
	mockTime.SetTime(time.Now().Add(2 * time.Minute))

	// Now it should timeout
	if !inst.ShouldTimeout() {
		t.Error("Instance should timeout after idle period")
	}

	// Reset running state to avoid shutdown issues
	inst.SetStatus(instance.Stopped)

	// Test that instance without timeout doesn't timeout
	noTimeoutOptions := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
		// No IdleTimeout set
	}

	noTimeoutInst, err := testManager.CreateInstance("no-timeout-test", noTimeoutOptions)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	noTimeoutInst.SetTimeProvider(mockTime)
	noTimeoutInst.SetStatus(instance.Running) // Set to running for timeout check
	noTimeoutInst.UpdateLastRequestTime()

	// Even with time advanced, should not timeout
	if noTimeoutInst.ShouldTimeout() {
		t.Error("Instance without timeout configuration should never timeout")
	}

	// Reset running state to avoid shutdown issues
	noTimeoutInst.SetStatus(instance.Stopped)
}

func TestConcurrentAccess(t *testing.T) {
	manager := createTestManager()
	defer manager.Shutdown()

	// Test concurrent operations
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Concurrent instance creation
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			options := &instance.CreateInstanceOptions{
				LlamaServerOptions: llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}
			instanceName := fmt.Sprintf("concurrent-test-%d", index)
			if _, err := manager.CreateInstance(instanceName, options); err != nil {
				errChan <- err
			}
		}(i)
	}

	// Concurrent list operations
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := manager.ListInstances(); err != nil {
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
	manager := createTestManager()

	// Create test instance
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}
	_, err := manager.CreateInstance("test-instance", options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Shutdown should not panic
	manager.Shutdown()

	// Multiple shutdowns should not panic
	manager.Shutdown()
}

// Helper function to create a test manager with standard config
func createTestManager() manager.InstanceManager {
	cfg := config.InstancesConfig{
		PortRange:            [2]int{8000, 9000},
		LogsDir:              "/tmp/test",
		MaxInstances:         10,
		LlamaExecutable:      "llama-server",
		DefaultAutoRestart:   true,
		DefaultMaxRestarts:   3,
		DefaultRestartDelay:  5,
		TimeoutCheckInterval: 5,
	}
	return manager.NewInstanceManager(cfg)
}

// Helper for timeout tests
type MockTimeProvider struct {
	currentTime time.Time
	mu          sync.RWMutex
}

func NewMockTimeProvider(t time.Time) *MockTimeProvider {
	return &MockTimeProvider{currentTime: t}
}

func (m *MockTimeProvider) Now() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentTime
}

func (m *MockTimeProvider) SetTime(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentTime = t
}

func TestEvictLRUInstance_Success(t *testing.T) {
	manager := createTestManager()
	// Don't defer manager.Shutdown() - we'll handle cleanup manually

	// Create 3 instances with idle timeout enabled (value doesn't matter for LRU logic)
	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model1.gguf",
		},
		IdleTimeout: func() *int { timeout := 1; return &timeout }(), // Any value > 0
	}
	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model2.gguf",
		},
		IdleTimeout: func() *int { timeout := 1; return &timeout }(), // Any value > 0
	}
	options3 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model3.gguf",
		},
		IdleTimeout: func() *int { timeout := 1; return &timeout }(), // Any value > 0
	}

	inst1, err := manager.CreateInstance("instance-1", options1)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	inst2, err := manager.CreateInstance("instance-2", options2)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	inst3, err := manager.CreateInstance("instance-3", options3)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Set up mock time and set instances to running
	mockTime := NewMockTimeProvider(time.Now())
	inst1.SetTimeProvider(mockTime)
	inst2.SetTimeProvider(mockTime)
	inst3.SetTimeProvider(mockTime)

	inst1.SetStatus(instance.Running)
	inst2.SetStatus(instance.Running)
	inst3.SetStatus(instance.Running)

	// Set different last request times (oldest to newest)
	// inst1: oldest (will be evicted)
	inst1.UpdateLastRequestTime()

	mockTime.SetTime(mockTime.Now().Add(1 * time.Minute))
	inst2.UpdateLastRequestTime()

	mockTime.SetTime(mockTime.Now().Add(1 * time.Minute))
	inst3.UpdateLastRequestTime()

	// Evict LRU instance (should be inst1)
	err = manager.EvictLRUInstance()
	if err != nil {
		t.Fatalf("EvictLRUInstance failed: %v", err)
	}

	// Verify inst1 is stopped
	if inst1.IsRunning() {
		t.Error("Expected instance-1 to be stopped after eviction")
	}

	// Verify inst2 and inst3 are still running
	if !inst2.IsRunning() {
		t.Error("Expected instance-2 to still be running")
	}
	if !inst3.IsRunning() {
		t.Error("Expected instance-3 to still be running")
	}

	// Clean up manually - set all to stopped and then shutdown
	inst2.SetStatus(instance.Stopped)
	inst3.SetStatus(instance.Stopped)
}

func TestEvictLRUInstance_NoEligibleInstances(t *testing.T) {
	manager := createTestManager()
	defer manager.Shutdown()

	// Test 1: No running instances
	err := manager.EvictLRUInstance()
	if err == nil {
		t.Error("Expected error when no running instances exist")
	}
	if err.Error() != "failed to find lru instance" {
		t.Errorf("Expected 'failed to find lru instance' error, got: %v", err)
	}

	// Test 2: Only instances with IdleTimeout <= 0
	options1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model1.gguf",
		},
		IdleTimeout: func() *int { timeout := 0; return &timeout }(), // 0 = no timeout
	}
	options2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model2.gguf",
		},
		IdleTimeout: func() *int { timeout := -1; return &timeout }(), // negative = no timeout
	}
	options3 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model3.gguf",
		},
		// No IdleTimeout set (nil)
	}

	inst1, err := manager.CreateInstance("no-timeout-1", options1)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	inst2, err := manager.CreateInstance("no-timeout-2", options2)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	inst3, err := manager.CreateInstance("no-timeout-3", options3)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Set instances to running
	inst1.SetStatus(instance.Running)
	inst2.SetStatus(instance.Running)
	inst3.SetStatus(instance.Running)

	// Try to evict - should fail because no eligible instances
	err = manager.EvictLRUInstance()
	if err == nil {
		t.Error("Expected error when no eligible instances exist")
	}
	if err.Error() != "failed to find lru instance" {
		t.Errorf("Expected 'failed to find lru instance' error, got: %v", err)
	}

	// Verify all instances are still running
	if !inst1.IsRunning() {
		t.Error("Expected no-timeout-1 to still be running")
	}
	if !inst2.IsRunning() {
		t.Error("Expected no-timeout-2 to still be running")
	}
	if !inst3.IsRunning() {
		t.Error("Expected no-timeout-3 to still be running")
	}

	// Reset instances to stopped to avoid shutdown panics
	inst1.SetStatus(instance.Stopped)
	inst2.SetStatus(instance.Stopped)
	inst3.SetStatus(instance.Stopped)
}

func TestEvictLRUInstance_SkipsInstancesWithoutTimeout(t *testing.T) {
	manager := createTestManager()
	defer manager.Shutdown()

	// Create mix of instances: some with timeout enabled, some disabled
	optionsWithTimeout := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model-with-timeout.gguf",
		},
		IdleTimeout: func() *int { timeout := 1; return &timeout }(), // Any value > 0
	}
	optionsWithoutTimeout1 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model-no-timeout1.gguf",
		},
		IdleTimeout: func() *int { timeout := 0; return &timeout }(), // 0 = no timeout
	}
	optionsWithoutTimeout2 := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model-no-timeout2.gguf",
		},
		// No IdleTimeout set (nil)
	}

	instWithTimeout, err := manager.CreateInstance("with-timeout", optionsWithTimeout)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	instNoTimeout1, err := manager.CreateInstance("no-timeout-1", optionsWithoutTimeout1)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	instNoTimeout2, err := manager.CreateInstance("no-timeout-2", optionsWithoutTimeout2)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	// Set all instances to running
	instWithTimeout.SetStatus(instance.Running)
	instNoTimeout1.SetStatus(instance.Running)
	instNoTimeout2.SetStatus(instance.Running)

	// Update request times
	instWithTimeout.UpdateLastRequestTime()
	instNoTimeout1.UpdateLastRequestTime()
	instNoTimeout2.UpdateLastRequestTime()

	// Evict LRU instance - should only consider the one with timeout
	err = manager.EvictLRUInstance()
	if err != nil {
		t.Fatalf("EvictLRUInstance failed: %v", err)
	}

	// Verify only the instance with timeout was evicted
	if instWithTimeout.IsRunning() {
		t.Error("Expected with-timeout instance to be stopped after eviction")
	}
	if !instNoTimeout1.IsRunning() {
		t.Error("Expected no-timeout-1 instance to still be running")
	}
	if !instNoTimeout2.IsRunning() {
		t.Error("Expected no-timeout-2 instance to still be running")
	}

	// Reset instances to stopped to avoid shutdown panics
	instNoTimeout1.SetStatus(instance.Stopped)
	instNoTimeout2.SetStatus(instance.Stopped)
}
