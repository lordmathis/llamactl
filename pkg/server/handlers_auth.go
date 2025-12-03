package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/auth"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type InstancePermission struct {
	InstanceID  int  `json:"instance_id"`
	CanInfer    bool `json:"can_infer"`
	CanViewLogs bool `json:"can_view_logs"`
}

type CreateKeyRequest struct {
	Name                string
	PermissionMode      auth.PermissionMode
	ExpiresAt           *int64
	InstancePermissions []InstancePermission
}

type CreateKeyResponse struct {
	ID             int                     `json:"id"`
	Name           string                  `json:"name"`
	UserID         string                  `json:"user_id"`
	PermissionMode auth.PermissionMode     `json:"permission_mode"`
	ExpiresAt      *int64                  `json:"expires_at"`
	Enabled        bool                    `json:"enabled"`
	CreatedAt      int64                   `json:"created_at"`
	UpdatedAt      int64                   `json:"updated_at"`
	LastUsedAt     *int64                  `json:"last_used_at"`
	Key            string                  `json:"key"`
}

type KeyResponse struct {
	ID             int                     `json:"id"`
	Name           string                  `json:"name"`
	UserID         string                  `json:"user_id"`
	PermissionMode auth.PermissionMode     `json:"permission_mode"`
	ExpiresAt      *int64                  `json:"expires_at"`
	Enabled        bool                    `json:"enabled"`
	CreatedAt      int64                   `json:"created_at"`
	UpdatedAt      int64                   `json:"updated_at"`
	LastUsedAt     *int64                  `json:"last_used_at"`
}

type KeyPermissionResponse struct {
	InstanceID   int    `json:"instance_id"`
	InstanceName string `json:"instance_name"`
	CanInfer     bool   `json:"can_infer"`
	CanViewLogs  bool   `json:"can_view_logs"`
}

