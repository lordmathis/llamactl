package server

import (
	"llamactl/pkg/auth"
	"llamactl/pkg/config"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestManagementAuthMiddlewareOriginCheck(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		forwardedHost  string
		requestHost    string
		expectedStatus int
	}{{
		name:           "same-origin Origin passes",
		origin:         "http://example.com",
		expectedStatus: http.StatusOK,
	},
		{
			name:           "sibling origin is rejected",
			origin:         "http://sibling.example.net",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing Origin and Referer is rejected",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Origin matching X-Forwarded-Host passes",
			origin:         "http://llamactl.example.com",
			forwardedHost:  "llamactl.example.com",
			requestHost:    "internal:8080",
			expectedStatus: http.StatusOK,
		},
	}

	middleware, store := newSessionMiddleware()
	sess := store.Create("user-1", "Alice", "alice@example.com")

	handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/x/stop", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
			if tt.requestHost != "" {
				req.Host = tt.requestHost
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.forwardedHost != "" {
				req.Header.Set("X-Forwarded-Host", tt.forwardedHost)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("status = %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}

// Existing key tests use GET only; unsafe methods are where a wrongly-scoped
// CSRF check would break every API client.
func TestManagementAuthMiddlewareSkipsOriginCheckForKeys(t *testing.T) {
	cfg := config.AuthConfig{
		RequireManagementAuth: true,
		ManagementKeys:        []string{"sk-management-test"},
	}
	middleware, err := NewAPIAuthMiddleware(cfg, nil)
	if err != nil {
		t.Fatalf("NewAPIAuthMiddleware() error = %v", err)
	}

	handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer sk-management-test")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, expected 200 for API-key mutation without Origin", recorder.Code)
	}
}

func TestOIDCLogoutRejectsCrossOrigin(t *testing.T) {
	h := &Handler{oidc: &OIDCService{
		Sessions: auth.NewSessionStore(time.Hour),
		cfg:      config.OIDCConfig{},
	}}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oidc/logout", nil)
	req.Header.Set("Origin", "http://sibling.example.net")

	recorder := httptest.NewRecorder()
	h.OIDCLogout().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, expected 403", recorder.Code)
	}
}
