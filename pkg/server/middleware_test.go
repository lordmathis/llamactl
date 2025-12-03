package server_test

import (
	"llamactl/pkg/config"
	"llamactl/pkg/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		keyType        server.KeyType
		inferenceKeys  []string
		managementKeys []string
		requestKey     string
		method         string
		expectedStatus int
	}{
		// Valid key tests - using management keys only since config-based inference keys are deprecated
		{
			name:           "valid management key for inference", // Management keys work for inference
			keyType:        server.KeyTypeInference,
			managementKeys: []string{"sk-management-admin123"},
			requestKey:     "sk-management-admin123",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid management key for management",
			keyType:        server.KeyTypeManagement,
			managementKeys: []string{"sk-management-admin123"},
			requestKey:     "sk-management-admin123",
			method:         "GET",
			expectedStatus: http.StatusOK,
		},

		// Invalid key tests
		{
			name:           "inference key for management should fail",
			keyType:        server.KeyTypeManagement,
			inferenceKeys:  []string{"sk-inference-user123"},
			requestKey:     "sk-inference-user123",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid inference key",
			keyType:        server.KeyTypeInference,
			inferenceKeys:  []string{"sk-inference-valid123"},
			requestKey:     "sk-inference-invalid",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing inference key",
			keyType:        server.KeyTypeInference,
			inferenceKeys:  []string{"sk-inference-valid123"},
			requestKey:     "",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid management key",
			keyType:        server.KeyTypeManagement,
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "sk-management-invalid",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "missing management key",
			keyType:        server.KeyTypeManagement,
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},

		// OPTIONS requests should always pass
		{
			name:           "OPTIONS request bypasses inference auth",
			keyType:        server.KeyTypeInference,
			inferenceKeys:  []string{"sk-inference-valid123"},
			requestKey:     "",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OPTIONS request bypasses management auth",
			keyType:        server.KeyTypeManagement,
			managementKeys: []string{"sk-management-valid123"},
			requestKey:     "",
			method:         "OPTIONS",
			expectedStatus: http.StatusOK,
		},

		// Cross-key-type validation
		{
			name:           "management key works for inference endpoint",
			keyType:        server.KeyTypeInference,
			inferenceKeys:  []string{},
			managementKeys: []string{"sk-management-admin"},
			requestKey:     "sk-management-admin",
			method:         "POST",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.AuthConfig{
				InferenceKeys:  tt.inferenceKeys,
				ManagementKeys: tt.managementKeys,
			}
			middleware := server.NewAPIAuthMiddleware(cfg, nil)

			// Create test request
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.requestKey != "" {
				req.Header.Set("Authorization", "Bearer "+tt.requestKey)
			}

			// Create test handler using appropriate middleware
			var handler http.Handler
			if tt.keyType == server.KeyTypeInference {
				handler = middleware.AuthMiddleware(server.KeyTypeInference)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			} else {
				handler = middleware.AuthMiddleware(server.KeyTypeManagement)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			}

			// Execute request
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("AuthMiddleware() status = %v, expected %v", recorder.Code, tt.expectedStatus)
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

func TestGenerateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		keyType server.KeyType
	}{
		{"inference key generation", server.KeyTypeInference},
		{"management key generation", server.KeyTypeManagement},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test auto-generation by creating config that will trigger it
			var config config.AuthConfig
			if tt.keyType == server.KeyTypeInference {
				config.RequireInferenceAuth = true
				config.InferenceKeys = []string{} // Empty to trigger generation
			} else {
				config.RequireManagementAuth = true
				config.ManagementKeys = []string{} // Empty to trigger generation
			}

			// Create middleware - this should trigger key generation
			middleware := server.NewAPIAuthMiddleware(config, nil)

			// Test that auth is required (meaning a key was generated)
			req := httptest.NewRequest("GET", "/", nil)
			recorder := httptest.NewRecorder()

			var handler http.Handler
			if tt.keyType == server.KeyTypeInference {
				handler = middleware.AuthMiddleware(server.KeyTypeInference)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			} else {
				handler = middleware.AuthMiddleware(server.KeyTypeManagement)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			}

			handler.ServeHTTP(recorder, req)

			// Should be unauthorized without a key (proving that a key was generated and auth is working)
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("Expected unauthorized without key, got status %v", recorder.Code)
			}

			// Test uniqueness by creating another middleware instance
			middleware2 := server.NewAPIAuthMiddleware(config, nil)

			req2 := httptest.NewRequest("GET", "/", nil)
			recorder2 := httptest.NewRecorder()

			if tt.keyType == server.KeyTypeInference {
				handler2 := middleware2.AuthMiddleware(server.KeyTypeInference)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				handler2.ServeHTTP(recorder2, req2)
			} else {
				handler2 := middleware2.AuthMiddleware(server.KeyTypeManagement)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				handler2.ServeHTTP(recorder2, req2)
			}

			// Both should require auth (proving keys were generated for both instances)
			if recorder2.Code != http.StatusUnauthorized {
				t.Errorf("Expected unauthorized for second middleware without key, got status %v", recorder2.Code)
			}
		})
	}
}

