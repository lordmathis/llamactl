package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"llamactl/pkg/auth"
	"llamactl/pkg/config"
)

const (
	sessionCookieName = "llamactl_session"
	stateCookieName   = "llamactl_oidc_state"
	callbackPath      = "/api/v1/auth/oidc/callback"
	stateCookieTTL    = 5 * time.Minute
)

// OIDCService implements browser login against an OpenID Connect provider.
// API keys remain the only way to authenticate non-browser clients.
type OIDCService struct {
	verifier *oidc.IDTokenVerifier
	oauth2   oauth2.Config
	cfg      config.OIDCConfig
	stateKey []byte

	Sessions *auth.SessionStore
}

// NewOIDCService performs IdP discovery eagerly so a misconfigured issuer
// fails at startup instead of at first login.
func NewOIDCService(authCfg config.AuthConfig) (*OIDCService, error) {
	cfg := authCfg.OIDC
	if !cfg.Enabled() {
		return nil, fmt.Errorf("oidc is not configured: issuer_url and client_id are required")
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 12 * time.Hour
	}
	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{"openid", "profile", "email"}
	}
	if cfg.GroupsClaim == "" {
		cfg.GroupsClaim = "groups"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("discovering OIDC provider at %s: %w "+
			"(check that the issuer URL is reachable and serves .well-known/openid-configuration)", cfg.IssuerURL, err)
	}

	stateKey := make([]byte, 32)
	if _, err := rand.Read(stateKey); err != nil {
		return nil, fmt.Errorf("generating OIDC state key: %w", err)
	}

	return &OIDCService{
		verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		oauth2: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			Scopes:       cfg.Scopes,
		},
		cfg:      cfg,
		stateKey: stateKey,
		Sessions: auth.NewSessionStore(cfg.SessionTTL),
	}, nil
}

// redirectURL resolves the OAuth2 callback URL for this request, preferring
// the configured override and otherwise deriving it from forwarded headers.
func (s *OIDCService) redirectURL(r *http.Request) string {
	if s.cfg.RedirectURL != "" {
		return s.cfg.RedirectURL
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}

	return fmt.Sprintf("%s://%s%s", scheme, host, callbackPath)
}

// signStateValue binds the CSRF state and the PKCE verifier together in a
// cookie value the callback can trust without server-side storage.
func signStateValue(key []byte, state, verifier string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(state))
	mac.Write([]byte("|"))
	mac.Write([]byte(verifier))
	return state + "|" + verifier + "|" + hex.EncodeToString(mac.Sum(nil))
}

// verifyStateValue checks the signed cookie against the state the IdP echoed
// back and returns the PKCE verifier for the code exchange.
func verifyStateValue(key []byte, cookieValue, queryState string) (verifier string, ok bool) {
	parts := strings.Split(cookieValue, "|")
	if len(parts) != 3 {
		return "", false
	}

	state, verifier := parts[0], parts[1]
	if state != queryState {
		return "", false
	}

	expected := signStateValue(key, state, verifier)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(cookieValue)) != 1 {
		return "", false
	}

	return verifier, true
}

// sessionFromRequest returns the session for the request's session cookie,
// or nil when there is no store, no cookie, or no valid session.
func sessionFromRequest(r *http.Request, store *auth.SessionStore) *auth.Session {
	if store == nil {
		return nil
	}
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	return store.Get(c.Value)
}

