package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"llamactl/pkg/config"
	"log"
	"net/http"
	"os"
	"strings"
)

type KeyType int

const (
	KeyTypeInference KeyType = iota
	KeyTypeManagement
)

type APIAuthMiddleware struct {
	requireInferenceAuth  bool
	inferenceKeys         map[string]bool
	requireManagementAuth bool
	managementKeys        map[string]bool
}

// NewAPIAuthMiddleware creates a new APIAuthMiddleware with the given configuration
func NewAPIAuthMiddleware(authCfg config.AuthConfig) *APIAuthMiddleware {

	var generated bool = false

	inferenceAPIKeys := make(map[string]bool)
	managementAPIKeys := make(map[string]bool)

	const banner = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

	if authCfg.RequireManagementAuth && len(authCfg.ManagementKeys) == 0 {
		key := generateAPIKey(KeyTypeManagement)
		managementAPIKeys[key] = true
		generated = true
		fmt.Printf("%s\n⚠️  MANAGEMENT AUTHENTICATION REQUIRED\n%s\n", banner, banner)
		fmt.Printf("🔑  Generated Management API Key:\n\n    %s\n\n", key)
	}
	for _, key := range authCfg.ManagementKeys {
		managementAPIKeys[key] = true
	}

	if authCfg.RequireInferenceAuth && len(authCfg.InferenceKeys) == 0 {
		key := generateAPIKey(KeyTypeInference)
		inferenceAPIKeys[key] = true
		generated = true
		fmt.Printf("%s\n⚠️  INFERENCE AUTHENTICATION REQUIRED\n%s\n", banner, banner)
		fmt.Printf("🔑  Generated Inference API Key:\n\n    %s\n\n", key)
	}
	for _, key := range authCfg.InferenceKeys {
		inferenceAPIKeys[key] = true
	}

	if generated {
		fmt.Printf("%s\n⚠️  IMPORTANT\n%s\n", banner, banner)
		fmt.Println("• These keys are auto-generated and will change on restart")
		fmt.Println("• For production, add explicit keys to your configuration")
		fmt.Println("• Copy these keys before they disappear from the terminal")
		fmt.Println(banner)
	}

	return &APIAuthMiddleware{
		requireInferenceAuth:  authCfg.RequireInferenceAuth,
		inferenceKeys:         inferenceAPIKeys,
		requireManagementAuth: authCfg.RequireManagementAuth,
		managementKeys:        managementAPIKeys,
	}
}

// generateAPIKey creates a cryptographically secure API key
func generateAPIKey(keyType KeyType) string {
	// Generate 32 random bytes (256 bits)
	randomBytes := make([]byte, 32)

	var prefix string

	switch keyType {
	case KeyTypeInference:
		prefix = "sk-inference"
	case KeyTypeManagement:
		prefix = "sk-management"
	default:
		prefix = "sk-unknown"
	}

	if _, err := rand.Read(randomBytes); err != nil {
		log.Printf("Warning: Failed to generate secure random key, using fallback")
		// Fallback to a less secure method if crypto/rand fails
		return fmt.Sprintf("%s-fallback-%d", prefix, os.Getpid())
	}

	// Convert to hex and add prefix
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(randomBytes))
}

// AuthMiddleware returns a middleware that checks API keys for the given key type
func (a *APIAuthMiddleware) AuthMiddleware(keyType KeyType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := a.extractAPIKey(r)
			if apiKey == "" {
				a.unauthorized(w, "Missing API key")
				return
			}

			var isValid bool
			switch keyType {
			case KeyTypeInference:
				// Management keys also work for OpenAI endpoints (higher privilege)
				isValid = a.isValidKey(apiKey, KeyTypeInference) || a.isValidKey(apiKey, KeyTypeManagement)
			case KeyTypeManagement:
				isValid = a.isValidKey(apiKey, KeyTypeManagement)
			default:
				isValid = false
			}

			if !isValid {
				a.unauthorized(w, "Invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
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

// isValidKey checks if the provided API key is valid for the given key type
func (a *APIAuthMiddleware) isValidKey(providedKey string, keyType KeyType) bool {
	var validKeys map[string]bool

	switch keyType {
	case KeyTypeInference:
		validKeys = a.inferenceKeys
	case KeyTypeManagement:
		validKeys = a.managementKeys
	default:
		return false
	}

	for validKey := range validKeys {
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
