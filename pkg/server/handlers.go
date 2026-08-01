package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"llamactl/pkg/models"
	"llamactl/pkg/validation"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Sentinel errors for instance start outcomes. Used by callers (via errors.Is)
// to classify the failure into the right HTTP response instead of string matching.
var (
	// ErrInstanceNotRunning is returned when the key is not permitted to start
	// the instance (can_start=false).
	ErrInstanceNotRunning = errors.New("instance is not running")
	// ErrMaxInstancesReached is returned when capacity is full and the key cannot
	// (or may not) evict to make room.
	ErrMaxInstancesReached = errors.New("cannot start instance, maximum number of instances reached")
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
	modelManager    *models.Manager
	cfg             config.AppConfig
	httpClient      *http.Client
	authStore       database.AuthStore
	authMiddleware  *APIAuthMiddleware
}

// NewHandler creates a new Handler instance with the provided instance manager and configuration
func NewHandler(im manager.InstanceManager, mm *models.Manager, cfg config.AppConfig, authStore database.AuthStore) *Handler {
	handler := &Handler{
		InstanceManager: im,
		modelManager:    mm,
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

// ensureInstanceRunning ensures that an instance is running by starting it if on-demand start is enabled.
// canStart controls whether the key may auto-start a stopped instance; canEvict controls whether it may
// trigger LRU eviction of other instances to make room. It performs hierarchical eviction when allowed:
// group quota check first, then global capacity check.
func (h *Handler) ensureInstanceRunning(inst *instance.Instance, canStart bool, canEvict bool) error {
	if !canStart {
		return ErrInstanceNotRunning
	}

	options := inst.GetOptions()
	if options == nil || options.OnDemandStart == nil || !*options.OnDemandStart {
		return fmt.Errorf("instance is not running and on-demand start is not enabled")
	}

	if !h.cfg.Instances.EnableLRUEviction {
		return h.rejectIfAtCapacity()
	}

	if canEvict {
		if err := h.evictFromGroupQuota(options.Group); err != nil {
			return err
		}
		if err := h.evictFromGlobalCapacity(); err != nil {
			return err
		}
	} else {
		// Cannot evict — fail if at capacity.
		if h.InstanceManager.AtMaxRunning() {
			return ErrMaxInstancesReached
		}
	}

	if _, err := h.InstanceManager.StartInstance(inst.Name); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	if err := inst.WaitForHealthy(h.cfg.Instances.OnDemandStartTimeout); err != nil {
		return fmt.Errorf("instance failed to become healthy: %w", err)
	}

	return nil
}

func (h *Handler) rejectIfAtCapacity() error {
	if h.InstanceManager.AtMaxRunning() {
		return ErrMaxInstancesReached
	}
	return nil
}

// writeStartError maps an ensureInstanceRunning error to the appropriate HTTP response.
// Shared by every auto-start call site.
func writeStartError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInstanceNotRunning):
		writeError(w, http.StatusServiceUnavailable, "instance_not_running", err.Error())
	case errors.Is(err, ErrMaxInstancesReached):
		writeError(w, http.StatusServiceUnavailable, "max_instances_reached", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "instance_start_failed", err.Error())
	}
}

func (h *Handler) evictFromGroupQuota(group string) error {
	if group == "" {
		return nil
	}
	groupLimit, hasLimit := h.cfg.Instances.GroupLimits[group]
	if !hasLimit {
		return nil
	}
	if h.InstanceManager.CountRunningInGroup(group) < groupLimit {
		return nil
	}
	if err := h.InstanceManager.EvictLRUInstance(group); err != nil {
		return fmt.Errorf("cannot start instance, failed to evict from group %s: %w", group, err)
	}
	return nil
}

func (h *Handler) evictFromGlobalCapacity() error {
	if !h.InstanceManager.AtMaxRunning() {
		return nil
	}
	if err := h.InstanceManager.EvictLRUInstance(""); err != nil {
		return fmt.Errorf("cannot start instance, failed to evict instance: %w", err)
	}
	return nil
}
