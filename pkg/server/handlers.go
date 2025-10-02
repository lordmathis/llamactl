package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/backends"
	"llamactl/pkg/backends/llamacpp"
	"llamactl/pkg/backends/mlx"
	"llamactl/pkg/backends/vllm"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"llamactl/pkg/manager"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	InstanceManager manager.InstanceManager
	cfg             config.AppConfig
	httpClient      *http.Client
}

func NewHandler(im manager.InstanceManager, cfg config.AppConfig) *Handler {
	return &Handler{
		InstanceManager: im,
		cfg:             cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VersionHandler godoc
// @Summary Get llamactl version
// @Description Returns the version of the llamactl command
// @Tags version
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /version [get]
func (h *Handler) VersionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Version: %s\nCommit: %s\nBuild Time: %s\n", h.cfg.Version, h.cfg.CommitHash, h.cfg.BuildTime)
	}
}

// LlamaServerHelpHandler godoc
// @Summary Get help for llama server
// @Description Returns the help text for the llama server command
// @Tags backends
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Help text"
// @Failure 500 {string} string "Internal Server Error"
// @Router /backends/llama-cpp/help [get]
func (h *Handler) LlamaServerHelpHandler() http.HandlerFunc {
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

// LlamaServerVersionHandler godoc
// @Summary Get version of llama server
// @Description Returns the version of the llama server command
// @Tags backends
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /backends/llama-cpp/version [get]
func (h *Handler) LlamaServerVersionHandler() http.HandlerFunc {
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

// LlamaServerListDevicesHandler godoc
// @Summary List available devices for llama server
// @Description Returns a list of available devices for the llama server
// @Tags backends
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "List of devices"
// @Failure 500 {string} string "Internal Server Error"
// @Router /backends/llama-cpp/devices [get]
func (h *Handler) LlamaServerListDevicesHandler() http.HandlerFunc {
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
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
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
		proxyPath := r.URL.Path[len(prefix):]

		// Ensure the proxy path starts with "/"
		if !strings.HasPrefix(proxyPath, "/") {
			proxyPath = "/" + proxyPath
		}

		// Update the last request time for the instance
		inst.UpdateLastRequestTime()

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

// RemoteInstanceProxy proxies requests to a remote instance
func (h *Handler) RemoteInstanceProxy(w http.ResponseWriter, r *http.Request, name string, inst *instance.Process) {
	// Get the node name from instance options
	options := inst.GetOptions()
	if options == nil || len(options.Nodes) == 0 {
		http.Error(w, "Instance has no node configured", http.StatusInternalServerError)
		return
	}

	nodeName := options.Nodes[0]
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

	// Strip the "/api/v1/instances/<name>/proxy" prefix from the request URL
	prefix := fmt.Sprintf("/api/v1/instances/%s/proxy", name)
	proxyPath := r.URL.Path[len(prefix):]

	// Build the remote URL
	remoteURL := fmt.Sprintf("%s/api/v1/instances/%s/proxy%s", nodeConfig.Address, name, proxyPath)

	// Create a new request to the remote node
	req, err := http.NewRequest(r.Method, remoteURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create remote request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	req.Header = r.Header.Clone()

	// Add API key if configured
	if nodeConfig.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", nodeConfig.APIKey))
	}

	// Forward the request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to proxy to remote instance: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

// OpenAIListInstances godoc
// @Summary List instances in OpenAI-compatible format
// @Description Returns a list of instances in a format compatible with OpenAI API
// @Tags openai
// @Security ApiKeyAuth
// @Produces json
// @Success 200 {object} OpenAIListInstancesResponse "List of OpenAI-compatible instances"
// @Failure 500 {string} string "Internal Server Error"
// @Router /v1/models [get]
func (h *Handler) OpenAIListInstances() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instances, err := h.InstanceManager.ListInstances()
		if err != nil {
			http.Error(w, "Failed to list instances: "+err.Error(), http.StatusInternalServerError)
			return
		}

		openaiInstances := make([]OpenAIInstance, len(instances))
		for i, inst := range instances {
			openaiInstances[i] = OpenAIInstance{
				ID:      inst.Name,
				Object:  "model",
				Created: inst.Created,
				OwnedBy: "llamactl",
			}
		}

		openaiResponse := OpenAIListInstancesResponse{
			Object: "list",
			Data:   openaiInstances,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(openaiResponse); err != nil {
			http.Error(w, "Failed to encode instances: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// OpenAIProxy godoc
// @Summary OpenAI-compatible proxy endpoint
// @Description Handles all POST requests to /v1/*, routing to the appropriate instance based on the request body. Requires API key authentication via the `Authorization` header.
// @Tags openai
// @Security ApiKeyAuth
// @Accept json
// @Produces json
// @Success 200 "OpenAI response"
// @Failure 400 {string} string "Invalid request body or instance name"
// @Failure 500 {string} string "Internal Server Error"
// @Router /v1/ [post]
func (h *Handler) OpenAIProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read the entire body first
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body.Close()

		// Parse the body to extract instance name
		var requestBody map[string]any
		if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		modelName, ok := requestBody["model"].(string)
		if !ok || modelName == "" {
			http.Error(w, "Instance name is required", http.StatusBadRequest)
			return
		}

		// Route to the appropriate inst based on instance name
		inst, err := h.InstanceManager.GetInstance(modelName)
		if err != nil {
			http.Error(w, "Failed to get instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if this is a remote instance
		if inst.IsRemote() {
			h.RemoteOpenAIProxy(w, r, modelName, inst, bodyBytes)
			return
		}

		if !inst.IsRunning() {
			allowOnDemand := inst.GetOptions() != nil && inst.GetOptions().OnDemandStart != nil && *inst.GetOptions().OnDemandStart
			if !allowOnDemand {
				http.Error(w, "Instance is not running", http.StatusServiceUnavailable)
				return
			}

			if h.InstanceManager.IsMaxRunningInstancesReached() {
				if h.cfg.Instances.EnableLRUEviction {
					err := h.InstanceManager.EvictLRUInstance()
					if err != nil {
						http.Error(w, "Cannot start Instance, failed to evict instance "+err.Error(), http.StatusInternalServerError)
						return
					}
				} else {
					http.Error(w, "Cannot start Instance, maximum number of instances reached", http.StatusConflict)
					return
				}
			}

			// If on-demand start is enabled, start the instance
			if _, err := h.InstanceManager.StartInstance(modelName); err != nil {
				http.Error(w, "Failed to start instance: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Wait for the instance to become healthy before proceeding
			if err := inst.WaitForHealthy(h.cfg.Instances.OnDemandStartTimeout); err != nil { // 2 minutes timeout
				http.Error(w, "Instance failed to become healthy: "+err.Error(), http.StatusServiceUnavailable)
				return
			}
		}

		proxy, err := inst.GetProxy()
		if err != nil {
			http.Error(w, "Failed to get proxy: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update last request time for the instance
		inst.UpdateLastRequestTime()

		// Recreate the request body from the bytes we read
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r.ContentLength = int64(len(bodyBytes))

		proxy.ServeHTTP(w, r)
	}
}

// RemoteOpenAIProxy proxies OpenAI-compatible requests to a remote instance
func (h *Handler) RemoteOpenAIProxy(w http.ResponseWriter, r *http.Request, modelName string, inst *instance.Process, bodyBytes []byte) {
	// Get the node name from instance options
	options := inst.GetOptions()
	if options == nil || len(options.Nodes) == 0 {
		http.Error(w, "Instance has no node configured", http.StatusInternalServerError)
		return
	}

	nodeName := options.Nodes[0]
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

	// Build the remote URL - forward to the same OpenAI endpoint on the remote node
	remoteURL := fmt.Sprintf("%s%s", nodeConfig.Address, r.URL.Path)
	if r.URL.RawQuery != "" {
		remoteURL += "?" + r.URL.RawQuery
	}

	// Create a new request to the remote node
	req, err := http.NewRequest(r.Method, remoteURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, "Failed to create remote request: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	req.Header = r.Header.Clone()

	// Add API key if configured
	if nodeConfig.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", nodeConfig.APIKey))
	}

	// Forward the request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to proxy to remote instance: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

// NodeResponse represents a sanitized node configuration for API responses
type NodeResponse struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// ParseCommandRequest represents the request body for command parsing
type ParseCommandRequest struct {
	Command string `json:"command"`
}

// ParseLlamaCommand godoc
// @Summary Parse llama-server command
// @Description Parses a llama-server command string into instance options
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.CreateInstanceOptions "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /backends/llama-cpp/parse-command [post]
func (h *Handler) ParseLlamaCommand() http.HandlerFunc {
	type errorResponse struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
	}
	writeError := func(w http.ResponseWriter, status int, code, details string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: code, Details: details})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req ParseCommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}
		if strings.TrimSpace(req.Command) == "" {
			writeError(w, http.StatusBadRequest, "invalid_command", "Command cannot be empty")
			return
		}
		llamaOptions, err := llamacpp.ParseLlamaCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parse_error", err.Error())
			return
		}
		options := &instance.CreateInstanceOptions{
			BackendType:        backends.BackendTypeLlamaCpp,
			LlamaServerOptions: llamaOptions,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(options); err != nil {
			writeError(w, http.StatusInternalServerError, "encode_error", err.Error())
		}
	}
}

// ParseMlxCommand godoc
// @Summary Parse mlx_lm.server command
// @Description Parses MLX-LM server command string into instance options
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.CreateInstanceOptions "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /backends/mlx/parse-command [post]
func (h *Handler) ParseMlxCommand() http.HandlerFunc {
	type errorResponse struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
	}
	writeError := func(w http.ResponseWriter, status int, code, details string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: code, Details: details})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req ParseCommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		if strings.TrimSpace(req.Command) == "" {
			writeError(w, http.StatusBadRequest, "invalid_command", "Command cannot be empty")
			return
		}

		mlxOptions, err := mlx.ParseMlxCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parse_error", err.Error())
			return
		}

		// Currently only support mlx_lm backend type
		backendType := backends.BackendTypeMlxLm

		options := &instance.CreateInstanceOptions{
			BackendType:      backendType,
			MlxServerOptions: mlxOptions,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(options); err != nil {
			writeError(w, http.StatusInternalServerError, "encode_error", err.Error())
		}
	}
}

// ParseVllmCommand godoc
// @Summary Parse vllm serve command
// @Description Parses a vLLM serve command string into instance options
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.CreateInstanceOptions "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /backends/vllm/parse-command [post]
func (h *Handler) ParseVllmCommand() http.HandlerFunc {
	type errorResponse struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
	}
	writeError := func(w http.ResponseWriter, status int, code, details string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(errorResponse{Error: code, Details: details})
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req ParseCommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
			return
		}

		if strings.TrimSpace(req.Command) == "" {
			writeError(w, http.StatusBadRequest, "invalid_command", "Command cannot be empty")
			return
		}

		vllmOptions, err := vllm.ParseVllmCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parse_error", err.Error())
			return
		}

		backendType := backends.BackendTypeVllm

		options := &instance.CreateInstanceOptions{
			BackendType:       backendType,
			VllmServerOptions: vllmOptions,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(options); err != nil {
			writeError(w, http.StatusInternalServerError, "encode_error", err.Error())
		}
	}
}

// ListNodes godoc
// @Summary List all configured nodes
// @Description Returns a list of all nodes configured in the server
// @Tags nodes
// @Security ApiKeyAuth
// @Produces json
// @Success 200 {array} NodeResponse "List of nodes"
// @Failure 500 {string} string "Internal Server Error"
// @Router /nodes [get]
func (h *Handler) ListNodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Convert to sanitized response format
		nodeResponses := make([]NodeResponse, len(h.cfg.Nodes))
		for i, node := range h.cfg.Nodes {
			nodeResponses[i] = NodeResponse{
				Name:    node.Name,
				Address: node.Address,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodeResponses); err != nil {
			http.Error(w, "Failed to encode nodes: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GetNode godoc
// @Summary Get details of a specific node
// @Description Returns the details of a specific node by name
// @Tags nodes
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Node Name"
// @Success 200 {object} NodeResponse "Node details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 404 {string} string "Node not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /nodes/{name} [get]
func (h *Handler) GetNode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Node name cannot be empty", http.StatusBadRequest)
			return
		}

		var nodeConfig *config.NodeConfig
		for i := range h.cfg.Nodes {
			if h.cfg.Nodes[i].Name == name {
				nodeConfig = &h.cfg.Nodes[i]
				break
			}
		}

		if nodeConfig == nil {
			http.Error(w, "Node not found", http.StatusNotFound)
			return
		}

		// Convert to sanitized response format
		nodeResponse := NodeResponse{
			Name:    nodeConfig.Name,
			Address: nodeConfig.Address,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodeResponse); err != nil {
			http.Error(w, "Failed to encode node: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
