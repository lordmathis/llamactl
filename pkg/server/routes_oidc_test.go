package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"llamactl/pkg/auth"
	"llamactl/pkg/config"
)

// TestRoutesOIDCBypassManagementAuth verifies the router (not just the
// handlers) serves whoami and the OIDC endpoints without a management key.
func TestRoutesOIDCBypassManagementAuth(t *testing.T) {
	cfg := config.AppConfig{}
	cfg.Auth.RequireManagementAuth = true
	cfg.Auth.ManagementKeys = []string{"sk-test"}

	h := &Handler{cfg: cfg}
	h.authMiddleware = NewAPIAuthMiddleware(cfg.Auth, nil)
	h.oidc = &OIDCService{Sessions: auth.NewSessionStore(3600e9)}
	h.authMiddleware.sessions = h.oidc.Sessions

	r := SetupRouter(h)

	for _, target := range []string{
		"/api/v1/auth/whoami",
		"/api/v1/auth/oidc/login",
		"/api/v1/auth/oidc/callback",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code == http.StatusUnauthorized {
			t.Errorf("%s: got 401, expected the route to bypass management auth", target)
		}
	}

	// And that management endpoints still require auth through the router.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("/api/v1/instances: got %d, expected 401", rec.Code)
	}
}
