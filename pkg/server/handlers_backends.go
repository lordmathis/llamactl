package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"net/http"
	"os/exec"
	"strings"
)

// ParseCommandRequest represents the request body for command parsing
type ParseCommandRequest struct {
	Command string `json:"command"`
}

func (h *Handler) LlamaCppProxy(onDemandStart bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		inst, err := h.getInstance(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_instance", err.Error())
			return
		}

		options := inst.GetOptions()
		if options == nil {
			writeError(w, http.StatusInternalServerError, "options_failed", "Cannot obtain Instance's options")
			return
		}

		if options.BackendOptions.BackendType != backends.BackendTypeLlamaCpp {
			writeError(w, http.StatusBadRequest, "invalid_backend", "Instance is not a llama.cpp server.")
			return
		}

		if !inst.IsRemote() && !inst.IsRunning() && onDemandStart {
			err := h.ensureInstanceRunning(inst)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "instance_start_failed", err.Error())
				return
			}
		}

		proxy, err := inst.GetProxy()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "proxy_failed", err.Error())
			return
		}

		if !inst.IsRemote() {
			// Strip the "/llama-cpp/<name>" prefix from the request URL
			prefix := fmt.Sprintf("/llama-cpp/%s", inst.Name)
			r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		}

		proxy.ServeHTTP(w, r)
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
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.Options "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /backends/mlx/parse-command [post]
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
// @Tags backends
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ParseCommandRequest true "Command to parse"
// @Success 200 {object} instance.Options "Parsed options"
// @Failure 400 {object} map[string]string "Invalid request or command"
// @Router /backends/vllm/parse-command [post]
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
			writeError(w, http.StatusInternalServerError, "command_failed", "Failed to get help: "+err.Error())
			return
		}
		writeText(w, http.StatusOK, string(output))
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
			writeError(w, http.StatusInternalServerError, "command_failed", "Failed to get version: "+err.Error())
			return
		}
		writeText(w, http.StatusOK, string(output))
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
			writeError(w, http.StatusInternalServerError, "command_failed", "Failed to list devices: "+err.Error())
			return
		}
		writeText(w, http.StatusOK, string(output))
	}
}
