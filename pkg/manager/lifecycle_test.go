package manager_test

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"sync"
	"testing"
	"time"
)

func TestInstanceTimeoutLogic(t *testing.T) {
	testManager := createTestManager()
	defer testManager.Shutdown()

	idleTimeout := 1 // 1 minute
	inst := createInstanceWithTimeout(t, testManager, "timeout-test", "/path/to/model.gguf", &idleTimeout)

	// Test timeout logic with mock time provider
	mockTime := NewMockTimeProvider(time.Now())
	inst.SetTimeProvider(mockTime)

	// Set instance to running state so timeout logic can work
	inst.SetStatus(instance.Running)
	defer inst.SetStatus(instance.Stopped)

	// Update last request time
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
}

func TestInstanceWithoutTimeoutNeverExpires(t *testing.T) {
	testManager := createTestManager()
	defer testManager.Shutdown()

	noTimeoutInst := createInstanceWithTimeout(t, testManager, "no-timeout-test", "/path/to/model.gguf", nil)

	mockTime := NewMockTimeProvider(time.Now())
	noTimeoutInst.SetTimeProvider(mockTime)
	noTimeoutInst.SetStatus(instance.Running)
	defer noTimeoutInst.SetStatus(instance.Stopped)

	noTimeoutInst.UpdateLastRequestTime()

	// Advance time significantly
	mockTime.SetTime(mockTime.Now().Add(24 * time.Hour))

	// Even with time advanced, should not timeout
	if noTimeoutInst.ShouldTimeout() {
		t.Error("Instance without timeout configuration should never timeout")
	}
}

func TestEvictLRUInstance_Success(t *testing.T) {
	manager := createTestManager()
	defer manager.Shutdown()

	// Create 3 instances with idle timeout enabled (value doesn't matter for LRU logic)
	validTimeout := 1
	inst1 := createInstanceWithTimeout(t, manager, "instance-1", "/path/to/model1.gguf", &validTimeout)
	inst2 := createInstanceWithTimeout(t, manager, "instance-2", "/path/to/model2.gguf", &validTimeout)
	inst3 := createInstanceWithTimeout(t, manager, "instance-3", "/path/to/model3.gguf", &validTimeout)

	// Set up mock time and set instances to running
	mockTime := NewMockTimeProvider(time.Now())
	inst1.SetTimeProvider(mockTime)
	inst2.SetTimeProvider(mockTime)
	inst3.SetTimeProvider(mockTime)

	inst1.SetStatus(instance.Running)
	inst2.SetStatus(instance.Running)
	inst3.SetStatus(instance.Running)
	defer func() {
		// Clean up - ensure all instances are stopped
		for _, inst := range []*instance.Instance{inst1, inst2, inst3} {
			if inst.IsRunning() {
				inst.SetStatus(instance.Stopped)
			}
		}
	}()

	// Set different last request times (oldest to newest)
	// inst1: oldest (will be evicted)
	inst1.UpdateLastRequestTime()

	mockTime.SetTime(mockTime.Now().Add(1 * time.Minute))
	inst2.UpdateLastRequestTime()

	mockTime.SetTime(mockTime.Now().Add(1 * time.Minute))
	inst3.UpdateLastRequestTime()

	// Evict LRU instance (should be inst1)
	if err := manager.EvictLRUInstance(); err != nil {
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
}

func TestEvictLRUInstance_NoRunningInstances(t *testing.T) {
	manager := createTestManager()
	defer manager.Shutdown()

	err := manager.EvictLRUInstance()
	if err == nil {
		t.Error("Expected error when no running instances exist")
	}
	if err.Error() != "failed to find lru instance" {
		t.Errorf("Expected 'failed to find lru instance' error, got: %v", err)
	}
}

func TestEvictLRUInstance_OnlyEvictsTimeoutEnabledInstances(t *testing.T) {
	manager := createTestManager()
	defer manager.Shutdown()

	// Create mix of instances: some with timeout enabled, some disabled
	// Only timeout-enabled instances should be eligible for eviction
	validTimeout := 1
	zeroTimeout := 0
	instWithTimeout := createInstanceWithTimeout(t, manager, "with-timeout", "/path/to/model-with-timeout.gguf", &validTimeout)
	instNoTimeout1 := createInstanceWithTimeout(t, manager, "no-timeout-1", "/path/to/model-no-timeout1.gguf", &zeroTimeout)
	instNoTimeout2 := createInstanceWithTimeout(t, manager, "no-timeout-2", "/path/to/model-no-timeout2.gguf", nil)

	// Set all instances to running
	instances := []*instance.Instance{instWithTimeout, instNoTimeout1, instNoTimeout2}
	for _, inst := range instances {
		inst.SetStatus(instance.Running)
		inst.UpdateLastRequestTime()
	}
	defer func() {
		// Reset instances to stopped to avoid shutdown panics
		for _, inst := range instances {
			if inst.IsRunning() {
				inst.SetStatus(instance.Stopped)
			}
		}
	}()

	// Evict LRU instance - should only consider the one with timeout
	err := manager.EvictLRUInstance()
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
}

// Helper function to create instances with different timeout configurations
func createInstanceWithTimeout(t *testing.T, manager manager.InstanceManager, name, model string, timeout *int) *instance.Instance {
	t.Helper()
	options := &instance.Options{
		IdleTimeout: timeout,
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: model,
			},
		},
	}
	inst, err := manager.CreateInstance(name, options)
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	return inst
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
