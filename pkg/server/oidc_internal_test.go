package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"llamactl/pkg/auth"
	"llamactl/pkg/config"
)

func TestSignVerifyStateValue(t *testing.T) {
	key := []byte("test-key")

	signed := signStateValue(key, "some-state", "some-verifier", "some-nonce")

	verifier, nonce, ok := verifyStateValue(key, signed, "some-state")
	if !ok {
		t.Fatal("expected valid state cookie to verify")
	}
	if verifier != "some-verifier" {
		t.Errorf("verifier = %q, expected %q", verifier, "some-verifier")
	}
	if nonce != "some-nonce" {
		t.Errorf("nonce = %q, expected %q", nonce, "some-nonce")
	}
}

func TestVerifyStateValueRejectsBadInput(t *testing.T) {
	key := []byte("test-key")
	signed := signStateValue(key, "some-state", "some-verifier", "some-nonce")

	tests := []struct {
		name        string
		cookieValue string
		queryState  string
	}{
		{"state mismatch", signed, "other-state"},
		{"tampered signature", "some-state|some-verifier|some-nonce|deadbeef", "some-state"},
		{"garbage cookie", "garbage", "some-state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, ok := verifyStateValue(key, tt.cookieValue, tt.queryState); ok {
				t.Error("expected verification to fail")
			}
		})
	}
}

func TestGroupsFromClaims(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		claim   string
		groups  []string
		present bool
		wantErr bool
	}{
		{"array of strings", `{"sub":"u1","groups":["admins","devs"]}`, "groups", []string{"admins", "devs"}, true, false},
		{"empty array", `{"groups":[]}`, "groups", []string{}, true, false},
		{"scalar string", `{"groups":"admins"}`, "groups", []string{"admins"}, true, false},
		{"claim absent", `{"sub":"u1","roles":["admins"]}`, "groups", nil, false, false},
		{"object shape", `{"groups":{"admins":true}}`, "groups", nil, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tt.payload), &raw); err != nil {
				t.Fatalf("fixture JSON invalid: %v", err)
			}

			groups, present, err := groupsFromClaims(raw, tt.claim)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error for a malformed claim")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if present != tt.present {
				t.Errorf("present = %v, expected %v", present, tt.present)
			}
			if !slices.Equal(groups, tt.groups) {
				t.Errorf("groups = %v, expected %v", groups, tt.groups)
			}
		})
	}
}

func TestAuthorizedForGroups(t *testing.T) {
	allow := []string{"llamactl-users", "admins"}

	if !authorizedForGroups(nil, nil) {
		t.Error("empty allowlist must admit everyone (gate disabled)")
	}
	if !authorizedForGroups(allow, []string{"devs", "admins"}) {
		t.Error("expected overlap to authorize")
	}
	if authorizedForGroups(allow, []string{"devs"}) {
		t.Error("expected disjoint groups to be denied")
	}
	if authorizedForGroups(allow, nil) {
		t.Error("expected missing/empty groups to be denied when the gate is on")
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
	if stateCookie.Path != callbackPath {
		t.Errorf("state cookie path = %q, expected %q", stateCookie.Path, callbackPath)
	}
	verifier, nonce, ok := verifyStateValue(s.stateKey, stateCookie.Value, redirect.Query().Get("state"))
	if !ok {
		t.Fatal("state cookie does not verify against the redirect's state parameter")
	}
	if verifier == "" {
		t.Error("expected a PKCE verifier bound into the state cookie")
	}
	// The nonce sent to the IdP must be the one the callback will demand in
	// the id_token.
	if got := redirect.Query().Get("nonce"); got == "" || got != nonce {
		t.Errorf("redirect nonce = %q, expected it to match the cookie-bound nonce %q", got, nonce)
	}
}

// Behind a subpath proxy the browser reaches the callback under the external
// base prefix, so the state cookie must be scoped to that path or the browser
// will not send it back and every login fails with auth_error=state.
func TestOIDCLoginStateCookiePathUnderSubpath(t *testing.T) {
	s := &OIDCService{
		stateKey: []byte("test-key"),
		cfg:      config.OIDCConfig{RedirectURL: "https://dash.example.com/llamactl" + callbackPath},
	}
	handler := &Handler{oidc: s}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	recorder := httptest.NewRecorder()
	handler.OIDCLogin().ServeHTTP(recorder, req)

	for _, c := range recorder.Result().Cookies() {
		if c.Name == stateCookieName {
			if want := "/llamactl" + callbackPath; c.Path != want {
				t.Errorf("state cookie path = %q, expected %q", c.Path, want)
			}
			return
		}
	}
	t.Fatal("expected state cookie to be set")
}

func TestOIDCCallbackRejectsBadRequests(t *testing.T) {
	h := &Handler{oidc: &OIDCService{
		stateKey: []byte("test-key"),
		Sessions: auth.NewSessionStore(time.Hour),
	}}

	tests := []struct {
		name       string
		target     string
		wantErrCod string
	}{
		{"IdP returned an error", "/api/v1/auth/oidc/callback?error=access_denied", authErrIDP},
		{"missing state parameter", "/api/v1/auth/oidc/callback?code=x", authErrState},
		{"no state cookie", "/api/v1/auth/oidc/callback?state=x&code=y", authErrState},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			recorder := httptest.NewRecorder()

			h.OIDCCallback().ServeHTTP(recorder, req)

			// Failures land the browser back in the login dialog with an
			// error code, not on a bare http.Error page.
			if recorder.Code != http.StatusFound {
				t.Fatalf("status = %d, expected 302", recorder.Code)
			}
			if loc := recorder.Header().Get("Location"); loc != "/?auth_error="+tt.wantErrCod {
				t.Errorf("Location = %q, expected \"/?auth_error=%s\"", loc, tt.wantErrCod)
			}
			for _, c := range recorder.Result().Cookies() {
				if c.Name == sessionCookieName {
					t.Error("no session cookie may be set on a failed login")
				}
			}
		})
	}
}

func TestWebRootPath(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		want        string
	}{
		{"derived from request", "", "/"},
		{"explicit root deployment", "https://dash.example.com" + callbackPath, "/"},
		{"subpath proxy", "https://dash.example.com/llamactl" + callbackPath, "/llamactl/"},
		{"override not ending in callback path", "https://dash.example.com/elsewhere", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OIDCService{cfg: config.OIDCConfig{RedirectURL: tt.redirectURL}}
			req := httptest.NewRequest(http.MethodGet, callbackPath, nil)
			if got := s.webRootPath(req); got != tt.want {
				t.Errorf("webRootPath() = %q, expected %q", got, tt.want)
			}
		})
	}
}

// Behind a subpath proxy the callback must send the browser back to the
// app's base path, not to the server root.
func TestOIDCCallbackRedirectHonorsSubpath(t *testing.T) {
	h := &Handler{oidc: &OIDCService{
		stateKey: []byte("test-key"),
		Sessions: auth.NewSessionStore(time.Hour),
		cfg:      config.OIDCConfig{RedirectURL: "https://dash.example.com/llamactl" + callbackPath},
	}}

	req := httptest.NewRequest(http.MethodGet, callbackPath+"?state=x&code=y", nil)
	recorder := httptest.NewRecorder()
	h.OIDCCallback().ServeHTTP(recorder, req)

	if loc := recorder.Header().Get("Location"); loc != "/llamactl/?auth_error=state" {
		t.Errorf("Location = %q, expected \"/llamactl/?auth_error=state\"", loc)
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
