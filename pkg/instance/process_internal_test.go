package instance

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// waitForHealthy is unexported, so this internal test exercises it directly
// to prove the backend-configured health path flows through (spec: custom
// backends). The server returns 200 only on the custom path, 404 on /health.
func TestWaitForHealthy_CustomHealthPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	host, portStr, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse test server address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse test server port: %v", err)
	}

	// The health path comes from backends.custom.<name>.health_path in
	// config, not from instance options.
	newTestInstance := func(healthPath string) *Instance {
		inst := &Instance{Name: "custom-health"}
		inst.status = newStatus(Running)
		inst.globalBackendSettings = &config.BackendConfig{
			Custom: map[string]config.BackendSettings{
				"test-backend": {HealthPath: healthPath},
			},
		}
		inst.options = newOptions(&Options{
			BackendOptions: backends.Options{
				BackendType: backends.BackendTypeCustom,
				CustomServerOptions: &backends.CustomServerOptions{
					Name: "test-backend",
					Host: host,
					Port: port,
				},
			},
		})
		inst.process = newProcess(inst)
		return inst
	}

	t.Run("configured path returns healthy", func(t *testing.T) {
		if err := newTestInstance("/ready").process.waitForHealthy(2); err != nil {
			t.Errorf("expected healthy on custom path, got: %v", err)
		}
	})

	t.Run("path other than the configured one times out", func(t *testing.T) {
		// The server 404s /health; success here would mean the
		// configured path was ignored and /health was probed anyway.
		if err := newTestInstance("/health").process.waitForHealthy(1); err == nil {
			t.Error("expected timeout on /health (server returns 404), got healthy")
		}
	})
}
