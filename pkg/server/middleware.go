package server

import (
	"context"
	"crypto/subtle"
	"fmt"
	"llamactl/pkg/auth"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	apiKeyContextKey contextKey = "apiKey"
	userContextKey   contextKey = "user"
)

type APIAuthMiddleware struct {
	authStore             database.AuthStore
	requireInferenceAuth  bool
	requireManagementAuth bool
	managementKeys        map[string]bool // Config-based management keys
	sessions              *auth.SessionStore
	hmacSecret            []byte
	legacySem             chan struct{} // serializes Argon2 fallback scans
}

// NewAPIAuthMiddleware creates a new APIAuthMiddleware with the given configuration
func NewAPIAuthMiddleware(authCfg config.AuthConfig, authStore database.AuthStore) (*APIAuthMiddleware, error) {
	// Load management keys from config into managementKeys map
	managementKeys := make(map[string]bool)
	for _, key := range authCfg.ManagementKeys {
		managementKeys[key] = true
	}

	// Handle legacy auto-generation for management keys if none provided and auth is required
	var generated bool = false
	const banner = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

	if authCfg.RequireManagementAuth && len(authCfg.ManagementKeys) == 0 {
		key, err := auth.GenerateKey("llamactl-mgmt")
		if err != nil {
			return nil, fmt.Errorf("generating management key: %w", err)
		}
		managementKeys[key] = true
		generated = true
		fmt.Printf("%s\n⚠️  MANAGEMENT AUTHENTICATION REQUIRED\n%s\n", banner, banner)
		fmt.Printf("🔑  Generated Management API Key:\n\n    %s\n\n", key)
	}

	if generated {
		fmt.Printf("%s\n⚠️  IMPORTANT\n%s\n", banner, banner)
		fmt.Println("• This key is auto-generated and will change on restart")
		fmt.Println("• For production, add explicit keys to your configuration")
		fmt.Println("• Copy this key before it disappears from the terminal")
		fmt.Println("• Stdout may be persisted in container or service logs; treat access to startup logs as key access")
		fmt.Println(banner)
	}

	m := &APIAuthMiddleware{
		authStore:             authStore,
		requireInferenceAuth:  authCfg.RequireInferenceAuth,
		requireManagementAuth: authCfg.RequireManagementAuth,
		managementKeys:        managementKeys,
		legacySem:             make(chan struct{}, 1),
	}

	if authStore != nil {
		secret, err := authStore.GetOrCreateHMACKey(context.Background())
		if err != nil {
			return nil, fmt.Errorf("loading API key HMAC secret: %w", err)
		}
		m.hmacSecret = secret
	}

	return m, nil
}

// InferenceAuthMiddleware returns middleware for inference endpoints
func (a *APIAuthMiddleware) InferenceAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from request
			apiKey := a.extractAPIKey(r)
			if apiKey == "" {
				a.unauthorized(w, "Missing API key")
				return
			}

			// Try database authentication first
			var foundKey *auth.APIKey
			if a.requireInferenceAuth && a.authStore != nil {
				foundKey = a.verifyInferenceKey(r.Context(), apiKey)
			}

			// If no database key found, try management key authentication (config-based)
			if foundKey == nil {
				if !a.isValidManagementKey(apiKey) {
					a.unauthorized(w, "Invalid API key")
					return
				}
				// Management key was used, continue without adding APIKey to context
			} else {
				// Add APIKey to context for permission checking
				ctx := context.WithValue(r.Context(), apiKeyContextKey, foundKey)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// verifyInferenceKey resolves a token to a database key via one indexed
// fingerprint lookup, falling back to the serialized Argon2 scan for keys
// created before fingerprints existed.
func (a *APIAuthMiddleware) verifyInferenceKey(ctx context.Context, token string) *auth.APIKey {
	fingerprint := auth.KeyFingerprint(a.hmacSecret, token)

	key, err := a.authStore.GetKeyByHMAC(ctx, fingerprint)
	if err != nil {
		log.Printf("Failed to look up API key fingerprint: %v", err)
		return nil
	}
	if key != nil {
		a.touchKeyAsync(key.ID)
		return key
	}

	return a.verifyLegacyKey(ctx, token, fingerprint)
}

// verifyLegacyKey Argon2-scans pending legacy keys, one caller at a time; a
// successful match upgrades the key to fingerprint lookup permanently.
func (a *APIAuthMiddleware) verifyLegacyKey(ctx context.Context, token, fingerprint string) *auth.APIKey {
	a.legacySem <- struct{}{}
	defer func() { <-a.legacySem }()

	legacyKeys, err := a.authStore.GetPendingLegacyKeys(ctx)
	if err != nil {
		log.Printf("Failed to get legacy API keys: %v", err)
		return nil
	}

	for _, key := range legacyKeys {
		if !auth.VerifyKey(token, key.KeyHash) {
			continue
		}

		upCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.authStore.SetKeyHMAC(upCtx, key.ID, fingerprint); err != nil {
			log.Printf("Failed to upgrade legacy API key %d to fingerprint lookup: %v", key.ID, err)
		} else {
			log.Printf("API key %d (%s) upgraded to fingerprint verification", key.ID, key.Name)
		}

		a.touchKeyAsync(key.ID)
		return key
	}

	return nil
}

func (a *APIAuthMiddleware) touchKeyAsync(keyID int) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.authStore.TouchKey(ctx, keyID); err != nil {
			log.Printf("Failed to update last used timestamp for key %d: %v", keyID, err)
		}
	}()
}

