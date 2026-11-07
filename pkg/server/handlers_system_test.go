package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"llamactl/pkg/config"
	"llamactl/pkg/server"
)

// The config endpoint must expose only allowlisted fields. Credential-bearing
// config (backend/docker environments, args, response headers) must never
// reach the response body.
func TestConfigHandler_AllowlistsFields(t *testing.T) {
	cfg := config.AppConfig{
		LocalNode: "local",
		Instances: config.InstancesConfig{
			DefaultIdleTimeout:   30,
			DefaultAutoRestart:   true,
			DefaultMaxRestarts:   3,
			DefaultRestartDelay:  5,
			DefaultOnDemandStart: true,
		},
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command:         "/usr/local/bin/llama-server",
				Args:            []string{"--api-key", "SENTINEL_ARG"},
				Environment:     map[string]string{"HF_TOKEN": "SENTINEL_ENV"},
				ResponseHeaders: map[string]string{"Authorization": "SENTINEL_HEADER"},
				Docker: &config.DockerSettings{
					Enabled:     true,
					Image:       "ghcr.io/llamactl/llama-cpp:latest",
					Args:        []string{"--secret", "SENTINEL_DOCKER_ARG"},
					Environment: map[string]string{"HF_TOKEN": "SENTINEL_DOCKER_ENV"},
				},
			},
			Custom: map[string]config.BackendSettings{
				"mybackend": {
					Command:     "/opt/my-backend",
					Environment: map[string]string{"HF_TOKEN": "SENTINEL_CUSTOM_ENV"},
				},
			},
		},
	}
	h, err := server.NewHandler(nil, nil, cfg, nil, nil)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	rec := httptest.NewRecorder()
	h.ConfigHandler()(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()

	for _, want := range []string{
		`"local_node":"local"`,
		`"instance_defaults"`,
		`"idle_timeout":30`,
		`"command":"/usr/local/bin/llama-server"`,
		`"enabled":true`,
		`"ghcr.io/llamactl/llama-cpp:latest"`,
		`"mybackend"`,
		`"/opt/my-backend"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %s (body: %s)", want, body)
		}
	}

	if strings.Contains(body, "SENTINEL") {
		t.Errorf("response leaked a sentinel secret (body: %s)", body)
	}
	for _, key := range []string{`"environment"`, `"args"`, `"response_headers"`} {
		if strings.Contains(body, key) {
			t.Errorf("response contains excluded key %s (body: %s)", key, body)
		}
	}
}
