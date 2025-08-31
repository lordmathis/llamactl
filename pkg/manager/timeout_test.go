package manager_test

import (
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"sync"
	"testing"
	"time"
)

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
	// Helper function to create instances with different timeout configurations
	createInstanceWithTimeout := func(manager manager.InstanceManager, name, model string, timeout *int) *instance.Process {
		options := &instance.CreateInstanceOptions{
			LlamaServerOptions: llamacpp.LlamaServerOptions{
				Model: model,
			},
			IdleTimeout: timeout,
		}
		inst, err := manager.CreateInstance(name, options)
		if err != nil {
			t.Fatalf("CreateInstance failed: %v", err)
		}
		return inst
	}

	t.Run("no running instances", func(t *testing.T) {
		manager := createTestManager()
		defer manager.Shutdown()

		err := manager.EvictLRUInstance()
		if err == nil {
			t.Error("Expected error when no running instances exist")
		}
		if err.Error() != "failed to find lru instance" {
			t.Errorf("Expected 'failed to find lru instance' error, got: %v", err)
		}
	})

	t.Run("only instances without timeout", func(t *testing.T) {
		manager := createTestManager()
		defer manager.Shutdown()

		// Create instances with various non-eligible timeout configurations
		zeroTimeout := 0
		negativeTimeout := -1
		inst1 := createInstanceWithTimeout(manager, "no-timeout-1", "/path/to/model1.gguf", &zeroTimeout)
		inst2 := createInstanceWithTimeout(manager, "no-timeout-2", "/path/to/model2.gguf", &negativeTimeout)
		inst3 := createInstanceWithTimeout(manager, "no-timeout-3", "/path/to/model3.gguf", nil)

		// Set instances to running
		instances := []*instance.Process{inst1, inst2, inst3}
		for _, inst := range instances {
			inst.SetStatus(instance.Running)
		}
		defer func() {
			// Reset instances to stopped to avoid shutdown panics
			for _, inst := range instances {
				inst.SetStatus(instance.Stopped)
			}
		}()

		// Try to evict - should fail because no eligible instances
		err := manager.EvictLRUInstance()
		if err == nil {
			t.Error("Expected error when no eligible instances exist")
		}
		if err.Error() != "failed to find lru instance" {
			t.Errorf("Expected 'failed to find lru instance' error, got: %v", err)
		}

		// Verify all instances are still running
		for i, inst := range instances {
			if !inst.IsRunning() {
				t.Errorf("Expected instance %d to still be running", i+1)
			}
		}
	})

	t.Run("mixed instances - evicts only eligible ones", func(t *testing.T) {
		manager := createTestManager()
		defer manager.Shutdown()

		// Create mix of instances: some with timeout enabled, some disabled
		validTimeout := 1
		zeroTimeout := 0
		instWithTimeout := createInstanceWithTimeout(manager, "with-timeout", "/path/to/model-with-timeout.gguf", &validTimeout)
		instNoTimeout1 := createInstanceWithTimeout(manager, "no-timeout-1", "/path/to/model-no-timeout1.gguf", &zeroTimeout)
		instNoTimeout2 := createInstanceWithTimeout(manager, "no-timeout-2", "/path/to/model-no-timeout2.gguf", nil)

		// Set all instances to running
		instances := []*instance.Process{instWithTimeout, instNoTimeout1, instNoTimeout2}
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
	})
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
