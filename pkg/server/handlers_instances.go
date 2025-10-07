package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ListInstances godoc
// @Summary List all instances
// @Description Returns a list of all instances managed by the server
// @Tags instances
// @Security ApiKeyAuth
// @Produces json
// @Success 200 {array} instance.Process "List of instances"
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
// @Security ApiKeyAuth
// @Accept json
// @Produces json
// @Param name path string true "Instance Name"
// @Param options body instance.CreateInstanceOptions true "Instance configuration options"
// @Success 201 {object} instance.Process "Created instance details"
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name} [post]
func (h *Handler) CreateInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		var options instance.CreateInstanceOptions
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		inst, err := h.InstanceManager.CreateInstance(name, &options)
		if err != nil {
			http.Error(w, "Failed to create instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(inst); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GetInstance godoc
// @Summary Get details of a specific instance
// @Description Returns the details of a specific instance by name
// @Tags instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Process "Instance details"
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

		inst, err := h.InstanceManager.GetInstance(name)
		if err != nil {
			http.Error(w, "Invalid instance: "+err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(inst); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// UpdateInstance godoc
// @Summary Update an instance's configuration
// @Description Updates the configuration of a specific instance by name
// @Tags instances
// @Security ApiKeyAuth
// @Accept json
// @Produces json
// @Param name path string true "Instance Name"
// @Param options body instance.CreateInstanceOptions true "Instance configuration options"
// @Success 200 {object} instance.Process "Updated instance details"
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

		var options instance.CreateInstanceOptions
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		inst, err := h.InstanceManager.UpdateInstance(name, &options)
		if err != nil {
			http.Error(w, "Failed to update instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(inst); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// StartInstance godoc
// @Summary Start a stopped instance
// @Description Starts a specific instance by name
// @Tags instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Process "Started instance details"
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

		inst, err := h.InstanceManager.StartInstance(name)
		if err != nil {
			// Check if error is due to maximum running instances limit
			if _, ok := err.(manager.MaxRunningInstancesError); ok {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}

			http.Error(w, "Failed to start instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(inst); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// StopInstance godoc
// @Summary Stop a running instance
// @Description Stops a specific instance by name
// @Tags instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Process "Stopped instance details"
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

		inst, err := h.InstanceManager.StopInstance(name)
		if err != nil {
			http.Error(w, "Failed to stop instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(inst); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// RestartInstance godoc
// @Summary Restart a running instance
// @Description Restarts a specific instance by name
// @Tags instances
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} instance.Process "Restarted instance details"
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

		inst, err := h.InstanceManager.RestartInstance(name)
		if err != nil {
			http.Error(w, "Failed to restart instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(inst); err != nil {
			http.Error(w, "Failed to encode instance: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// DeleteInstance godoc
// @Summary Delete an instance
// @Description Stops and removes a specific instance by name
// @Tags instances
// @Security ApiKeyAuth
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

// GetInstanceLogs godoc
// @Summary Get logs from a specific instance
// @Description Returns the logs from a specific instance by name with optional line limit
// @Tags instances
// @Security ApiKeyAuth
// @Param name path string true "Instance Name"
// @Param lines query string false "Number of lines to retrieve (default: all lines)"
// @Produces text/plain
// @Success 200 {string} string "Instance logs"
// @Failure 400 {string} string "Invalid name format or lines parameter"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{name}/logs [get]
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

		inst, err := h.InstanceManager.GetInstance(name)
		if err != nil {
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		logs, err := inst.GetLogs(num_lines)
		if err != nil {
			http.Error(w, "Failed to get logs: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(logs))
	}
}

// ProxyToInstance godoc
// @Summary Proxy requests to a specific instance
// @Description Forwards HTTP requests to the llama-server instance running on a specific port
// @Tags instances
// @Security ApiKeyAuth
// @Param name path string true "Instance Name"
// @Success 200 "Request successfully proxied to instance"
// @Failure 400 {string} string "Invalid name format"
// @Failure 500 {string} string "Internal Server Error"
// @Failure 503 {string} string "Instance is not running"
// @Router /instances/{name}/proxy [get]
// @Router /instances/{name}/proxy [post]
func (h *Handler) ProxyToInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Instance name cannot be empty", http.StatusBadRequest)
			return
		}

		inst, err := h.InstanceManager.GetInstance(name)
		if err != nil {
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if this is a remote instance
		if inst.IsRemote() {
			h.RemoteInstanceProxy(w, r, name, inst)
			return
		}

		if !inst.IsRunning() {
			http.Error(w, "Instance is not running", http.StatusServiceUnavailable)
			return
		}

		// Get the cached proxy for this instance
		proxy, err := inst.GetProxy()
		if err != nil {
			http.Error(w, "Failed to get proxy: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Strip the "/api/v1/instances/<name>/proxy" prefix from the request URL
		prefix := fmt.Sprintf("/api/v1/instances/%s/proxy", name)
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)

		// Update the last request time for the instance
		inst.UpdateLastRequestTime()

		// Set forwarded headers
		r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
		r.Header.Set("X-Forwarded-Proto", "http")

		// Forward the request using the cached proxy
		proxy.ServeHTTP(w, r)
	}
}

// RemoteInstanceProxy proxies requests to a remote instance
func (h *Handler) RemoteInstanceProxy(w http.ResponseWriter, r *http.Request, name string, inst *instance.Process) {
	// Get the node name from instance options
	options := inst.GetOptions()
	if options == nil || len(options.Nodes) == 0 {
		http.Error(w, "Instance has no node configured", http.StatusInternalServerError)
		return
	}

	nodeName := options.Nodes[0]

	// Check if we have a cached proxy for this node
	h.remoteProxiesMu.RLock()
	proxy, exists := h.remoteProxies[nodeName]
	h.remoteProxiesMu.RUnlock()

	if !exists {
		// Find node configuration
		var nodeConfig *config.NodeConfig
		for i := range h.cfg.Nodes {
			if h.cfg.Nodes[i].Name == nodeName {
				nodeConfig = &h.cfg.Nodes[i]
				break
			}
		}

		if nodeConfig == nil {
			http.Error(w, fmt.Sprintf("Node %s not found", nodeName), http.StatusInternalServerError)
			return
		}

		// Create reverse proxy to remote node
		targetURL, err := url.Parse(nodeConfig.Address)
		if err != nil {
			http.Error(w, "Failed to parse node address: "+err.Error(), http.StatusInternalServerError)
			return
		}

		proxy = httputil.NewSingleHostReverseProxy(targetURL)

		// Modify request before forwarding
		originalDirector := proxy.Director
		apiKey := nodeConfig.APIKey // Capture for closure
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// Add API key if configured
			if apiKey != "" {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
			}
		}

		// Cache the proxy by node name
		h.remoteProxiesMu.Lock()
		h.remoteProxies[nodeName] = proxy
		h.remoteProxiesMu.Unlock()
	}

	// Forward the request using the cached proxy
	proxy.ServeHTTP(w, r)
}
