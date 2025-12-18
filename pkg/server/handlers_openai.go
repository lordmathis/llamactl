package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"net/http"
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

			if inst.IsLlamaCpp() && inst.IsRunning() {
				// Try to fetch models from the instance
				models, err := inst.GetModels()
				if err != nil {
					fmt.Printf("Failed to fetch models from instance %s: %v", inst.Name, err)
					continue
				}

				for _, model := range models {
					openaiInstances = append(openaiInstances, OpenAIInstance{
						ID:      model.ID,
						Object:  "model",
						Created: model.Created,
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

		modelName, ok := requestBody["model"].(string)
		if !ok || modelName == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "Model name is required")
			return
		}

		// Resolve model name to instance name (checks instance names first, then model registry)
		instanceName, err := h.InstanceManager.ResolveInstance(modelName)
		if err != nil {
			writeError(w, http.StatusBadRequest, "model_not_found", err.Error())
			return
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

		if !inst.IsRemote() && !inst.IsRunning() {
			err := h.ensureInstanceRunning(inst)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "instance_start_failed", err.Error())
				return
			}
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