// ManagementAuthMiddleware returns middleware for management endpoints.
// Requests are authenticated either by a valid OIDC session cookie or by a
// management API key.
func (a *APIAuthMiddleware) ManagementAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			if sess := sessionFromRequest(r, a.sessions); sess != nil {
				if !isSafeMethod(r.Method) && !sameOriginRequest(r) {
					forbidden(w, "Cross-origin request rejected")
					return
				}
				ctx := context.WithValue(r.Context(), userContextKey, sess)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Extract API key from request
			apiKey := a.extractAPIKey(r)
			if apiKey == "" {
				a.unauthorized(w, "Missing API key")
				return
			}

			// Check if key exists in managementKeys map using constant-time comparison
			if !a.isValidManagementKey(apiKey) {
				a.unauthorized(w, "Invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func sameOriginRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}

	// Browsers always send Origin or Referer on unsafe methods, so neither present is not a browser request.
	if origin == "" {
		return false
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}

	// The browser's Origin cannot be forged; proxies only change which host it must match.
	forwarded := strings.TrimSpace(strings.SplitN(r.Header.Get("X-Forwarded-Host"), ",", 2)[0])
	if forwarded != "" && strings.EqualFold(u.Host, forwarded) {
		return true
	}
	return strings.EqualFold(u.Host, r.Host)
}

// ResolveInstancePermission returns the effective permission for the authenticated
// key on the given instance. Management keys and allow_all keys short-circuit with a
// synthetic record (CanStart=true, CanEvict=true) and no DB query. Returns an error
// if a per_instance key has no matching permission row.
func (a *APIAuthMiddleware) ResolveInstancePermission(ctx context.Context, instanceID int) (*auth.KeyPermission, error) {
	// Extract APIKey from context
	apiKey, ok := ctx.Value(apiKeyContextKey).(*auth.APIKey)
	if !ok {
		// APIKey is nil, management key was used, allow all
		return &auth.KeyPermission{InstanceID: instanceID, CanStart: true, CanEvict: true}, nil
	}

	// If permission_mode == "allow_all", allow all
	if apiKey.PermissionMode == auth.PermissionModeAllowAll {
		return &auth.KeyPermission{KeyID: apiKey.ID, InstanceID: instanceID, CanStart: true, CanEvict: true}, nil
	}

	// Check per-instance permissions
	perm, err := a.authStore.GetInstancePermission(ctx, apiKey.ID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permission: %w", err)
	}

	if perm == nil {
		return nil, fmt.Errorf("permission denied: key does not have access to this instance")
	}

	return perm, nil
}

// CheckInstancePermission checks if the authenticated key has permission for the instance.
// It is a thin wrapper around ResolveInstancePermission for call sites that only need the
// access (allow/deny) decision.
func (a *APIAuthMiddleware) CheckInstancePermission(ctx context.Context, instanceID int) error {
	_, err := a.ResolveInstancePermission(ctx, instanceID)
	return err
}

// UserFromContext returns the OIDC session user, or nil when the request was
// authenticated with an API key.
func UserFromContext(ctx context.Context) *auth.Session {
	user, ok := ctx.Value(userContextKey).(*auth.Session)
	if !ok {
		return nil
	}
	return user
}

// extractAPIKey extracts the API key from the request
func (a *APIAuthMiddleware) extractAPIKey(r *http.Request) string {
	// Check Authorization header: "Bearer sk-..."
	if auth := r.Header.Get("Authorization"); auth != "" {
		if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
			return after
		}
	}

	// Check X-API-Key header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// isValidManagementKey checks if the provided API key is a valid management key
func (a *APIAuthMiddleware) isValidManagementKey(providedKey string) bool {
	for validKey := range a.managementKeys {
		if len(providedKey) == len(validKey) &&
			subtle.ConstantTimeCompare([]byte(providedKey), []byte(validKey)) == 1 {
			return true
		}
	}
	return false
}

// unauthorized sends an unauthorized response
func (a *APIAuthMiddleware) unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	response := fmt.Sprintf(`{"error": {"message": "%s", "type": "authentication_error"}}`, message)
	w.Write([]byte(response))
}

// forbidden sends a forbidden response
func forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	response := fmt.Sprintf(`{"error": {"message": "%s", "type": "permission_denied"}}`, message)
	w.Write([]byte(response))
}
