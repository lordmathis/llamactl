package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"llamactl/pkg/validation"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ListInstances godoc
// @Summary List all instances
// @Description Returns a list of all instances managed by the server
// @Tags Instances
// @Security ApiKeyAuth
// @Produces json
// @Success 200 {array} instance.Instance "List of instances"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances [get]
func (h *Handler) ListInstances() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instances, err := h.InstanceManager.ListInstances()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list_failed", "Failed to list instances: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, instances)
	}
}

// CreateInstance godoc
// @Summary Create and start a new instance
// @Description Creates a new instance with the provided configuration options
// @Tags Instances
// @Security ApiKeyAuth
// @Accept json
// @Produces json
// @Param name path string true "Instance Name"
// @Param options body instance.Options true "Instance configuration options"
// @Success 201 {object} instance.Instance "Created instance details"
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name} [post]
func (h *Handler) CreateInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		var options instance.Options
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}

		inst, err := h.InstanceManager.CreateInstance(validatedName, &options)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "create_failed", "Failed to create instance: "+err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, inst)
	}
}

// GetInstance godoc
// @Summary Get details of a specific instance
// @Description Returns the details of a specific instance by name
// @Tags Instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Instance "Instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name} [get]
func (h *Handler) GetInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		inst, err := h.InstanceManager.GetInstance(validatedName)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, inst)
	}
}

// UpdateInstance godoc
// @Summary Update an instance's configuration
// @Description Updates the configuration of a specific instance by name
// @Tags Instances
// @Security ApiKeyAuth
// @Accept json
// @Produces json
// @Param name path string true "Instance Name"
// @Param options body instance.Options true "Instance configuration options"
// @Success 200 {object} instance.Instance "Updated instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name} [put]
func (h *Handler) UpdateInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		var options instance.Options
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}

		inst, err := h.InstanceManager.UpdateInstance(validatedName, &options)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "update_failed", "Failed to update instance: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, inst)
	}
}

// StartInstance godoc
// @Summary Start a stopped instance
// @Description Starts a specific instance by name
// @Tags Instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Instance "Started instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name}/start [post]
func (h *Handler) StartInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		inst, err := h.InstanceManager.StartInstance(validatedName)
		if err != nil {
			// Check if error is due to maximum running instances limit
			if _, ok := err.(manager.MaxRunningInstancesError); ok {
				writeError(w, http.StatusConflict, "max_instances_reached", err.Error())
				return
			}

			writeError(w, http.StatusInternalServerError, "start_failed", "Failed to start instance: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, inst)
	}
}

// StopInstance godoc
// @Summary Stop a running instance
// @Description Stops a specific instance by name
// @Tags Instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Instance "Stopped instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name}/stop [post]
func (h *Handler) StopInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		inst, err := h.InstanceManager.StopInstance(validatedName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "stop_failed", "Failed to stop instance: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, inst)
	}
}

// RestartInstance godoc
// @Summary Restart a running instance
// @Description Restarts a specific instance by name
// @Tags Instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Instance "Restarted instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name}/restart [post]
func (h *Handler) RestartInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		inst, err := h.InstanceManager.RestartInstance(validatedName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "restart_failed", "Failed to restart instance: "+err.Error())
			return
		}

		writeJSON(w, http.StatusOK, inst)
	}
}

// DeleteInstance godoc
// @Summary Delete an instance
// @Description Stops and removes a specific instance by name
// @Tags Instances
// @Security ApiKeyAuth
// @Param name path string true "Instance Name"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name} [delete]
func (h *Handler) DeleteInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		if err := h.InstanceManager.DeleteInstance(validatedName); err != nil {
			writeError(w, http.StatusInternalServerError, "delete_failed", "Failed to delete instance: "+err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetInstanceLogs godoc
// @Summary Get logs from a specific instance
// @Description Returns the logs from a specific instance by name with optional line limit
// @Tags Instances
// @Security ApiKeyAuth
// @Param name path string true "Instance Name"
// @Param lines query string false "Number of lines to retrieve (default: all lines)"
// @Produces text/plain
// @Success 200 {string} string "Instance logs"
// @Failure 400 {string} string "Invalid name format or lines parameter"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/instances/{name}/logs [get]
func (h *Handler) GetInstanceLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		validatedName, err := validation.ValidateInstanceName(name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		lines := r.URL.Query().Get("lines")
		numLines := -1 // Default to all lines
		if lines != "" {
			parsedLines, err := strconv.Atoi(lines)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_parameter", "Invalid lines parameter: "+err.Error())
				return
			}
			numLines = parsedLines
		}

		// Use the instance manager which handles both local and remote instances
		logs, err := h.InstanceManager.GetInstanceLogs(validatedName, numLines)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "logs_failed", "Failed to get logs: "+err.Error())
			return
		}

		writeText(w, http.StatusOK, logs)
	}
}

// InstanceProxy godoc
// @Summary Proxy requests to a specific instance, does not autostart instance if stopped
// @Description Forwards HTTP requests to the llama-server instance running on a specific port
// @Tags Instances
// @Security ApiKeyAuth
// @Param name path string true "Instance Name"
// @Success 200 "Request successfully proxied to instance"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Failure 503 {string} string "Instance is not running"
// @Router /api/v1/instances/{name}/proxy [get]
// @Router /api/v1/instances/{name}/proxy [post]
func (h *Handler) InstanceProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst, err := h.getInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		// Check instance permissions
		if err := h.authMiddleware.CheckInstancePermission(r.Context(), inst.ID); err != nil {
			writeError(w, http.StatusForbidden, "permission_denied", err.Error())
			return
		}

		if !inst.IsRunning() {
			writeError(w, http.StatusServiceUnavailable, "instance_not_running", "Instance is not running")
			return
		}

		if !inst.IsRemote() {
			// Strip the "/api/v1/instances/<name>/proxy" prefix from the request URL
			prefix := fmt.Sprintf("/api/v1/instances/%s/proxy", inst.Name)
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}

		// Set forwarded headers
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Header.Set("X-Forwarded-Proto", "http")

		// Use instance's ServeHTTP which tracks inflight requests and handles shutting down state
		err = inst.ServeHTTP(w, r)
		if err != nil {
			// Error is already handled in ServeHTTP (response written)
			return
		}
	}
}
