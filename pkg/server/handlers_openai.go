package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"net/http"
	"strings"
)

// OpenAIListInstancesResponse represents the response structure for listing instances (models) in OpenAI-compatible format
type OpenAIListInstancesResponse struct {
	Object string           `json:"object"`
	Data   []OpenAIInstance `json:"data"`
}

// OpenAIInstance represents a single instance (model) in OpenAI-compatible format
type OpenAIInstance struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// LlamaCppModel represents a model available in a llama.cpp instance
type LlamaCppModel struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	OwnedBy string              `json:"owned_by"`
	Created int64               `json:"created"`
	InCache bool                `json:"in_cache"`
	Path    string              `json:"path"`
	Status  LlamaCppModelStatus `json:"status"`
}

// LlamaCppModelStatus represents the status of a model in a llama.cpp instance
type LlamaCppModelStatus struct {
	Value string   `json:"value"` // "loaded" | "loading" | "unloaded"
	Args  []string `json:"args"`
}

// fetchLlamaCppModels fetches models from a llama.cpp instance using the proxy
func fetchLlamaCppModels(inst *instance.Instance) ([]LlamaCppModel, error) {
	// Create a request to the instance's /models endpoint
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/models", inst.GetHost(), inst.GetPort()), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Use a custom response writer to capture the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data []LlamaCppModel `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// OpenAIListInstances godoc
// @Summary List instances in OpenAI-compatible format
// @Description Returns a list of instances in a format compatible with OpenAI API
// @Tags OpenAI
// @Security ApiKeyAuth
// @Produces json
// @Success 200 {object} OpenAIListInstancesResponse "List of OpenAI-compatible instances"
// @Failure 500 {string} string "Internal Server Error"
// @Router /v1/models [get]
func (h *Handler) OpenAIListInstances() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instances, err := h.InstanceManager.ListInstances()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list_failed", "Failed to list instances: "+err.Error())
			return
		}

		var openaiInstances []OpenAIInstance

		// For each llama.cpp instance, try to fetch models and add them as separate entries
		for _, inst := range instances {

			if inst.GetBackendType() == backends.BackendTypeLlamaCpp && inst.IsRunning() {
				// Try to fetch models from the instance
				models, err := fetchLlamaCppModels(inst)
				if err != nil {
					fmt.Printf("Failed to fetch models from instance %s: %v", inst.Name, err)
					continue
				}

				for _, model := range models {
					openaiInstances = append(openaiInstances, OpenAIInstance{
						ID:      inst.Name + "/" + model.ID,
						Object:  "model",
						Created: inst.Created,
						OwnedBy: inst.Name,
					})
				}

				if len(models) > 1 {
					// Skip adding the instance name if multiple models are present
					continue
				}
			}

			// Add instance name as single entry (for non-llama.cpp or if model fetch failed)
			openaiInstances = append(openaiInstances, OpenAIInstance{
				ID:      inst.Name,
				Object:  "model",
				Created: inst.Created,
				OwnedBy: "llamactl",
			})
		}

		openaiResponse := OpenAIListInstancesResponse{
			Object: "list",
			Data:   openaiInstances,
		}

		writeJSON(w, http.StatusOK, openaiResponse)
	}
}

// OpenAIProxy godoc
// @Summary OpenAI-compatible proxy endpoint
// @Description Handles all POST requests to /v1/*, routing to the appropriate instance based on the request body. Requires API key authentication via the `Authorization` header.
// @Tags OpenAI
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
			writeError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body")
			return
		}
		r.Body.Close()

		// Parse the body to extract instance name
		var requestBody map[string]any
		if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
			return
		}

		reqModelName, ok := requestBody["model"].(string)
		if !ok || reqModelName == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "Model name is required")
			return
		}

		// Parse instance name and model name from <instance_name>/<model_name> format
		var instanceName string
		var modelName string

		// Check if model name contains "/"
		if idx := strings.Index(reqModelName, "/"); idx != -1 {
			// Split into instance and model parts
			instanceName = reqModelName[:idx]
			modelName = reqModelName[idx+1:]
		} else {
			instanceName = reqModelName
			modelName = reqModelName
		}

		// Validate instance name at the entry point
		validatedName, err := validation.ValidateInstanceName(instanceName)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance_name", err.Error())
			return
		}

		// Route to the appropriate inst based on instance name
		inst, err := h.InstanceManager.GetInstance(validatedName)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		// Check instance permissions
		if err := h.authMiddleware.CheckInstancePermission(r.Context(), inst.ID); err != nil {
			writeError(w, http.StatusForbidden, "permission_denied", err.Error())
			return
		}

		// Check if instance is shutting down before autostart logic
		if inst.GetStatus() == instance.ShuttingDown {
			writeError(w, http.StatusServiceUnavailable, "instance_shutting_down", "Instance is shutting down")
			return
		}

		if inst.IsRemote() {
			// Don't replace model name for remote instances
			modelName = reqModelName
		}

		if !inst.IsRemote() && !inst.IsRunning() {
			err := h.ensureInstanceRunning(inst)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "instance_start_failed", err.Error())
				return
			}
		}

		// Update the request body with just the model name
		requestBody["model"] = modelName

		// Re-marshal the updated body
		bodyBytes, err = json.Marshal(requestBody)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "marshal_error", "Failed to update request body")
			return
		}

		// Recreate the request body from the bytes we read
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r.ContentLength = int64(len(bodyBytes))

		// Use instance's ServeHTTP which tracks inflight requests and handles shutting down state
		err = inst.ServeHTTP(w, r)
		if err != nil {
			// Error is already handled in ServeHTTP (response written)
			return
		}
	}
}
