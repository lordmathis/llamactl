package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ParseCommandRequest represents the request body for backend command parsing
type ParseCommandRequest struct {
	Command string `json:"command"`
}

// validateLlamaCppInstance validates that the instance specified in the request is a llama.cpp instance
func (h *Handler) validateLlamaCppInstance(r *http.Request) (*instance.Instance, error) {
	inst, err := h.getInstance(r)
	if err != nil {
		return nil, fmt.Errorf("invalid instance: %w", err)
	}

	options := inst.GetOptions()
	if options == nil {
		return nil, fmt.Errorf("cannot obtain instance's options")
	}

	if options.BackendOptions.BackendType != backends.BackendTypeLlamaCpp {
		return nil, fmt.Errorf("instance is not a llama.cpp server")
	}

	return inst, nil
}

// stripLlamaCppPrefix removes the llama.cpp proxy prefix from the request URL path
func (h *Handler) stripLlamaCppPrefix(r *http.Request, instName string) {
	// Strip the "/llama-cpp/<name>" prefix from the request URL
	prefix := fmt.Sprintf("/llama-cpp/%s", instName)
	r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
}

// LlamaCppUIProxy godoc
// @Summary Proxy requests to llama.cpp UI for the instance
// @Description Proxies requests to the llama.cpp UI for the specified instance
// @Tags Llama.cpp
// @Security ApiKeyAuth
// @Produce html
// @Param name query string true "Instance Name"
// @Success 200 {string} string "Proxied HTML response"
// @Failure 400 {string} string "Invalid instance"
// @Failure 500 {string} string "Internal Server Error"
// @Router /llama-cpp/{name}/ [get]
func (h *Handler) LlamaCppUIProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		inst, err := h.validateLlamaCppInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid instance", err.Error())
			return
		}

		if !inst.IsRemote() && !inst.IsRunning() {
			writeError(w, http.StatusBadRequest, "instance is not running", "Instance is not running")
			return
		}

		if !inst.IsRemote() {
			h.stripLlamaCppPrefix(r, inst.Name)
		}

		// Use instance's ServeHTTP which tracks inflight requests and handles shutting down state
		err = inst.ServeHTTP(w, r)
		if err != nil {
			// Error is already handled in ServeHTTP (response written)
			return
		}
	}
}

// LlamaCppProxy godoc
// @Summary Proxy requests to llama.cpp server instance
// @Description Proxies requests to the specified llama.cpp server instance, starting it on-demand if configured
// @Tags Llama.cpp
// @Security ApiKeyAuth
// @Produce json
// @Param name path string true "Instance Name"
// @Success 200 {object} map[string]any "Proxied response"
// @Failure 400 {string} string "Invalid instance"
// @Failure 500 {string} string "Internal Server Error"
// @Router /llama-cpp/{name}/props [get]
// @Router /llama-cpp/{name}/slots [get]
// @Router /llama-cpp/{name}/apply-template [post]
// @Router /llama-cpp/{name}/completion [post]
// @Router /llama-cpp/{name}/detokenize [post]
// @Router /llama-cpp/{name}/embeddings [post]
// @Router /llama-cpp/{name}/infill [post]
// @Router /llama-cpp/{name}/metrics [post]
// @Router /llama-cpp/{name}/props [post]
// @Router /llama-cpp/{name}/reranking [post]
// @Router /llama-cpp/{name}/tokenize [post]
func (h *Handler) LlamaCppProxy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		inst, err := h.validateLlamaCppInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid instance", err.Error())
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
				writeError(w, http.StatusInternalServerError, "instance start failed", err.Error())
				return
			}
		}

		if !inst.IsRemote() {
			h.stripLlamaCppPrefix(r, inst.Name)
		}

		// Use instance's ServeHTTP which tracks inflight requests and handles shutting down state
		err = inst.ServeHTTP(w, r)
		if err != nil {
			// Error is already handled in ServeHTTP (response written)
			return
		}
	}
}

// parseHelper parses a backend command and returns the parsed options
func parseHelper(w http.ResponseWriter, r *http.Request, backend interface {
	ParseCommand(string) (any, error)
}) (any, bool) {
	var req ParseCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return nil, false
	}

	if strings.TrimSpace(req.Command) == "" {
		writeError(w, http.StatusBadRequest, "invalid_command", "Command cannot be empty")
		return nil, false
	}

	// Parse command using the backend's ParseCommand method
	parsedOptions, err := backend.ParseCommand(req.Command)
	if err != nil {
		writeError(w, http.StatusBadRequest, "parse_error", err.Error())
		return nil, false
	}

	return parsedOptions, true
}

// ParseLlamaCommand godoc
// @Summary Parse llama-server command
// @Description Parses a llama-server command string into instance options
// @Tags Backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.Options "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/parse-command [post]
func (h *Handler) ParseLlamaCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parsedOptions, ok := parseHelper(w, r, &backends.LlamaServerOptions{})
		if !ok {
			return
		}

		options := &instance.Options{
			BackendOptions: backends.Options{
				BackendType:        backends.BackendTypeLlamaCpp,
				LlamaServerOptions: parsedOptions.(*backends.LlamaServerOptions),
			},
		}

		writeJSON(w, http.StatusOK, options)
	}
}

