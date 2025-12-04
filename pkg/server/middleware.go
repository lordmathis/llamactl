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
	"os"
	"strings"
	"time"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	apiKeyContextKey contextKey = "apiKey"
)

type APIAuthMiddleware struct {
	authStore             database.AuthStore
	requireInferenceAuth  bool
	requireManagementAuth bool
	managementKeys        map[string]bool // Config-based management keys
}

// NewAPIAuthMiddleware creates a new APIAuthMiddleware with the given configuration
func NewAPIAuthMiddleware(authCfg config.AuthConfig, authStore database.AuthStore) *APIAuthMiddleware {
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
			log.Printf("Warning: Failed to generate management key: %v", err)
			// Fallback to PID-based key for safety
			key = fmt.Sprintf("sk-management-fallback-%d", os.Getpid())
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
		fmt.Println(banner)
	}

	return &APIAuthMiddleware{
		authStore:             authStore,
		requireInferenceAuth:  authCfg.RequireInferenceAuth,
		requireManagementAuth: authCfg.RequireManagementAuth,
		managementKeys:        managementKeys,
	}
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
				activeKeys, err := a.authStore.GetActiveKeys(r.Context())
				if err != nil {
					log.Printf("Failed to get active inference keys: %v", err)
					// Continue to management key fallback
				} else {
					for _, key := range activeKeys {
						if auth.VerifyKey(apiKey, key.KeyHash) {
							foundKey = key
							// Async update last_used_at
							go func(keyID int) {
								ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
								defer cancel()
								if err := a.authStore.TouchKey(ctx, keyID); err != nil {
									log.Printf("Failed to update last used timestamp for key %d: %v", keyID, err)
								}
							}(key.ID)
							break
						}
					}
				}
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

// ManagementAuthMiddleware returns middleware for management endpoints
func (a *APIAuthMiddleware) ManagementAuthMiddleware() func(http.Handler) http.Handler {
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

			// Check if key exists in managementKeys map using constant-time comparison
			if !a.isValidManagementKey(apiKey) {
				a.unauthorized(w, "Invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CheckInstancePermission checks if the authenticated key has permission for the instance
func (a *APIAuthMiddleware) CheckInstancePermission(ctx context.Context, instanceID int) error {
	// Extract APIKey from context
	apiKey, ok := ctx.Value(apiKeyContextKey).(*auth.APIKey)
	if !ok {
		// APIKey is nil, management key was used, allow all
		return nil
	}

	// If permission_mode == "allow_all", allow all
	if apiKey.PermissionMode == auth.PermissionModeAllowAll {
		return nil
	}

	// Check per-instance permissions
	canInfer, err := a.authStore.HasPermission(ctx, apiKey.ID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to check permission: %w", err)
	}

	if !canInfer {
		return fmt.Errorf("permission denied: key does not have access to this instance")
	}

	return nil
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

	// Check query parameter
	if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
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
func (a *APIAuthMiddleware) forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	response := fmt.Sprintf(`{"error": {"message": "%s", "type": "permission_denied"}}`, message)
	w.Write([]byte(response))
}
