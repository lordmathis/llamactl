# llamactl OIDC Authentication — Implementation Spec

**Version:** 1.1 (MVP + group gate)
**Status:** Approved design, ready for implementation

## Overview

Add OpenID Connect login to the llamactl WebUI so homelab users can authenticate against their existing IdP (Authentik, Keycloak, Authelia, etc.) instead of pasting management API keys into the browser. Humans log in via OIDC with a server-side session cookie; machines (curl, SDKs, node-to-node) continue using static API keys unchanged. The feature is strictly opt-in: no OIDC config, no behavior change.

Also included: remove API-key-via-query-parameter support (agreed security cleanup — keys in URLs leak into proxy and request logs).

## Tech Stack

- Backend: Go, chi router (existing), `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2` (new deps)
- Frontend: React 19 + Vite, existing `AuthContext`/`LoginDialog` (no router added)
- Storage: in-memory session store (no schema changes; `api_keys.user_id` and `instances.owner_user_id` already exist)

## Core Principles

1. **Opt-in and isolated.** Absent OIDC config = byte-for-byte current behavior. All new auth paths are additive alternatives, never replacements.
2. **Humans get sessions, machines get keys.** OIDC never applies to `/v1/*` inference or node-to-node auth.
3. **No RBAC.** Authenticated OIDC user = dashboard access (admin). The dashboard is admin-only today; role mapping is deferred until a real multi-user need exists.
4. **Boring session mechanics.** Server-side in-memory sessions, HMAC-signed short-lived state cookie, PKCE S256. Restart = re-login. Acceptable for a dashboard.
5. **Subpath/reverse-proxy safe.** The WebUI already uses `document.baseURI` for API calls; OIDC redirect handling must work behind subpath proxies (default callback URL derived from config or request, overridable).

---

## Phase 1: Cleanup + Config Foundation

### Backend

**Kill query-param API keys:**
- Remove the `r.URL.Query().Get("api_key")` branch from `extractAPIKey` (`pkg/server/middleware.go:209-212`).
- Grep docs/, Swagger annotations, and the WebUI for `api_key` query usage; update anything found.
- Breaking change — changelog entry: "API keys are no longer accepted via `?api_key=` query parameter; use `Authorization: Bearer` or `X-API-Key`."

**OIDC config types** (`pkg/config/types.go`, extend `AuthConfig`):

```go
type OIDCConfig struct {
    IssuerURL    string   // e.g. https://auth.example/application/o/llamactl/
    ClientID     string
    ClientSecret string
    RedirectURL  string   // optional; default derived from request
    Scopes       []string // default: ["openid", "profile", "email"]
    SessionTTL   time.Duration // default: 12h
    SecureCookie bool     // default: true; set false for plain-HTTP LAN deployments
}
```

- YAML: `auth.oidc.*`. Env vars follow the existing `LLAMACTL_*` scheme (`pkg/config/env.go`): `LLAMACTL_AUTH_OIDC_ISSUER_URL`, `_CLIENT_ID`, `_CLIENT_SECRET`, `_REDIRECT_URL`, `_SECURE_COOKIE`.
- Validation: OIDC considered *enabled* iff `IssuerURL != "" && ClientID != ""`. Warn (log) if enabled but `RequireManagementAuth == false` — enforcement stays orthogonal; document the recommended combo (`require_management_auth: true`) in configuration docs.
- **Sanitize `auth.oidc.client_secret`** out of `GET /api/v1/config` responses (follow the node `api_key` sanitization pattern in the config handler).

**Deliverable:** Query-param keys are gone; OIDC config parses, validates, round-trips through all layers (defaults/env/YAML), and is sanitized in API output.

---

## Phase 2: Backend OIDC Core

### Backend

**Routes — mounting order is the gotcha.** `SetupRouter` applies `ManagementAuthMiddleware` to the whole `/api/v1` group (`pkg/server/routes.go:38-40`). Mount the OIDC + whoami endpoints *outside* that middleware:

