package server

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"llamactl/pkg/config"
)

// fakeIdP is a minimal OpenID Connect provider: discovery, a JWKS endpoint,
// and a token endpoint that mints RS256 id_tokens. The nonce embedded in
// issued tokens is test-controlled so the callback's nonce check can be
// exercised both ways.
type fakeIdP struct {
	server   *httptest.Server
	key      *rsa.PrivateKey
	clientID string
	nonce    string
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating signing key: %v", err)
	}

	idp := &fakeIdP{key: key, clientID: "llamactl-test"}
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONTest(w, map[string]any{
			"issuer":                                idp.server.URL,
			"authorization_endpoint":                idp.server.URL + "/auth",
			"token_endpoint":                        idp.server.URL + "/token",
			"jwks_uri":                              idp.server.URL + "/keys",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		writeJSONTest(w, map[string]any{
			"keys": []map[string]any{{
				"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "test-key",
				"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e": "AQAB",
			}},
		})
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		token := signRS256Test(t, key, map[string]any{
			"iss":    idp.server.URL,
			"aud":    idp.clientID,
			"sub":    "user-1",
			"name":   "Alice",
			"email":  "alice@example.com",
			"groups": []string{"llamactl-admins"},
			"exp":    time.Now().Add(time.Hour).Unix(),
			"iat":    time.Now().Unix(),
			"nonce":  idp.nonce,
		})
		writeJSONTest(w, map[string]any{
			"access_token": "fake-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			"id_token":     token,
		})
	})

	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	return idp
}

func writeJSONTest(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// signRS256Test produces a minimal JWT; go-oidc verifies the signature
// against the JWKS endpoint above.
func signRS256Test(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]any{"alg": "RS256", "typ": "JWT", "kid": "test-key"})
	if err != nil {
		t.Fatalf("marshaling JWT header: %v", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshaling JWT claims: %v", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("signing JWT: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// runLogin drives the login handler and returns the query state, the browser
// state cookie, and the nonce bound into both.
func runLogin(t *testing.T, h *Handler) (state string, cookie *http.Cookie, nonce string) {
	t.Helper()

	rec := httptest.NewRecorder()
	h.OIDCLogin().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("login status = %d, expected 302", rec.Code)
	}

	redirect, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parsing login redirect: %v", err)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == stateCookieName {
			cookie = c
		}
	}
	if cookie == nil {
		t.Fatal("expected state cookie from login")
	}

	_, nonce, ok := verifyStateValue(h.oidc.stateKey, cookie.Value, redirect.Query().Get("state"))
	if !ok {
		t.Fatal("login state cookie does not verify")
	}
	return redirect.Query().Get("state"), cookie, nonce
}

// TestOIDCFlowWithFakeIdP walks the whole login flow against a real (fake)
// provider: discovery, code exchange, id_token verification, the nonce and
// group gate checks, and session creation.
func TestOIDCFlowWithFakeIdP(t *testing.T) {
	idp := newFakeIdP(t)

	newHandler := func(t *testing.T) *Handler {
		t.Helper()
		svc, err := NewOIDCService(config.AuthConfig{OIDC: config.OIDCConfig{
			IssuerURL:     idp.server.URL,
			ClientID:      idp.clientID,
			ClientSecret:  "test-secret",
			Scopes:        []string{"openid", "profile", "email"},
			GroupsClaim:   "groups",
			AllowedGroups: []string{"llamactl-admins"},
			SessionTTL:    time.Hour,
		}})
		if err != nil {
			t.Fatalf("NewOIDCService: %v", err)
		}
		return &Handler{oidc: svc}
	}

	runCallback := func(t *testing.T, h *Handler, state string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet,
			callbackPath+"?state="+url.QueryEscape(state)+"&code=fake-auth-code", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		h.OIDCCallback().ServeHTTP(rec, req)
		return rec
	}

	t.Run("happy path establishes a session", func(t *testing.T) {
		h := newHandler(t)
		state, cookie, nonce := runLogin(t, h)
		idp.nonce = nonce

		rec := runCallback(t, h, state, cookie)

		if loc := rec.Header().Get("Location"); loc != "/" {
			t.Fatalf("Location = %q, expected \"/\"", loc)
		}
		var sessionID string
		for _, c := range rec.Result().Cookies() {
			if c.Name == sessionCookieName {
				sessionID = c.Value
			}
		}
		if sessionID == "" {
			t.Fatal("expected a session cookie after successful login")
		}

		// The session must authenticate whoami and carry the user's claims.
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/whoami", nil)
		req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
		rec = httptest.NewRecorder()
		h.Whoami().ServeHTTP(rec, req)
		if body := rec.Body.String(); !containsAll(body,
			`"authenticated":true`, `"sub":"user-1"`, `"name":"Alice"`) {
			t.Errorf("unexpected whoami body: %s", body)
		}
	})

	t.Run("id_token with wrong nonce is rejected", func(t *testing.T) {
		h := newHandler(t)
		state, cookie, _ := runLogin(t, h)
		idp.nonce = "minted-for-someone-else"

		rec := runCallback(t, h, state, cookie)

		if loc := rec.Header().Get("Location"); loc != "/?auth_error="+authErrToken {
			t.Fatalf("Location = %q, expected \"/?auth_error=%s\"", loc, authErrToken)
		}
		for _, c := range rec.Result().Cookies() {
			if c.Name == sessionCookieName {
				t.Error("no session cookie may be set when the nonce does not match")
			}
		}
	})
}
