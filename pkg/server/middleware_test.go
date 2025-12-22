package server_test

import (
	"llamactl/pkg/config"
	"llamactl/pkg/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInferenceAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		managementKeys []string
		requestKey     string
		method         string
		expectedStatus int
	}{
		{
			name:           "valid management key for inference",
			managementKeys: []string{"sk-management-admin123"},
			requestKey:     "sk-management-admin123",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid key",
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "sk-management-invalid",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing key",
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "OPTIONS request bypasses auth",
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "management key works for inference endpoint",
			managementKeys: []string{"sk-management-admin"},
			requestKey:     "sk-management-admin",
			method:         "POST",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.AuthConfig{
				RequireInferenceAuth: true,
				ManagementKeys:       tt.managementKeys,
			}
			middleware := server.NewAPIAuthMiddleware(cfg, nil)

			// Create test request
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.requestKey != "" {
				req.Header.Set("Authorization", "Bearer "+tt.requestKey)
			}

			// Create test handler
			handler := middleware.InferenceAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Execute request
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("InferenceAuthMiddleware() status = %v, expected %v", recorder.Code, tt.expectedStatus)
			}

			// Check that unauthorized responses have proper format
			if recorder.Code == http.StatusUnauthorized {
				contentType := recorder.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Unauthorized response Content-Type = %v, expected application/json", contentType)
				}

				body := recorder.Body.String()
				if !strings.Contains(body, `"type": "authentication_error"`) {
					t.Errorf("Unauthorized response missing proper error type: %v", body)
				}
			}
		})
	}
}

func TestManagementAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		managementKeys []string
		requestKey     string
		method         string
		expectedStatus int
	}{
		{
			name:           "valid management key",
			managementKeys: []string{"sk-management-admin123"},
			requestKey:     "sk-management-admin123",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid management key",
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "sk-management-invalid",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing management key",
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "OPTIONS request bypasses management auth",
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.AuthConfig{
				RequireManagementAuth: true,
				ManagementKeys:        tt.managementKeys,
			}
			middleware := server.NewAPIAuthMiddleware(cfg, nil)

			// Create test request
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.requestKey != "" {
				req.Header.Set("Authorization", "Bearer "+tt.requestKey)
			}

			// Create test handler
			handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Execute request
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("ManagementAuthMiddleware() status = %v, expected %v", recorder.Code, tt.expectedStatus)
			}

			// Check that unauthorized responses have proper format
			if recorder.Code == http.StatusUnauthorized {
				contentType := recorder.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("Unauthorized response Content-Type = %v, expected application/json", contentType)
				}

				body := recorder.Body.String()
				if !strings.Contains(body, `"type": "authentication_error"`) {
					t.Errorf("Unauthorized response missing proper error type: %v", body)
				}
			}
		})
	}
}

func TestManagementKeyAutoGeneration(t *testing.T) {
	// Test auto-generation for management keys
	config := config.AuthConfig{
		RequireManagementAuth: true,
		ManagementKeys:        []string{}, // Empty to trigger generation
	}

	// Create middleware - this should trigger key generation
	middleware := server.NewAPIAuthMiddleware(config, nil)

	// Test that auth is required (meaning a key was generated)
	req := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(recorder, req)

	// Should be unauthorized without a key (proving that a key was generated and auth is working)
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized without key, got status %v", recorder.Code)
	}

	// Test uniqueness by creating another middleware instance
	middleware2 := server.NewAPIAuthMiddleware(config, nil)

	req2 := httptest.NewRequest("GET", "/", nil)
	recorder2 := httptest.NewRecorder()

	handler2 := middleware2.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler2.ServeHTTP(recorder2, req2)

	// Both should require auth (proving keys were generated for both instances)
	if recorder2.Code != http.StatusUnauthorized {
		t.Errorf("Expected unauthorized for second middleware without key, got status %v", recorder2.Code)
	}
}

func TestAutoGenerationScenarios(t *testing.T) {
	tests := []struct {
		name               string
		requireManagement  bool
		providedManagement []string
		shouldGenerate     bool
	}{
		{
			name:               "management auth required, keys provided - no generation",
			requireManagement:  true,
			providedManagement: []string{"sk-management-provided"},
			shouldGenerate:     false,
		},
		{
			name:               "management auth required, no keys - should auto-generate",
			requireManagement:  true,
			providedManagement: []string{},
			shouldGenerate:     true,
		},
		{
			name:               "management auth not required - no generation",
			requireManagement:  false,
			providedManagement: []string{},
			shouldGenerate:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.AuthConfig{
				RequireManagementAuth: tt.requireManagement,
				ManagementKeys:        tt.providedManagement,
			}

			middleware := server.NewAPIAuthMiddleware(cfg, nil)

			// Test management behavior if management auth is required
			if tt.requireManagement {
				req := httptest.NewRequest("GET", "/api/v1/instances", nil)
				recorder := httptest.NewRecorder()

				handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				handler.ServeHTTP(recorder, req)

				// Should always be unauthorized without a key (since middleware assumes auth is required)
				if recorder.Code != http.StatusUnauthorized {
					t.Errorf("Expected unauthorized for management without key, got status %v", recorder.Code)
				}
			}
		})
	}
}