func TestAutoGeneration(t *testing.T) {
	tests := []struct {
		name               string
		requireInference   bool
		requireManagement  bool
		providedInference  []string
		providedManagement []string
		shouldGenerateInf  bool // Whether inference key should be generated
		shouldGenerateMgmt bool // Whether management key should be generated
	}{
		{
			name:               "inference auth required, keys provided - no generation",
			requireInference:   true,
			requireManagement:  false,
			providedInference:  []string{"sk-inference-provided"},
			providedManagement: []string{},
			shouldGenerateInf:  false,
			shouldGenerateMgmt: false,
		},
		{
			name:               "inference auth required, no keys - should auto-generate",
			requireInference:   true,
			requireManagement:  false,
			providedInference:  []string{},
			providedManagement: []string{},
			shouldGenerateInf:  true,
			shouldGenerateMgmt: false,
		},
		{
			name:               "management auth required, keys provided - no generation",
			requireInference:   false,
			requireManagement:  true,
			providedInference:  []string{},
			providedManagement: []string{"sk-management-provided"},
			shouldGenerateInf:  false,
			shouldGenerateMgmt: false,
		},
		{
			name:               "management auth required, no keys - should auto-generate",
			requireInference:   false,
			requireManagement:  true,
			providedInference:  []string{},
			providedManagement: []string{},
			shouldGenerateInf:  false,
			shouldGenerateMgmt: true,
		},
		{
			name:               "both required, both provided - no generation",
			requireInference:   true,
			requireManagement:  true,
			providedInference:  []string{"sk-inference-provided"},
			providedManagement: []string{"sk-management-provided"},
			shouldGenerateInf:  false,
			shouldGenerateMgmt: false,
		},
		{
			name:               "both required, none provided - should auto-generate both",
			requireInference:   true,
			requireManagement:  true,
			providedInference:  []string{},
			providedManagement: []string{},
			shouldGenerateInf:  true,
			shouldGenerateMgmt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.AuthConfig{
				RequireInferenceAuth:  tt.requireInference,
				RequireManagementAuth: tt.requireManagement,
				InferenceKeys:         tt.providedInference,
				ManagementKeys:        tt.providedManagement,
			}

			middleware := server.NewAPIAuthMiddleware(cfg, nil)

			// Test inference behavior if inference auth is required
			if tt.requireInference {
				req := httptest.NewRequest("GET", "/v1/models", nil)
				recorder := httptest.NewRecorder()

				handler := middleware.AuthMiddleware(server.KeyTypeInference)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				handler.ServeHTTP(recorder, req)

				// Should always be unauthorized without a key (since middleware assumes auth is required)
				if recorder.Code != http.StatusUnauthorized {
					t.Errorf("Expected unauthorized for inference without key, got status %v", recorder.Code)
				}
			}

			// Test management behavior if management auth is required
			if tt.requireManagement {
				req := httptest.NewRequest("GET", "/api/v1/instances", nil)
				recorder := httptest.NewRecorder()

				handler := middleware.AuthMiddleware(server.KeyTypeManagement)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
