package server

import (
	"fmt"
	"llamactl/pkg/config"
	"net/http"
)

// VersionHandler godoc
// @Summary Get llamactl version
// @Description Returns the version of the llamactl command
// @Tags System
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/version [get]
func (h *Handler) VersionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		versionInfo := fmt.Sprintf("Version: %s\nCommit: %s\nBuild Time: %s\n", h.cfg.Version, h.cfg.CommitHash, h.cfg.BuildTime)
		writeText(w, http.StatusOK, versionInfo)
	}
}

// ConfigResponse is the allowlisted projection of AppConfig exposed via
// GET /api/v1/config. Fields here are the only server configuration that
// leaves the API; adding a field is a deliberate, reviewed decision.
type ConfigResponse struct {
	LocalNode        string           `json:"local_node,omitempty"`
	InstanceDefaults InstanceDefaults `json:"instance_defaults"`
	Backends         Backends         `json:"backends"`
}

// InstanceDefaults mirrors the global instance defaults that
// validateAndApplyDefaults applies server-side to new instances.
type InstanceDefaults struct {
	IdleTimeout    int  `json:"idle_timeout"`
	AutoRestart    bool `json:"auto_restart"`
	MaxRestarts    int  `json:"max_restarts"`
	RestartDelay   int  `json:"restart_delay"`
	OnDemandStart  bool `json:"on_demand_start"`
}

type Backends struct {
	LlamaCpp BackendSummary            `json:"llama-cpp"`
	VLLM     BackendSummary            `json:"vllm"`
	MLX      BackendSummary            `json:"mlx"`
	Custom   map[string]BackendSummary `json:"custom,omitempty"`
}

// BackendSummary carries only display metadata for the UI; credential-bearing
// BackendSettings fields (environment, args, response_headers) are excluded.
type BackendSummary struct {
	Command string        `json:"command"`
	Docker  *DockerSummary `json:"docker,omitempty"`
}

type DockerSummary struct {
	Enabled bool   `json:"enabled"`
	Image   string `json:"image"`
}

func newBackendSummary(s config.BackendSettings) BackendSummary {
	summary := BackendSummary{Command: s.Command}
	if s.Docker != nil {
		summary.Docker = &DockerSummary{Enabled: s.Docker.Enabled, Image: s.Docker.Image}
	}
	return summary
}

func newConfigResponse(cfg *config.AppConfig) ConfigResponse {
	resp := ConfigResponse{
		LocalNode: cfg.LocalNode,
		InstanceDefaults: InstanceDefaults{
			IdleTimeout:    cfg.Instances.DefaultIdleTimeout,
			AutoRestart:    cfg.Instances.DefaultAutoRestart,
			MaxRestarts:    cfg.Instances.DefaultMaxRestarts,
			RestartDelay:   cfg.Instances.DefaultRestartDelay,
			OnDemandStart:  cfg.Instances.DefaultOnDemandStart,
		},
		Backends: Backends{
			LlamaCpp: newBackendSummary(cfg.Backends.LlamaCpp),
			VLLM:     newBackendSummary(cfg.Backends.VLLM),
			MLX:      newBackendSummary(cfg.Backends.MLX),
		},
	}
	for name, settings := range cfg.Backends.Custom {
		if resp.Backends.Custom == nil {
			resp.Backends.Custom = make(map[string]BackendSummary)
		}
		resp.Backends.Custom[name] = newBackendSummary(settings)
	}
	return resp
}

// ConfigHandler godoc
// @Summary Get server configuration
// @Description Returns an allowlisted projection of the server configuration:
// @Description global instance defaults, backend command/docker summaries, and
// @Description the local node name. Secret-bearing fields are never included.
// @Tags System
// @Security ApiKeyAuth
// @Produces application/json
// @Success 200 {object} ConfigResponse "Allowlisted configuration"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/config [get]
func (h *Handler) ConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, newConfigResponse(&h.cfg))
	}
}
