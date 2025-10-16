package instance_test

import (
	"llamactl/pkg/backends"
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
	backendConfig := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	options := &instance.Options{
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, mockOnStatusChange)

	// Test that UpdateLastRequestTime doesn't panic
	inst.UpdateLastRequestTime()
}

func TestShouldTimeout_NotRunning(t *testing.T) {
	backendConfig := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 1 // 1 minute
	options := &instance.Options{
		IdleTimeout: &idleTimeout,
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, mockOnStatusChange)

	// Instance is not running, should not timeout regardless of configuration
	if inst.ShouldTimeout() {
		t.Error("Non-running instance should never timeout")
	}
}

func TestShouldTimeout_NoTimeoutConfigured(t *testing.T) {
	backendConfig := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

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
			// Mock onStatusChange function
			mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

			options := &instance.Options{
				IdleTimeout: tt.idleTimeout,
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, mockOnStatusChange)
			// Simulate running state
			inst.SetStatus(instance.Running)

			if inst.ShouldTimeout() {
				t.Errorf("Instance with %s should not timeout", tt.name)
			}
		})
	}
}

func TestShouldTimeout_WithinTimeLimit(t *testing.T) {
	backendConfig := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 5 // 5 minutes
	options := &instance.Options{
		IdleTimeout: &idleTimeout,
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, mockOnStatusChange)
	inst.SetStatus(instance.Running)

	// Update last request time to now
	inst.UpdateLastRequestTime()

	// Should not timeout immediately
	if inst.ShouldTimeout() {
		t.Error("Instance should not timeout when last request was recent")
	}
}

func TestShouldTimeout_ExceedsTimeLimit(t *testing.T) {
	backendConfig := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	idleTimeout := 1 // 1 minute
	options := &instance.Options{
		IdleTimeout: &idleTimeout,
		BackendType: backends.BackendTypeLlamaCpp,
		LlamaServerOptions: &llamacpp.LlamaServerOptions{
			Model: "/path/to/model.gguf",
		},
	}

	// Mock onStatusChange function
	mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

	inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, mockOnStatusChange)
	inst.SetStatus(instance.Running)

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

func TestTimeoutConfiguration_Validation(t *testing.T) {
	backendConfig := &config.BackendConfig{
		LlamaCpp: config.BackendSettings{
			Command: "llama-server",
		},
		MLX: config.BackendSettings{
			Command: "mlx_lm.server",
		},
	}

	globalSettings := &config.InstancesConfig{
		LogsDir: "/tmp/test",
	}

	tests := []struct {
		name            string
		inputTimeout    *int
		expectedTimeout int
	}{
		{"default value when nil", nil, 0},
		{"positive value", testutil.IntPtr(10), 10},
		{"zero value", testutil.IntPtr(0), 0},
		{"negative value gets corrected", testutil.IntPtr(-5), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &instance.Options{
				IdleTimeout: tt.inputTimeout,
				BackendType: backends.BackendTypeLlamaCpp,
				LlamaServerOptions: &llamacpp.LlamaServerOptions{
					Model: "/path/to/model.gguf",
				},
			}

			// Mock onStatusChange function
			mockOnStatusChange := func(oldStatus, newStatus instance.Status) {}

			inst := instance.NewInstance("test-instance", backendConfig, globalSettings, options, mockOnStatusChange)
			opts := inst.GetOptions()

			if opts.IdleTimeout == nil || *opts.IdleTimeout != tt.expectedTimeout {
				t.Errorf("Expected IdleTimeout %d, got %v", tt.expectedTimeout, opts.IdleTimeout)
			}
		})
	}
}