// CreateInferenceKey handles POST /api/v1/keys
func (h *Handler) CreateInferenceKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON in request body")
			return
		}

		// Validate request
		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_name", "Name is required")
			return
		}
		if len(req.Name) > 100 {
			writeError(w, http.StatusBadRequest, "invalid_name", "Name must be 100 characters or less")
			return
		}
		if req.PermissionMode != auth.PermissionModeAllowAll && req.PermissionMode != auth.PermissionModePerInstance {
			writeError(w, http.StatusBadRequest, "invalid_permission_mode", "Permission mode must be 'allow_all' or 'per_instance'")
			return
		}
		if req.PermissionMode == auth.PermissionModePerInstance && len(req.InstancePermissions) == 0 {
			writeError(w, http.StatusBadRequest, "missing_permissions", "Instance permissions required when permission mode is 'per_instance'")
			return
		}
		if req.ExpiresAt != nil && *req.ExpiresAt <= time.Now().Unix() {
			writeError(w, http.StatusBadRequest, "invalid_expires_at", "Expiration time must be in future")
			return
		}

		// Validate instance IDs exist
		if req.PermissionMode == auth.PermissionModePerInstance {
			instances, err := h.InstanceManager.ListInstances()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "fetch_instances_failed", fmt.Sprintf("Failed to fetch instances: %v", err))
				return
			}
			instanceIDMap := make(map[int]bool)
			for _, inst := range instances {
				instanceIDMap[inst.ID] = true
			}

			for _, perm := range req.InstancePermissions {
				if !instanceIDMap[perm.InstanceID] {
					writeError(w, http.StatusBadRequest, "invalid_instance_id", fmt.Sprintf("Instance ID %d does not exist", perm.InstanceID))
					return
				}
			}
		}

		// Generate plain-text key
		plainTextKey, err := auth.GenerateKey("llamactl-")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "key_generation_failed", "Failed to generate API key")
			return
		}

		// Hash key
		keyHash, err := auth.HashKey(plainTextKey)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "key_hashing_failed", "Failed to hash API key")
			return
		}

		// Create APIKey struct
		now := time.Now().Unix()
		apiKey := &auth.APIKey{
			KeyHash:        keyHash,
			Name:           req.Name,
			UserID:         "system",
			PermissionMode: req.PermissionMode,
			ExpiresAt:      req.ExpiresAt,
			Enabled:        true,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		// Convert InstancePermissions to KeyPermissions
		var keyPermissions []auth.KeyPermission
		for _, perm := range req.InstancePermissions {
			keyPermissions = append(keyPermissions, auth.KeyPermission{
				KeyID:       0, // Will be set by database after key creation
				InstanceID:  perm.InstanceID,
				CanInfer:    perm.CanInfer,
				CanViewLogs: perm.CanViewLogs,
			})
		}

		// Create in database
		err = h.authStore.CreateKey(r.Context(), apiKey, keyPermissions)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "creation_failed", fmt.Sprintf("Failed to create API key: %v", err))
			return
		}

		// Return response with plain-text key (only shown once)
		response := CreateKeyResponse{
			ID:             apiKey.ID,
			Name:           apiKey.Name,
			UserID:         apiKey.UserID,
			PermissionMode: apiKey.PermissionMode,
			ExpiresAt:      apiKey.ExpiresAt,
			Enabled:        apiKey.Enabled,
			CreatedAt:      apiKey.CreatedAt,
			UpdatedAt:      apiKey.UpdatedAt,
			LastUsedAt:     apiKey.LastUsedAt,
			Key:            plainTextKey,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// ListInferenceKeys handles GET /api/v1/keys
func (h *Handler) ListInferenceKeys() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keys, err := h.authStore.GetUserKeys(r.Context(), "system")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "fetch_failed", fmt.Sprintf("Failed to fetch API keys: %v", err))
			return
		}

		// Remove key_hash from all keys
		response := make([]KeyResponse, 0, len(keys))
		for _, key := range keys {
			response = append(response, KeyResponse{
				ID:             key.ID,
				Name:           key.Name,
				UserID:         key.UserID,
				PermissionMode: key.PermissionMode,
				ExpiresAt:      key.ExpiresAt,
				Enabled:        key.Enabled,
				CreatedAt:      key.CreatedAt,
				UpdatedAt:      key.UpdatedAt,
				LastUsedAt:     key.LastUsedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetInferenceKey handles GET /api/v1/keys/{id}
func (h *Handler) GetInferenceKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "Invalid key ID")
			return
		}

		key, err := h.authStore.GetKeyByID(r.Context(), id)
		if err != nil {
			if err.Error() == "API key not found" {
				writeError(w, http.StatusNotFound, "not_found", "API key not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "fetch_failed", fmt.Sprintf("Failed to fetch API key: %v", err))
			return
		}

		// Remove key_hash from response
		response := KeyResponse{
			ID:             key.ID,
			Name:           key.Name,
			UserID:         key.UserID,
			PermissionMode: key.PermissionMode,
			ExpiresAt:      key.ExpiresAt,
			Enabled:        key.Enabled,
			CreatedAt:      key.CreatedAt,
			UpdatedAt:      key.UpdatedAt,
			LastUsedAt:     key.LastUsedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// DeleteInferenceKey handles DELETE /api/v1/keys/{id}
func (h *Handler) DeleteInferenceKey() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "Invalid key ID")
			return
		}

		err = h.authStore.DeleteKey(r.Context(), id)
		if err != nil {
			if err.Error() == "API key not found" {
				writeError(w, http.StatusNotFound, "not_found", "API key not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "deletion_failed", fmt.Sprintf("Failed to delete API key: %v", err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetInferenceKeyPermissions handles GET /api/v1/keys/{id}/permissions
func (h *Handler) GetInferenceKeyPermissions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_id", "Invalid key ID")
			return
		}

		// Verify key exists
		_, err = h.authStore.GetKeyByID(r.Context(), id)
		if err != nil {
			if err.Error() == "API key not found" {
				writeError(w, http.StatusNotFound, "not_found", "API key not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "fetch_failed", fmt.Sprintf("Failed to fetch API key: %v", err))
			return
		}

		permissions, err := h.authStore.GetPermissions(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "fetch_failed", fmt.Sprintf("Failed to fetch permissions: %v", err))
			return
		}

		// Get instance names for the permissions
		instances, err := h.InstanceManager.ListInstances()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "fetch_instances_failed", fmt.Sprintf("Failed to fetch instances: %v", err))
			return
		}
		instanceNameMap := make(map[int]string)
		for _, inst := range instances {
			instanceNameMap[inst.ID] = inst.Name
		}

		response := make([]KeyPermissionResponse, 0, len(permissions))
		for _, perm := range permissions {
			response = append(response, KeyPermissionResponse{
				InstanceID:   perm.InstanceID,
				InstanceName: instanceNameMap[perm.InstanceID],
				CanInfer:     perm.CanInfer,
				CanViewLogs:  perm.CanViewLogs,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
