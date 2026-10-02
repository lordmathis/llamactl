package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"llamactl/pkg/auth"
	"llamactl/pkg/config"
)

func TestSignVerifyStateValue(t *testing.T) {
	key := []byte("test-key")

	signed := signStateValue(key, "some-state", "some-verifier")

	verifier, ok := verifyStateValue(key, signed, "some-state")
	if !ok {
		t.Fatal("expected valid state cookie to verify")
	}
	if verifier != "some-verifier" {
		t.Errorf("verifier = %q, expected %q", verifier, "some-verifier")
	}
}

func TestVerifyStateValueRejectsBadInput(t *testing.T) {
	key := []byte("test-key")
	signed := signStateValue(key, "some-state", "some-verifier")

	tests := []struct {
		name        string
		cookieValue string
		queryState  string
	}{
		{"state mismatch", signed, "other-state"},
		{"tampered signature", "some-state|some-verifier|deadbeef", "some-state"},
		{"garbage cookie", "garbage", "some-state"},
		{"empty cookie", "", "some-state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := verifyStateValue(key, tt.cookieValue, tt.queryState); ok {
				t.Error("expected verification to fail")
			}
		})
	}
}

func TestVerifyStateValueWrongKey(t *testing.T) {
	signed := signStateValue([]byte("key-one"), "some-state", "some-verifier")
	if _, ok := verifyStateValue([]byte("key-two"), signed, "some-state"); ok {
		t.Error("expected verification to fail under a different key")
	}
}

func newSessionMiddleware() (*APIAuthMiddleware, *auth.SessionStore) {
	middleware := NewAPIAuthMiddleware(config.AuthConfig{}, nil)
	store := auth.NewSessionStore(time.Hour)
	middleware.sessions = store
	return middleware, store
}

func TestManagementAuthMiddlewareAcceptsSession(t *testing.T) {
	middleware, store := newSessionMiddleware()
	sess := store.Create("user-1", "Alice", "alice@example.com")

	var gotUser *auth.Session
	handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, expected 200", recorder.Code)
	}
	if gotUser == nil || gotUser.Sub != "user-1" {
		t.Errorf("expected session user in context, got %+v", gotUser)
	}
}

func TestManagementAuthMiddlewareRejectsBogusSessionCookie(t *testing.T) {
	middleware, _ := newSessionMiddleware()

	handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler must not be reached")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "bogus"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, expected 401", recorder.Code)
	}
}

func TestManagementAuthMiddlewareKeyStillWorks(t *testing.T) {
	cfg := config.AuthConfig{
		RequireManagementAuth: true,
		ManagementKeys:        []string{"sk-management-test"},
	}
	middleware := NewAPIAuthMiddleware(cfg, nil)
	middleware.sessions = auth.NewSessionStore(time.Hour)

	handler := middleware.ManagementAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) != nil {
			t.Error("key auth must not populate the session user")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer sk-management-test")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, expected 200", recorder.Code)
	}
}

func TestWhoamiUnauthenticated(t *testing.T) {
	handler := &Handler{oidc: &OIDCService{Sessions: auth.NewSessionStore(time.Hour)}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
	recorder := httptest.NewRecorder()
	handler.Whoami().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, expected 200", recorder.Code)
	}
	if body := recorder.Body.String(); !containsAll(body,
		`"authenticated":false`, `"oidc_enabled":true`, `"user":null`) {
		t.Errorf("unexpected whoami body: %s", body)
	}
}

func TestWhoamiAuthenticatedSession(t *testing.T) {
	store := auth.NewSessionStore(time.Hour)
	handler := &Handler{oidc: &OIDCService{Sessions: store}}

	sess := store.Create("user-1", "Alice", "alice@example.com")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sess.ID})
	recorder := httptest.NewRecorder()
	handler.Whoami().ServeHTTP(recorder, req)

	if body := recorder.Body.String(); !containsAll(body,
		`"authenticated":true`, `"sub":"user-1"`, `"name":"Alice"`) {
		t.Errorf("unexpected whoami body: %s", body)
	}
}

func TestWhoamiWithoutOIDC(t *testing.T) {
	handler := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
	recorder := httptest.NewRecorder()
	handler.Whoami().ServeHTTP(recorder, req)

	if body := recorder.Body.String(); !containsAll(body,
		`"authenticated":false`, `"oidc_enabled":false`) {
		t.Errorf("unexpected whoami body: %s", body)
	}
}

func TestOIDCLoginBindsStateCookieToRedirect(t *testing.T) {
	s := &OIDCService{stateKey: []byte("test-key")}
	handler := &Handler{oidc: s}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	recorder := httptest.NewRecorder()
	handler.OIDCLogin().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, expected 302", recorder.Code)
	}

	redirect, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing redirect location: %v", err)
	}
	if state := redirect.Query().Get("state"); state == "" {
		t.Error("expected state parameter in redirect")
	}
	if method := redirect.Query().Get("code_challenge_method"); method != "S256" {
		t.Errorf("code_challenge_method = %q, expected S256", method)
	}

	var stateCookie *http.Cookie
	for _, c := range recorder.Result().Cookies() {
		if c.Name == stateCookieName {
			stateCookie = c
		}
	}
	if stateCookie == nil {
		t.Fatal("expected state cookie to be set")
	}
	if _, ok := verifyStateValue(s.stateKey, stateCookie.Value, redirect.Query().Get("state")); !ok {
		t.Error("state cookie does not verify against the redirect's state parameter")
	}
}

func TestOIDCCallbackRejectsBadRequests(t *testing.T) {
	h := &Handler{oidc: &OIDCService{
		stateKey: []byte("test-key"),
		Sessions: auth.NewSessionStore(time.Hour),
	}}
	signed := signStateValue([]byte("test-key"), "some-state", "some-verifier")

	tests := []struct {
		name    string
		target  string
		cookies []*http.Cookie
	}{
		{"IdP returned an error", "/api/v1/auth/oidc/callback?error=access_denied", nil},
		{"missing state parameter", "/api/v1/auth/oidc/callback?code=x", nil},
		{"missing code parameter", "/api/v1/auth/oidc/callback?state=x", nil},
		{"no state cookie", "/api/v1/auth/oidc/callback?state=x&code=y", nil},
		{"garbage state cookie", "/api/v1/auth/oidc/callback?state=x&code=y", []*http.Cookie{{Name: stateCookieName, Value: "garbage"}}},
		{"state does not match cookie", "/api/v1/auth/oidc/callback?state=other&code=y", []*http.Cookie{{Name: stateCookieName, Value: signed}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			for _, c := range tt.cookies {
				req.AddCookie(c)
			}
			recorder := httptest.NewRecorder()

			h.OIDCCallback().ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d, expected 400", recorder.Code)
			}
		})
	}
}

func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
