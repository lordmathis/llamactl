package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"net/http"
	"net/http/httputil"
	"net/url"
)

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

		// Validate instance name at the entry point
		validatedName, err := validation.ValidateInstanceName(modelName)
		if err != nil {
			http.Error(w, "Invalid instance name: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Route to the appropriate inst based on instance name
		inst, err := h.InstanceManager.GetInstance(validatedName)
		if err != nil {
			http.Error(w, "Invalid instance: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Check if this is a remote instance
		if inst.IsRemote() {
			// Restore the body for the remote proxy
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			h.RemoteOpenAIProxy(w, r, validatedName, inst)
			return
		}

		if !inst.IsRunning() {
			options := inst.GetOptions()
			allowOnDemand := options != nil && options.OnDemandStart != nil && *options.OnDemandStart
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
			if _, err := h.InstanceManager.StartInstance(validatedName); err != nil {
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
func (h *Handler) RemoteOpenAIProxy(w http.ResponseWriter, r *http.Request, modelName string, inst *instance.Instance) {
	// Get the node name from instance options
	options := inst.GetOptions()
	if options == nil {
		http.Error(w, "Instance has no options configured", http.StatusInternalServerError)
		return
	}

	// Get the first node from the set
	var nodeName string
	for node := range options.Nodes {
		nodeName = node
		break
	}
	if nodeName == "" {
		http.Error(w, "Instance has no node configured", http.StatusInternalServerError)
		return
	}

	// Check if we have a cached proxy for this node
	h.remoteProxiesMu.RLock()
	proxy, exists := h.remoteProxies[nodeName]
	h.remoteProxiesMu.RUnlock()

	if !exists {
		// Find node configuration
		nodeConfig, exists := h.cfg.Nodes[nodeName]
		if !exists {
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

		// Cache the proxy
		h.remoteProxiesMu.Lock()
		h.remoteProxies[nodeName] = proxy
		h.remoteProxiesMu.Unlock()
	}

	// Forward the request using the cached proxy
	proxy.ServeHTTP(w, r)
}
