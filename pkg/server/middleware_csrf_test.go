package server

import (
	"llamactl/pkg/auth"
	"llamactl/pkg/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestManagementAuthMiddlewareOriginCheck(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		origin         string
		referer        string
		forwardedHost  string
		requestHost    string
		expectedStatus int
	}{
		{
			name:           "same-origin Origin passes",
			method:         http.MethodPost,
			origin:         "http://example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "sibling origin is rejected",
			method:         http.MethodPost,
			origin:         "http://sibling.example.net",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Origin null is rejected",
			method:         http.MethodPost,
			origin:         "null",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "missing Origin and Referer is rejected",
			method:         http.MethodPost,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "matching Referer passes when Origin absent",
			method:         http.MethodPost,
			referer:        "http://example.com/instances",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Origin matching X-Forwarded-Host passes",
			method:         http.MethodPost,
			origin:         "http://llamactl.example.com",
			forwardedHost:  "llamactl.example.com",
			requestHost:    "internal:8080",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET without Origin passes",
			method:         http.MethodGet,
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
			req := httptest.NewRequest(tt.method, "/api/v1/instances/x/stop", nil)
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
			if tt.requestHost != "" {
				req.Host = tt.requestHost
			}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			if tt.forwardedHost != "" {
				req.Header.Set("X-Forwarded-Host", tt.forwardedHost)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("status = %d, expected %d", recorder.Code, tt.expectedStatus)
			}
			if tt.expectedStatus == http.StatusForbidden && !strings.Contains(recorder.Body.String(), `"type": "permission_denied"`) {
				t.Errorf("forbidden response missing error type: %s", recorder.Body.String())
			}
		})
	}
}

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

func TestOIDCLogoutOriginCheck(t *testing.T) {
	h := &Handler{oidc: &OIDCService{
		Sessions: auth.NewSessionStore(time.Hour),
		cfg:      config.OIDCConfig{},
	}}
	handler := h.OIDCLogout()

	tests := []struct {
		name           string
		origin         string
		expectedStatus int
	}{
		{
			name:           "same-origin Origin logs out",
			origin:         "http://example.com",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "cross-origin Origin is rejected",
			origin:         "http://sibling.example.net",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oidc/logout", nil)
			req.Header.Set("Origin", tt.origin)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf("status = %d, expected %d", recorder.Code, tt.expectedStatus)
			}
		})
	}
}
