package llamactl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	InstanceManager InstanceManager
}

func NewHandler(im InstanceManager) *Handler {
	return &Handler{
		InstanceManager: im,
	}
}

// HelpHandler godoc
// @Summary Get help for llama server
// @Description Returns the help text for the llama server command
// @Tags server
// #Produces text/plain
// @Success 200 {string} string "Help text"
// @Failure 500 {string} string "Internal Server Error"
// @Router /server/help [get]
func (h *Handler) HelpHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		helpCmd := exec.Command("llama-server", "--help")
		output, err := helpCmd.CombinedOutput()
		if err != nil {
			http.Error(w, "Failed to get help: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(output)
	}
}

// VersionHandler godoc
// @Summary Get version of llama server
// @Description Returns the version of the llama server command
// @Tags server
// #Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /server/version [get]
func (h *Handler) VersionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		versionCmd := exec.Command("llama-server", "--version")
		output, err := versionCmd.CombinedOutput()
		if err != nil {
			http.Error(w, "Failed to get version: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(output)
	}
}

// ListDevicesHandler godoc
// @Summary List available devices for llama server
// @Description Returns a list of available devices for the llama server
// @Tags server
// #Produces text/plain
// @Success 200 {string} string "List of devices"
// @Failure 500 {string} string "Internal Server Error"
// @Router /server/devices [get]
func (h *Handler) ListDevicesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listCmd := exec.Command("llama-server", "--list-devices")
		output, err := listCmd.CombinedOutput()
		if err != nil {
			http.Error(w, "Failed to list devices: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write(output)
	}
}

// ListInstances godoc
// @Summary List all instances
// @Description Returns a list of all instances managed by the server
// @Tags instances
// @Produce json
// @Success 200 {array} Instance "List of instances"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances [get]
func (h *Handler) ListInstances() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instances, err := h.InstanceManager.ListInstances()
		if err != nil {
			http.Error(w, "Failed to list instances: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instances); err != nil {
			http.Error(w, "Failed to encode instances: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// CreateInstance godoc
// @Summary Create and start a new instance
// @Description Creates a new instance with the provided configuration options
// @Tags instances
// @Accept json
// @Produce json
// @Param options body InstanceOptions true "Instance configuration options"
// @Success 201 {object} Instance "Created instance details"
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances [post]
func (h *Handler) CreateInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		var options CreateInstanceOptions
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.CreateInstance(name, &options)
		if err != nil {
			http.Error(w, "Failed to create instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GetInstance godoc
// @Summary Get details of a specific instance
// @Description Returns the details of a specific instance by name
// @Tags instances
// @Param name path string true "Instance Name"
// @Success 200 {object} Instance "Instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name} [get]
func (h *Handler) GetInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.GetInstance(name)
		if err != nil {
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// UpdateInstance godoc
// @Summary Update an instance's configuration
// @Description Updates the configuration of a specific instance by name
// @Tags instances
// @Accept json
// @Produce json
// @Param name path string true "Instance Name"
// @Param options body InstanceOptions true "Instance configuration options"
// @Success 200 {object} Instance "Updated instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name} [put]
func (h *Handler) UpdateInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		var options CreateInstanceOptions
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.UpdateInstance(name, &options)
		if err != nil {
			http.Error(w, "Failed to update instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		instance, err = h.InstanceManager.RestartInstance(name)
		if err != nil {
			http.Error(w, "Failed to restart instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// StartInstance godoc
// @Summary Start a stopped instance
// @Description Starts a specific instance by name
// @Tags instances
// @Produce json
// @Param name path string true "Instance Name"
// @Success 200 {object} Instance "Started instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name}/start [post]
func (h *Handler) StartInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.StartInstance(name)
		if err != nil {
			http.Error(w, "Failed to start instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// StopInstance godoc
// @Summary Stop a running instance
// @Description Stops a specific instance by name
// @Tags instances
// @Produce json
// @Param name path string true "Instance Name"
// @Success 200 {object} Instance "Stopped instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name}/stop [post]
func (h *Handler) StopInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.StopInstance(name)
		if err != nil {
			http.Error(w, "Failed to stop instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// RestartInstance godoc
// @Summary Restart a running instance
// @Description Restarts a specific instance by name
// @Tags instances
// @Produce json
// @Param name path string true "Instance Name"
// @Success 200 {object} Instance "Restarted instance details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name}/restart [post]
func (h *Handler) RestartInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.RestartInstance(name)
		if err != nil {
			http.Error(w, "Failed to restart instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// DeleteInstance godoc
// @Summary Delete an instance
// @Description Stops and removes a specific instance by name
// @Tags instances
// @Produce json
// @Param name path string true "Instance Name"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name} [delete]
func (h *Handler) DeleteInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		if err := h.InstanceManager.DeleteInstance(name); err != nil {
			http.Error(w, "Failed to delete instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *Handler) GetInstanceLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		lines := r.URL.Query().Get("lines")
		if lines == "" {
			lines = "-1"
		}

		num_lines, err := strconv.Atoi(lines)
		if err != nil {
			http.Error(w, "Invalid lines parameter: "+err.Error(), http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.GetInstance(name)
		if err != nil {
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		logs, err := instance.GetLogs(num_lines)
		if err != nil {
			http.Error(w, "Failed to get logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(logs))
	}
}

func (h *Handler) ProxyToInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.GetInstance(name)
		if err != nil {
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !instance.Running {
			http.Error(w, "Instance is not running", http.StatusServiceUnavailable)
			return
		}

		// Get the cached proxy for this instance
		proxy, err := instance.GetProxy()
		if err != nil {
			http.Error(w, "Failed to get proxy: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Strip the "/api/v1/instances/<name>/proxy" prefix from the request URL
		prefix := fmt.Sprintf("/api/v1/instances/%s/proxy", name)
		proxyPath := r.URL.Path[len(prefix):]

		// Ensure the proxy path starts with "/"
		if !strings.HasPrefix(proxyPath, "/") {
			proxyPath = "/" + proxyPath
		}

		// Modify the request to remove the proxy prefix
		originalPath := r.URL.Path
		r.URL.Path = proxyPath

		// Set forwarded headers
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Header.Set("X-Forwarded-Proto", "http")

		// Restore original path for logging purposes
		defer func() {
			r.URL.Path = originalPath
		}()

		// Forward the request using the cached proxy
		proxy.ServeHTTP(w, r)
	}
}