// ParseMlxCommand godoc
// @Summary Parse mlx_lm.server command
// @Description Parses MLX-LM server command string into instance options
// @Tags Backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.Options "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /api/v1/backends/mlx/parse-command [post]
func (h *Handler) ParseMlxCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parsedOptions, ok := parseHelper(w, r, &backends.MlxServerOptions{})
		if !ok {
			return
		}

		options := &instance.Options{
			BackendOptions: backends.Options{
				BackendType:      backends.BackendTypeMlxLm,
				MlxServerOptions: parsedOptions.(*backends.MlxServerOptions),
			},
		}

		writeJSON(w, http.StatusOK, options)
	}
}

// ParseVllmCommand godoc
// @Summary Parse vllm serve command
// @Description Parses a vLLM serve command string into instance options
// @Tags Backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.Options "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /api/v1/backends/vllm/parse-command [post]
func (h *Handler) ParseVllmCommand() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parsedOptions, ok := parseHelper(w, r, &backends.VllmServerOptions{})
		if !ok {
			return
		}

		options := &instance.Options{
			BackendOptions: backends.Options{
				BackendType:       backends.BackendTypeVllm,
				VllmServerOptions: parsedOptions.(*backends.VllmServerOptions),
			},
		}

		writeJSON(w, http.StatusOK, options)
	}
}

// executeLlamaServerCommand executes a llama-server command with the specified flag and returns the output
func (h *Handler) executeLlamaServerCommand(flag, errorMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cmd := exec.Command("llama-server", flag)
		output, err := cmd.CombinedOutput()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "command failed", errorMsg+": "+err.Error())
			return
		}
		writeText(w, http.StatusOK, string(output))
	}
}

// LlamaServerHelpHandler godoc
// @Summary Get help for llama server
// @Description Returns the help text for the llama server command
// @Tags Backends
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Help text"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/help [get]
func (h *Handler) LlamaServerHelpHandler() http.HandlerFunc {
	return h.executeLlamaServerCommand("--help", "Failed to get help")
}

// LlamaServerVersionHandler godoc
// @Summary Get version of llama server
// @Description Returns the version of the llama server command
// @Tags Backends
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/version [get]
func (h *Handler) LlamaServerVersionHandler() http.HandlerFunc {
	return h.executeLlamaServerCommand("--version", "Failed to get version")
}

// LlamaServerListDevicesHandler godoc
// @Summary List available devices for llama server
// @Description Returns a list of available devices for the llama server
// @Tags Backends
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "List of devices"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/devices [get]
func (h *Handler) LlamaServerListDevicesHandler() http.HandlerFunc {
	return h.executeLlamaServerCommand("--list-devices", "Failed to list devices")
}

// LlamaCppListModels godoc
// @Summary List models in a llama.cpp instance
// @Description Returns a list of models available in the specified llama.cpp instance
// @Tags Llama.cpp
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Success 200 {object} map[string]any "Models list response"
// @Failure 400 {string} string "Invalid instance"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/llama-cpp/{name}/models [get]
func (h *Handler) LlamaCppListModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst, err := h.getInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		models, err := inst.GetModels()
		if err != nil {
			writeError(w, http.StatusBadRequest, "get_models_failed", err.Error())
			return
		}

		response := map[string]any{
			"object": "list",
			"data":   models,
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// LlamaCppLoadModel godoc
// @Summary Load a model in a llama.cpp instance
// @Description Loads the specified model in the given llama.cpp instance
// @Tags Llama.cpp
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Param model path string true "Model Name"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {string} string "Invalid request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/llama-cpp/{name}/models/{model}/load [post]
func (h *Handler) LlamaCppLoadModel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst, err := h.getInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		modelName := chi.URLParam(r, "model")

		if err := inst.LoadModel(modelName); err != nil {
			writeError(w, http.StatusBadRequest, "load_model_failed", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Model %s loaded successfully", modelName),
		})
	}
}

// LlamaCppUnloadModel godoc
// @Summary Unload a model in a llama.cpp instance
// @Description Unloads the specified model in the given llama.cpp instance
// @Tags Llama.cpp
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Instance Name"
// @Param model path string true "Model Name"
// @Success 200 {object} map[string]string "Success message"
// @Failure 400 {string} string "Invalid request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/llama-cpp/{name}/models/{model}/unload [post]
func (h *Handler) LlamaCppUnloadModel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		inst, err := h.getInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		modelName := chi.URLParam(r, "model")

		if err := inst.UnloadModel(modelName); err != nil {
			writeError(w, http.StatusBadRequest, "unload_model_failed", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "success",
			"message": fmt.Sprintf("Model %s unloaded successfully", modelName),
		})
	}
}