- `GET  /api/v1/auth/oidc/login` — 302 to IdP authorization endpoint. Generates CSRF `state` + PKCE verifier, stores state in a short-lived (≤5 min) signed cookie (`llamactl_oidc_state`, HttpOnly, SameSite=Lax).
- `GET  /api/v1/auth/oidc/callback` — verifies state cookie, exchanges code (+ PKCE), verifies ID token via `go-oidc` (`IDTokenVerifier` from discovery — fail fast at startup if `.well-known/openid-configuration` is unreachable, with an actionable log line). Creates server-side session, sets `llamactl_session` cookie (HttpOnly, Secure per config, SameSite=Lax, TTL per config), 302 → `/`.
- `POST /api/v1/auth/oidc/logout` — deletes server session, clears cookie. (IdP-initiated/RP-initiated logout is Phase 4 optional.)
- `GET  /api/v1/auth/whoami` — unauthenticated, returns:

```json
{"authenticated": false, "oidc_enabled": true, "user": null}
{"authenticated": true,  "oidc_enabled": true, "user": {"sub": "...", "name": "...", "email": "..."}}
```

**Session store:** `map[sessionID]*Session` + RWMutex + janitor goroutine (or lazy expiry on read — simpler, sufficient). Session: `{ID, Sub, Name, Email, ExpiresAt}`. Random 256-bit IDs via `crypto/rand`.

**Middleware — alternative path, not a rewrite:** `ManagementAuthMiddleware` becomes: valid management key (existing constant-time path) **or** valid `llamactl_session` cookie. On session, stash `user` in request context under a new context key. `RequireManagementAuth == false` keeps today's no-auth semantics.

**Redirect URL resolution:** if `RedirectURL` empty, derive from the request (`X-Forwarded-Proto`/`X-Forwarded-Host` respected) + `/api/v1/auth/oidc/callback`. Document that deployments behind TLS-terminating proxies without forwarded headers must set `redirect_url` explicitly.

**Wire up ownership (already-modeled fields):**
- `CreateKey` (`pkg/server/handlers_auth.go:143`): `UserID` = context user's `sub` when session-authed, `"system"` otherwise (key path unchanged). **Implemented.**
- `CreateInstance`: populate `instances.owner_user_id` from session user when present. **Deferred:** the column exists in the schema but is dead weight — the instance store never reads or writes it (`instanceToRow` ignores it), so wiring it means plumbing an owner field through `Instance` JSON, the manager, and the DB layer for a badge. Revisit when instance ownership has a consumer.

**CORS note:** current CORS sets `AllowCredentials: false` (`routes.go:25`). This is fine — the WebUI is same-origin (embedded + vite dev proxy). Split-origin WebUI deployments are out of scope; note it in docs.

**Tests:** callback rejects bad/missing state (CSRF), expired sessions fail whoami, middleware accepts key XOR session, config sanitization hides secret.

**Deliverable:** Complete browser flow works via curl/bruker-verifiable endpoints — login redirect, callback sets session, whoami reflects it, logout revokes; management API accepts sessions and keys.

---

## Phase 3: WebUI Integration

### Frontend

**AuthContext** (`webui/src/contexts/AuthContext.tsx`) — extend, don't replace:
- Boot sequence: sessionStorage key → validate as today; if none/invalid → `GET /api/v1/auth/whoami` (cookie rides along same-origin; no fetch changes needed). Session response ⇒ authenticated, store user info in state.
- New state: `user` (from whoami), `authMethod: 'key' | 'session'`.

**LoginDialog** (`webui/src/components/LoginDialog.tsx`):
- `oidc_enabled` ⇒ primary button "Sign in with SSO" → `window.location.assign(document.baseURI + "api/v1/auth/oidc/login")` (full page nav; backend drives the flow).
- Existing key input stays, collapsed behind a "Use an API key instead" disclosure. Both paths coexist.

