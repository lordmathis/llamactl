package server_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/manager"
	"llamactl/pkg/server"
)

// newInstancesTestRouter wires the instance endpoints to a real manager over
// an in-memory DB, bypassing auth middleware.
func newInstancesTestRouter(t *testing.T) http.Handler {
	t.Helper()

	cfg := &config.AppConfig{
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
			MaxInstances:         10,
			MaxRunningInstances:  10,
			DefaultAutoRestart:   true,
			DefaultMaxRestarts:   3,
			LogsDir:              t.TempDir(),
			DefaultRestartDelay:  5,
			DefaultIdleTimeout:   30,
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

	db, err := database.Open(&database.Config{
		Path:               cfg.Database.Path,
		MaxOpenConnections: cfg.Database.MaxOpenConnections,
		MaxIdleConnections: cfg.Database.MaxIdleConnections,
		ConnMaxLifetime:    cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	h := server.NewHandler(manager.New(cfg, db), nil, *cfg, nil, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/instances/{name}", h.GetInstance())
	r.Put("/api/v1/instances/{name}", h.UpdateInstance())
	r.Post("/api/v1/instances/{name}/start", h.StartInstance())
	r.Post("/api/v1/instances/{name}/stop", h.StopInstance())
	r.Post("/api/v1/instances/{name}/restart", h.RestartInstance())
	r.Delete("/api/v1/instances/{name}", h.DeleteInstance())
	r.Get("/api/v1/instances/{name}/logs", h.GetInstanceLogs())
	r.Get("/api/v1/instances/{name}/proxy", h.InstanceProxy())
	return r
}

// TestInstanceEndpoints_NotFound verifies every instance endpoint returns
// 404 for a missing instance.
func TestInstanceEndpoints_NotFound(t *testing.T) {
	router := newInstancesTestRouter(t)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{"get instance", http.MethodGet, "/api/v1/instances/ghost", "", http.StatusNotFound},
		{"update instance", http.MethodPut, "/api/v1/instances/ghost", `{"backend_type":"llama_cpp"}`, http.StatusNotFound},
		{"start instance", http.MethodPost, "/api/v1/instances/ghost/start", "", http.StatusNotFound},
		{"stop instance", http.MethodPost, "/api/v1/instances/ghost/stop", "", http.StatusNotFound},
		{"restart instance", http.MethodPost, "/api/v1/instances/ghost/restart", "", http.StatusNotFound},
		{"delete instance", http.MethodDelete, "/api/v1/instances/ghost", "", http.StatusNotFound},
		{"instance logs", http.MethodGet, "/api/v1/instances/ghost/logs", "", http.StatusNotFound},
		{"instance proxy", http.MethodGet, "/api/v1/instances/ghost/proxy", "", http.StatusNotFound},
		// Controls: these must stay 400 so 404 stays meaningful.
		{"invalid name is still 400", http.MethodGet, "/api/v1/instances/bad@name", "", http.StatusBadRequest},
		{"invalid name on proxy is still 400", http.MethodGet, "/api/v1/instances/bad@name/proxy", "", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = bytes.NewReader([]byte(tt.body))
			}
			req := httptest.NewRequest(tt.method, tt.path, body)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("%s %s: status = %d, expected %d, body: %s",
					tt.method, tt.path, recorder.Code, tt.expectedStatus, recorder.Body.String())
			}
		})
	}
}
