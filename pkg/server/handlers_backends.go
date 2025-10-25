package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"llamactl/pkg/validation"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ParseCommandRequest represents the request body for command parsing
type ParseCommandRequest struct {
	Command string `json:"command"`
}

func (h *Handler) LlamaCppProxy(onDemandStart bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get the instance name from the URL parameter
		name := chi.URLParam(r, "name")

		// Validate instance name at the entry point
		validatedName, err := validation.ValidateInstanceName(name)
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

		options := inst.GetOptions()
		if options == nil {
			http.Error(w, "Cannot obtain Instance's options", http.StatusInternalServerError)
			return
		}

		if options.BackendOptions.BackendType != backends.BackendTypeLlamaCpp {
			http.Error(w, "Instance is not a llama.cpp server.", http.StatusBadRequest)
			return
		}

		if !inst.IsRemote() && !inst.IsRunning() {

			if !(onDemandStart && options.OnDemandStart != nil && *options.OnDemandStart) {
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

		if !inst.IsRemote() {
			// Strip the "/llama-cpp/<name>" prefix from the request URL
			prefix := fmt.Sprintf("/llama-cpp/%s", validatedName)
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}

		proxy.ServeHTTP(w, r)
	}
}

// ParseLlamaCommand godoc
// @Summary Parse llama-server command
// @Description Parses a llama-server command string into instance options
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.Options "Parsed options"
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
		llamaOptions, err := backends.ParseLlamaCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parse_error", err.Error())
			return
		}
		options := &instance.Options{
			BackendOptions: backends.Options{
				BackendType:        backends.BackendTypeLlamaCpp,
				LlamaServerOptions: llamaOptions,
			},
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
// @Success 200 {object} instance.Options "Parsed options"
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

		mlxOptions, err := backends.ParseMlxCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parse_error", err.Error())
			return
		}

		// Currently only support mlx_lm backend type
		backendType := backends.BackendTypeMlxLm

		options := &instance.Options{
			BackendOptions: backends.Options{
				BackendType:      backendType,
				MlxServerOptions: mlxOptions,
			},
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
// @Success 200 {object} instance.Options "Parsed options"
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

		vllmOptions, err := backends.ParseVllmCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusBadRequest, "parse_error", err.Error())
			return
		}

		backendType := backends.BackendTypeVllm

		options := &instance.Options{
			BackendOptions: backends.Options{
				BackendType:       backendType,
				VllmServerOptions: vllmOptions,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(options); err != nil {
			writeError(w, http.StatusInternalServerError, "encode_error", err.Error())
		}
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
