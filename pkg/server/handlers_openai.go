package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"net/http"
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
