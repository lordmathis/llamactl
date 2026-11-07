package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"llamactl/pkg/models"
	"llamactl/pkg/validation"
	"log"
	"net/http"
	"sync"
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

// maxBodyBytes returns the request body limit in bytes, or -1 when unlimited.
func (h *Handler) maxBodyBytes() int64 {
	mb := h.cfg.Server.MaxRequestBodyMB
	switch {
	case mb < 0:
		return -1
	case mb == 0:
		mb = 100
	}
	return int64(mb) << 20
}

// limitRequestBody rejects oversized bodies: Content-Length first, then MaxBytesReader cuts off chunked senders.
func (h *Handler) limitRequestBody(w http.ResponseWriter, r *http.Request) bool {
	limit := h.maxBodyBytes()
	if limit < 0 {
		return true
	}
	if r.ContentLength > limit {
		writeRequestTooLarge(w, limit)
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	return true
}

func writeRequestTooLarge(w http.ResponseWriter, limit int64) {
	writeError(w, http.StatusRequestEntityTooLarge, "request_too_large",
		fmt.Sprintf("request body exceeds the configured limit of %d MB", limit>>20))
}

func (h *Handler) readBodyLimited(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	if !h.limitRequestBody(w, r) {
		return nil, false
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if maxErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeRequestTooLarge(w, maxErr.Limit)
			return nil, false
		}
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body")
		return nil, false
	}
	return body, true
}

// decodeJSONBody decodes dst; code and details form the 400 response, oversized bodies get a 413.
func (h *Handler) decodeJSONBody(w http.ResponseWriter, r *http.Request, code, details string, dst any) bool {
	if !h.limitRequestBody(w, r) {
		return false
	}
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if maxErr, ok := errors.AsType[*http.MaxBytesError](err); ok {
			writeRequestTooLarge(w, maxErr.Limit)
			return false
		}
		writeError(w, http.StatusBadRequest, code, details)
		return false
	}
	return true
}

// Handler provides HTTP handlers for the llamactl server API
type Handler struct {
	InstanceManager manager.InstanceManager
	modelManager    *models.Manager
	cfg             config.AppConfig
	httpClient      *http.Client
	authStore       database.AuthStore
	authMiddleware  *APIAuthMiddleware
	oidc            *OIDCService

	// startMu serializes the evict→start critical section in
	// ensureInstanceRunning. Group quotas are check-then-act; without this,
	// two concurrent requests can each see room (or evict the other's LRU)
	// and start two models in the same group. The health wait deliberately
	// runs outside this lock so unrelated starts don't serialize behind a
	// model load.
	startMu sync.Mutex
}

// NewHandler creates a new Handler instance with the provided instance manager and configuration.
// oidcService may be nil to run without OIDC login.
func NewHandler(im manager.InstanceManager, mm *models.Manager, cfg config.AppConfig, authStore database.AuthStore, oidcService *OIDCService) (*Handler, error) {
	handler := &Handler{
		InstanceManager: im,
		modelManager:    mm,
		cfg:             cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authStore: authStore,
		oidc:      oidcService,
	}
	middleware, err := NewAPIAuthMiddleware(cfg.Auth, authStore)
	if err != nil {
		return nil, fmt.Errorf("initializing auth middleware: %w", err)
	}
	handler.authMiddleware = middleware
	if oidcService != nil {
		handler.authMiddleware.sessions = oidcService.Sessions
	}
	return handler, nil
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

	// See Handler.startMu for why the start section is serialized.
	h.startMu.Lock()
	startErr := h.maybeStartLocked(inst, canEvict)
	h.startMu.Unlock()
	if startErr != nil {
		return startErr
	}

	// The health wait is deliberately outside the start lock: it can run up
	// to OnDemandStartTimeout (a full model load), and serializing unrelated
	// instance starts behind it is the slow path startMu exists to avoid.
	if err := inst.WaitForHealthy(h.cfg.Instances.OnDemandStartTimeout); err != nil {
		return fmt.Errorf("instance failed to become healthy: %w", err)
	}

	return nil
}

// maybeStartLocked starts inst if it is not already running, enforcing group
// quotas and global capacity. Callers must hold h.startMu. It re-checks
// IsRunning() under the lock because the caller's check (done before we took
// the lock) may be stale: a concurrent request may have started this same
// instance in the meantime, and calling StartInstance on an already-running
// instance would surface as a spurious 500 for requests that should just be
// served.
func (h *Handler) maybeStartLocked(inst *instance.Instance, canEvict bool) error {
	options := inst.GetOptions()
	if options == nil || options.OnDemandStart == nil || !*options.OnDemandStart {
		return fmt.Errorf("instance is not running and on-demand start is not enabled")
	}

	// A concurrent request already started it while we waited for the lock —
	// nothing to do; the caller waits for health below.
	if inst.IsRunning() {
		return nil
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

// writeInstanceLookupError writes 404 for not-found, 400 for anything else
func writeInstanceLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, manager.ErrInstanceNotFound) {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
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
