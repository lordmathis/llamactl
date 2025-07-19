package llamactl

import (
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

// GetInstance godoc
// @Summary Get details of a specific instance
// @Description Returns the details of a specific instance by ID
// @Tags instances
// @Param id path string true "Instance ID"
// @Success 200 {object} Instance "Instance details"
// @Failure 400 {string} string "Invalid UUID format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id} [get]
func (h *Handler) GetInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uuid, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid UUID format", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.GetInstance(uuid)
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
// @Description Updates the configuration of a specific instance by ID
// @Tags instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID"
// @Param options body InstanceOptions true "Instance configuration options"
// @Success 200 {object} Instance "Updated instance details"
// @Failure 400 {string} string "Invalid UUID format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id} [put]
func (h *Handler) UpdateInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uuid, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid UUID format", http.StatusBadRequest)
			return
		}

		var options InstanceOptions
		if err := json.NewDecoder(r.Body).Decode(&options); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.UpdateInstance(uuid, &options)
		if err != nil {
			http.Error(w, "Failed to update instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		instance, err = h.InstanceManager.RestartInstance(uuid)
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
// @Description Starts a specific instance by ID
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} Instance "Started instance details"
// @Failure 400 {string} string "Invalid UUID format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id}/start [post]
func (h *Handler) StartInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uuid, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid UUID format", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.StartInstance(uuid)
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
// @Description Stops a specific instance by ID
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} Instance "Stopped instance details"
// @Failure 400 {string} string "Invalid UUID format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id}/stop [post]
func (h *Handler) StopInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uuid, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid UUID format", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.StopInstance(uuid)
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
// @Description Restarts a specific instance by ID
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} Instance "Restarted instance details"
// @Failure 400 {string} string "Invalid UUID format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id}/restart [post]
func (h *Handler) RestartInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uuid, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid UUID format", http.StatusBadRequest)
			return
		}

		instance, err := h.InstanceManager.RestartInstance(uuid)
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
// @Description Stops and removes a specific instance by ID
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid UUID format"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id} [delete]
func (h *Handler) DeleteInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uuid, err := uuid.Parse(id)
		if err != nil {
			http.Error(w, "Invalid UUID format", http.StatusBadRequest)
			return
		}

		if err := h.InstanceManager.DeleteInstance(uuid); err != nil {
			http.Error(w, "Failed to delete instance: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
