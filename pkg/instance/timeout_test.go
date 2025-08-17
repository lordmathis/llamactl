package instance_test

import (
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/testutil"
	"sync/atomic"
	"testing"
	"time"
)

// MockTimeProvider implements TimeProvider for testing
type MockTimeProvider struct {
	currentTime atomic.Int64 // Unix timestamp
}

func NewMockTimeProvider(t time.Time) *MockTimeProvider {
	m := &MockTimeProvider{}
	m.currentTime.Store(t.Unix())
	return m
}

func (m *MockTimeProvider) Now() time.Time {
	return time.Unix(m.currentTime.Load(), 0)
}

func (m *MockTimeProvider) SetTime(t time.Time) {
	m.currentTime.Store(t.Unix())
}

// Timeout-related tests

func TestUpdateLastRequestTime(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)

	// Test that UpdateLastRequestTime doesn't panic
	inst.UpdateLastRequestTime()

	// Test concurrent calls to ensure thread safety
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			inst.UpdateLastRequestTime()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestShouldTimeout_NotRunning(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 1 // 1 minute
	options := &instance.CreateInstanceOptions{
		IdleTimeout: &idleTimeout,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)

	// Instance is not running, should not timeout regardless of configuration
	if inst.ShouldTimeout() {
		t.Error("Non-running instance should never timeout")
	}
}

func TestShouldTimeout_NoTimeoutConfigured(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	tests := []struct {
		name        string
		idleTimeout *int
	}{
		{"nil timeout", nil},
		{"zero timeout", testutil.IntPtr(0)},
		{"negative timeout", testutil.IntPtr(-5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.CreateInstanceOptions{
				IdleTimeout: tt.idleTimeout,
				LlamaServerOptions: llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			inst := instance.NewInstance("test-instance", globalSettings, options)
			// Simulate running state
			inst.Running = true

			if inst.ShouldTimeout() {
				t.Errorf("Instance with %s should not timeout", tt.name)
			}
		})
	}
}

func TestShouldTimeout_WithinTimeLimit(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 5 // 5 minutes
	options := &instance.CreateInstanceOptions{
		IdleTimeout: &idleTimeout,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)
	inst.Running = true

	// Update last request time to now
	inst.UpdateLastRequestTime()

	// Should not timeout immediately
	if inst.ShouldTimeout() {
		t.Error("Instance should not timeout when last request was recent")
	}
}

func TestShouldTimeout_ExceedsTimeLimit(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 1 // 1 minute
	options := &instance.CreateInstanceOptions{
		IdleTimeout: &idleTimeout,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)
	inst.Running = true

	// Use MockTimeProvider to simulate old last request time
	mockTime := NewMockTimeProvider(time.Now())
	inst.SetTimeProvider(mockTime)

	// Set last request time to now
	inst.UpdateLastRequestTime()

	// Advance time by 2 minutes (exceeds 1 minute timeout)
	mockTime.SetTime(time.Now().Add(2 * time.Minute))

	if !inst.ShouldTimeout() {
		t.Error("Instance should timeout when last request exceeds idle timeout")
	}
}

func TestShouldTimeout_TimeUnitConversion(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	tests := []struct {
		name           string
		idleTimeout    int // in minutes
		lastRequestAge int // seconds ago
		shouldTimeout  bool
	}{
		{"exactly at timeout boundary", 2, 120, false}, // 2 minutes = 120 seconds
		{"just over timeout", 2, 121, true},            // 121 seconds > 120 seconds
		{"well under timeout", 5, 60, false},           // 1 minute < 5 minutes
		{"well over timeout", 1, 300, true},            // 5 minutes > 1 minute
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.CreateInstanceOptions{
				IdleTimeout: &tt.idleTimeout,
				LlamaServerOptions: llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			inst := instance.NewInstance("test-instance", globalSettings, options)
			inst.Running = true

			// Use MockTimeProvider to control time
			mockTime := NewMockTimeProvider(time.Now())
			inst.SetTimeProvider(mockTime)

			// Set last request time to now
			inst.UpdateLastRequestTime()

			// Advance time by the specified amount
			mockTime.SetTime(time.Now().Add(time.Duration(tt.lastRequestAge) * time.Second))

			result := inst.ShouldTimeout()
			if result != tt.shouldTimeout {
				t.Errorf("Expected timeout=%v for %s, got %v", tt.shouldTimeout, tt.name, result)
			}
		})
	}
}

func TestInstanceTimeoutInitialization(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 5
	options := &instance.CreateInstanceOptions{
		IdleTimeout: &idleTimeout,
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)
	inst.Running = true

	// Use MockTimeProvider to control time and verify initialization
	mockTime := NewMockTimeProvider(time.Now())
	inst.SetTimeProvider(mockTime)

	// Update last request time (simulates what Start() does)
	inst.UpdateLastRequestTime()

	// Should not timeout immediately after initialization with a 5-minute timeout
	if inst.ShouldTimeout() {
		t.Error("Fresh instance should not timeout immediately")
	}

	// Now advance time by 6 minutes and verify it would timeout
	mockTime.SetTime(time.Now().Add(6 * time.Minute))
	if !inst.ShouldTimeout() {
		t.Error("Instance should timeout after exceeding idle timeout period")
	}
}

func TestTimeoutConfiguration_DefaultValue(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	// Create instance without specifying idle timeout
	options := &instance.CreateInstanceOptions{
		LlamaServerOptions: llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	inst := instance.NewInstance("test-instance", globalSettings, options)
	opts := inst.GetOptions()

	// Default should be 0 (disabled)
	if opts.IdleTimeout == nil || *opts.IdleTimeout != 0 {
		t.Errorf("Expected default IdleTimeout to be 0, got %v", opts.IdleTimeout)
	}
}

func TestTimeoutConfiguration_Validation(t *testing.T) {
	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	tests := []struct {
		name            string
		inputTimeout    *int
		expectedTimeout int
	}{
		{"positive value", testutil.IntPtr(10), 10},
		{"zero value", testutil.IntPtr(0), 0},
		{"negative value gets corrected", testutil.IntPtr(-5), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.CreateInstanceOptions{
				IdleTimeout: tt.inputTimeout,
				LlamaServerOptions: llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			inst := instance.NewInstance("test-instance", globalSettings, options)
			opts := inst.GetOptions()

			if opts.IdleTimeout == nil || *opts.IdleTimeout != tt.expectedTimeout {
				t.Errorf("Expected IdleTimeout %d, got %v", tt.expectedTimeout, opts.IdleTimeout)
			}
		})
	}
}
