package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"llamactl/pkg/validation"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// errorResponse represents an error response returned by the API
type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// writeError writes a JSON error response with the specified HTTP status code
func writeError(w http.ResponseWriter, status int, code, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: code, Details: details}); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// writeJSON writes a JSON response with the specified HTTP status code
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

// writeText writes a plain text response with the specified HTTP status code
func writeText(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(data)); err != nil {
		log.Printf("Failed to write text response: %v", err)
	}
}

// Handler provides HTTP handlers for the llamactl server API
type Handler struct {
	InstanceManager manager.InstanceManager
	cfg             config.AppConfig
	httpClient      *http.Client
	authStore       database.AuthStore
	authMiddleware  *APIAuthMiddleware
}

// NewHandler creates a new Handler instance with the provided instance manager and configuration
func NewHandler(im manager.InstanceManager, cfg config.AppConfig, authStore database.AuthStore) *Handler {
	handler := &Handler{
		InstanceManager: im,
		cfg:             cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authStore: authStore,
	}
	handler.authMiddleware = NewAPIAuthMiddleware(cfg.Auth, authStore)
	return handler
}

// getInstance retrieves an instance by name from request query parameters
func (h *Handler) getInstance(r *http.Request) (*instance.Instance, error) {
	name := chi.URLParam(r, "name")
	validatedName, err := validation.ValidateInstanceName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid instance name: %w", err)
	}

	inst, err := h.InstanceManager.GetInstance(validatedName)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance by name: %w", err)
	}

	return inst, nil
}

// ensureInstanceRunning ensures that an instance is running by starting it if on-demand start is enabled
// It handles LRU eviction when the maximum number of running instances is reached
func (h *Handler) ensureInstanceRunning(inst *instance.Instance) error {
	options := inst.GetOptions()
	allowOnDemand := options != nil && options.OnDemandStart != nil && *options.OnDemandStart
	if !allowOnDemand {
		return fmt.Errorf("instance is not running and on-demand start is not enabled")
	}

	if h.InstanceManager.AtMaxRunning() {
		if h.cfg.Instances.EnableLRUEviction {
			err := h.InstanceManager.EvictLRUInstance()
			if err != nil {
				return fmt.Errorf("cannot start instance, failed to evict instance: %w", err)
			}
		} else {
			return fmt.Errorf("cannot start instance, maximum number of instances reached")
		}
	}

	// If on-demand start is enabled, start the instance
	if _, err := h.InstanceManager.StartInstance(inst.Name); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	// Wait for the instance to become healthy before proceeding
	if err := inst.WaitForHealthy(h.cfg.Instances.OnDemandStartTimeout); err != nil {
		return fmt.Errorf("instance failed to become healthy: %w", err)
	}

	return nil
}