// OIDCLogin godoc
// @Summary Start OIDC login
// @Description Redirects the browser to the configured OpenID Connect provider to begin the login flow
// @Tags Auth
// @Produce html
// @Success 302 {string} string "Redirect to IdP"
// @Router /api/v1/auth/oidc/login [get]
func (h *Handler) OIDCLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := h.oidc

		state := auth.RandomToken()
		verifier := auth.RandomToken()

		oc := s.oauth2
		oc.RedirectURL = s.redirectURL(r)
		authURL := oc.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

		http.SetCookie(w, &http.Cookie{
			Name:     stateCookieName,
			Value:    signStateValue(s.stateKey, state, verifier),
			Path:     callbackPath,
			MaxAge:   int(stateCookieTTL.Seconds()),
			HttpOnly: true,
			Secure:   s.cfg.SecureCookie,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// groupsFromClaims extracts the user's groups from raw ID-token claims.
// present distinguishes "claim not in token" from "claim present but empty".
// A scalar string is accepted and wrapped; any other shape is an error so a
// misconfigured claim fails closed instead of silently matching nothing.
func groupsFromClaims(raw map[string]json.RawMessage, claim string) (groups []string, present bool, err error) {
	v, ok := raw[claim]
	if !ok {
		return nil, false, nil
	}
	if err := json.Unmarshal(v, &groups); err == nil {
		return groups, true, nil
	}
	var single string
	if err := json.Unmarshal(v, &single); err == nil {
		return []string{single}, true, nil
	}
	return nil, true, fmt.Errorf("groups claim %q is not an array of strings: %s", claim, v)
}

// tokenGroups reads the configured groups claim plus the names of all claims
// the token carries, so a denied login can be logged with enough context to
// spot a wrong groups_claim setting.
func (s *OIDCService) tokenGroups(idToken *oidc.IDToken) (groups, claimNames []string, err error) {
	var raw map[string]json.RawMessage
	if err := idToken.Claims(&raw); err != nil {
		return nil, nil, fmt.Errorf("parsing id_token claims: %w", err)
	}
	groups, _, err = groupsFromClaims(raw, s.cfg.GroupsClaim)
	return groups, slices.Sorted(maps.Keys(raw)), err
}

// authorizedForGroups reports whether the token's groups satisfy the
// allowed_groups gate; an empty allowlist admits every authenticated user.
func authorizedForGroups(allowed, tokenGroups []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, g := range tokenGroups {
		if slices.Contains(allowed, g) {
			return true
		}
	}
	return false
}

// OIDC callback failure codes, surfaced to the WebUI as ?auth_error=<code>
// on the app root. The SPA maps them to messages in the login dialog.
const (
	authErrDenied   = "denied"   // token groups do not intersect allowed_groups
	authErrState    = "state"    // missing, invalid, or expired CSRF state
	authErrIDP      = "idp"      // the IdP itself reported an OAuth error
	authErrExchange = "exchange" // authorization code exchange failed
	authErrToken    = "token"    // id_token missing, unverifiable, or unreadable
	authErrGroups   = "groups"   // groups claim present but malformed
)

// redirectAuthError sends the browser back to the WebUI root with an error
// code, where the login dialog renders it in the app's own style instead of
// a bare http.Error page. Details stay in the server log.
func redirectAuthError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/?auth_error="+code, http.StatusFound)
}

// OIDCCallback godoc
// @Summary Complete OIDC login
// @Description Handles the IdP redirect, verifies state and ID token, and establishes a session cookie
// @Tags Auth
// @Produce html
// @Success 302 {string} string "Redirect to the WebUI root with an established session, or with ?auth_error=<denied|state|idp|exchange|token|groups> on failure"
// @Router /api/v1/auth/oidc/callback [get]
func (h *Handler) OIDCCallback() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := h.oidc

		if errParam := r.URL.Query().Get("error"); errParam != "" {
			log.Printf("OIDC login failed: IdP returned error %q", errParam)
			redirectAuthError(w, r, authErrIDP)
			return
		}

		queryState := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		if queryState == "" || code == "" {
			redirectAuthError(w, r, authErrState)
			return
		}

		stateCookie, err := r.Cookie(stateCookieName)
		if err != nil {
			redirectAuthError(w, r, authErrState)
			return
		}

		verifier, ok := verifyStateValue(s.stateKey, stateCookie.Value, queryState)
		if !ok {
			redirectAuthError(w, r, authErrState)
			return
		}

		// The state cookie is one-shot; clear it regardless of outcome.
		http.SetCookie(w, &http.Cookie{
			Name:     stateCookieName,
			Value:    "",
			Path:     callbackPath,
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   s.cfg.SecureCookie,
		})

		oc := s.oauth2
		oc.RedirectURL = s.redirectURL(r)

		ctx := r.Context()
		token, err := oc.Exchange(ctx, code, oauth2.VerifierOption(verifier))
		if err != nil {
			log.Printf("OIDC code exchange failed: %v", err)
			redirectAuthError(w, r, authErrExchange)
			return
		}

		rawIDToken, ok := token.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			log.Printf("OIDC token response missing id_token (requested scopes: %v)", s.cfg.Scopes)
			redirectAuthError(w, r, authErrToken)
			return
		}

		idToken, err := s.verifier.Verify(ctx, rawIDToken)
		if err != nil {
			log.Printf("OIDC id_token verification failed: %v", err)
			redirectAuthError(w, r, authErrToken)
			return
		}

		var claims struct {
			Sub   string `json:"sub"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := idToken.Claims(&claims); err != nil {
			log.Printf("OIDC id_token claims parsing failed: %v", err)
			redirectAuthError(w, r, authErrToken)
			return
		}

		if len(s.cfg.AllowedGroups) > 0 {
			groups, claimNames, err := s.tokenGroups(idToken)
			if err != nil {
				log.Printf("OIDC login failed for %s: %v", claims.Sub, err)
				redirectAuthError(w, r, authErrGroups)
				return
			}
			if !authorizedForGroups(s.cfg.AllowedGroups, groups) {
				log.Printf("OIDC login denied for %s: token groups %v (claim %q, token claims: %v), allowed_groups %v",
					claims.Sub, groups, s.cfg.GroupsClaim, claimNames, s.cfg.AllowedGroups)
				redirectAuthError(w, r, authErrDenied)
				return
			}
		}

		sess := s.Sessions.Create(claims.Sub, claims.Name, claims.Email)
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    sess.ID,
			Path:     "/",
			MaxAge:   int(s.cfg.SessionTTL.Seconds()),
			HttpOnly: true,
			Secure:   s.cfg.SecureCookie,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// OIDCLogout godoc
// @Summary End OIDC session
// @Description Revokes the server-side session and clears the session cookie
// @Tags Auth
// @Success 204 "Session terminated"
// @Router /api/v1/auth/oidc/logout [post]
func (h *Handler) OIDCLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil {
			h.oidc.Sessions.Delete(c.Value)
		}

		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.oidc.cfg.SecureCookie,
			SameSite: http.SameSiteLaxMode,
		})

		w.WriteHeader(http.StatusNoContent)
	}
}

// WhoamiUser describes the authenticated user in whoami responses.
type WhoamiUser struct {
	Sub   string `json:"sub"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// WhoamiResponse reports the caller's authentication state so the WebUI can
// decide between session reuse, SSO login, and API-key login.
type WhoamiResponse struct {
	Authenticated bool        `json:"authenticated"`
	OIDCEnabled   bool        `json:"oidc_enabled"`
	User          *WhoamiUser `json:"user"`
}

// Whoami godoc
// @Summary Get current authentication state
// @Description Returns whether the caller has an OIDC session and whether OIDC login is configured. Intended for the WebUI bootstrap; never requires authentication.
// @Tags Auth
// @Produce json
// @Success 200 {object} WhoamiResponse "Authentication state"
// @Router /api/v1/auth/whoami [get]
func (h *Handler) Whoami() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := WhoamiResponse{OIDCEnabled: h.oidc != nil}

		if h.oidc != nil {
			if sess := sessionFromRequest(r, h.oidc.Sessions); sess != nil {
				resp.Authenticated = true
				resp.User = &WhoamiUser{Sub: sess.Sub, Name: sess.Name, Email: sess.Email}
			}
		}

		writeJSON(w, http.StatusOK, resp)
	}
}