**Logout** (`webui/src/components/Header.tsx`): session mode → `POST /api/v1/auth/oidc/logout` then clear state; key mode → current behavior.

**401 interceptor (fixes existing gap):** `apiCall` (`webui/src/lib/api.ts:13-67`) dispatches `window.dispatchEvent(new CustomEvent('llamactl:unauthorized'))` on 401; AuthContext listens → clears key/state → App falls back to LoginDialog. Today a mid-session 401 just strands contexts with error banners.

**Keys settings page:** show `user_id` column/badge on key list (distinguishes `system` vs named users).

**Deliverable:** A homelaber with Authentik clicks "Sign in with SSO", lands in the dashboard, logs out; API-key login still works.

---

## Phase 4: Polish + Ship

- Swagger annotations for the four new endpoints; regenerate `docs/` (swag pipeline).
- `docs/configuration.md`: new `auth.oidc` section with Authentik/Keycloak client-config walkthroughs; redirect-URL/proxy-header guidance; `secure_cookie` explanation for HTTP LANs.
- Optional (implement only if trivial via discovery): RP-initiated logout using `end_session_endpoint`.
- CHANGELOG: OIDC feature + query-param key removal (breaking).
- Manual test matrix: Chrome + Firefox, subpath reverse proxy, HTTP LAN with `secure_cookie: false`, IdP downtime mid-session (existing sessions survive; new logins fail with clear error).

**Deliverable:** Shippable, documented feature.

---

## Addendum (v1.1): Group Allowlist Gate

The MVP admitted every user the IdP authenticates. Real deployments point llamactl at an IdP that serves more users than should see the dashboard, so login is now optionally gated on group membership.

**Config:** `auth.oidc.allowed_groups` ([]string) + `auth.oidc.groups_claim` (default `"groups"`). Env: `LLAMACTL_AUTH_OIDC_ALLOWED_GROUPS`, `LLAMACTL_AUTH_OIDC_GROUPS_CLAIM`.

**Semantics:**
- `allowed_groups` empty → every authenticated user (MVP behavior, opt-in principle preserved).
- Otherwise the ID token's groups claim must intersect `allowed_groups` or login is denied with 403 before a session is created.
- Fail closed: missing, empty, or malformed claim denies. A malformed claim (non-array, non-string shape) is a 502 since it is an IdP misconfiguration, not an access decision.
- Enforcement is at login only — no per-request group checks. Group removal takes effect at session expiry (`session_ttl`, default 12h) or restart. The dashboard is still single-role: members are admins, everyone else is out. Group→role mapping remains out of scope.

**Explicitly rejected:** per-request group re-evaluation (config is startup-loaded; nothing to re-evaluate against), a userinfo-endpoint fallback for IdPs that cannot put groups in the ID token (add if a real deployment needs it), and email/sub allowlists as a second gate mechanism (one gate, managed where group membership already lives).

---

## Out of Scope (explicitly)

- Group→role mapping / RBAC (dashboard stays admin-only; the binary `allowed_groups` gate above is as far as this goes)
- Multiple IdPs; OIDC for `/v1/*` inference traffic; refresh tokens / silent renew (re-login on expiry)
- Forward-auth header trust (`Remote-User`) — footgun when exposed directly; native OIDC is the answer
- Remote-node authentication via OIDC (nodes stay on static keys)
- Instance ownership *UI* beyond badge display; per-user key management
- Sessions surviving restart (in-memory by design)

## Testing Notes

- Unit: middleware accept/reject matrix, state/CSRF negatives, session expiry, secret sanitization, `UserID` wiring.
- Integration-lite: `go-oidc` against a stub IdP handler table (authorize → code → token → verify) — no live IdP in CI.
- Manual: real Authentik or Keycloak container; the flows that bite are redirect-URL derivation and cookie flags behind proxies.
