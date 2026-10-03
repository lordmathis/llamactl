package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"llamactl/pkg/config"
	"llamactl/pkg/server"
)

func newNodesTestHandler(t *testing.T, nodes map[string]config.NodeConfig, localNode string) http.Handler {
	t.Helper()
	cfg := config.AppConfig{
		LocalNode: localNode,
		Nodes:     nodes,
	}
	h := server.NewHandler(nil, nil, cfg, nil, nil)
	return h.GetNodeHealth()
}

func TestGetNodeHealth(t *testing.T) {
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/version" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer node-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer remote.Close()

	tests := []struct {
		name           string
		nodes          map[string]config.NodeConfig
		localNode      string
		requestName    string
		expectedStatus int
		expectedHealth server.NodeHealthStatus
	}{
		{
			name:           "local node is healthy without network call",
			nodes:          map[string]config.NodeConfig{"main": {}},
			localNode:      "main",
			requestName:    "main",
			expectedStatus: http.StatusOK,
			expectedHealth: server.NodeStatusHealthy,
		},
		{
			name: "reachable remote node with valid api key",
			nodes: map[string]config.NodeConfig{
				"main":  {},
				"other": {Address: remote.URL, APIKey: "node-key"},
			},
			localNode:      "main",
			requestName:    "other",
			expectedStatus: http.StatusOK,
			expectedHealth: server.NodeStatusHealthy,
		},
		{
			name: "remote node rejecting the api key",
			nodes: map[string]config.NodeConfig{
				"main":  {},
				"other": {Address: remote.URL, APIKey: "wrong-key"},
			},
			localNode:      "main",
			requestName:    "other",
			expectedStatus: http.StatusOK,
			expectedHealth: server.NodeStatusUnreachable,
		},
		{
			name: "unreachable remote node",
			nodes: map[string]config.NodeConfig{
				"main":  {},
				"other": {Address: "http://127.0.0.1:1", APIKey: "node-key"},
			},
			localNode:      "main",
			requestName:    "other",
			expectedStatus: http.StatusOK,
			expectedHealth: server.NodeStatusUnreachable,
		},
		{
			name:           "unknown node returns 404",
			nodes:          map[string]config.NodeConfig{"main": {}},
			localNode:      "main",
			requestName:    "ghost",
			expectedStatus: http.StatusNotFound,
			expectedHealth: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Route through chi so URL params are populated
			r := chi.NewRouter()
			r.Get("/api/v1/nodes/{name}/health", newNodesTestHandler(t, tt.nodes, tt.localNode).ServeHTTP)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/"+tt.requestName+"/health", nil)
			recorder := httptest.NewRecorder()
			r.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf("GetNodeHealth() status = %d, expected %d, body: %s", recorder.Code, tt.expectedStatus, recorder.Body.String())
			}

			if tt.expectedStatus != http.StatusOK {
				return
			}

			var health server.NodeHealthResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &health); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if health.Status != tt.expectedHealth {
				t.Errorf("GetNodeHealth() status field = %q, expected %q (error: %q)", health.Status, tt.expectedHealth, health.Error)
			}

			if health.Status == server.NodeStatusUnreachable && health.Error == "" {
				t.Errorf("unreachable node should report an error, got empty")
			}
		})
	}
}
