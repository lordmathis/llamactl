package server

import (
	"fmt"
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

// ConfigHandler godoc
// @Summary Get server configuration
// @Description Returns the current server configuration (sanitized)
// @Tags System
// @Security ApiKeyAuth
// @Produces application/json
// @Success 200 {object} config.AppConfig "Sanitized configuration"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/config [get]
func (h *Handler) ConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Return a sanitized copy of the configuration
		sanitizedConfig := h.cfg.SanitizedCopy()
		writeJSON(w, http.StatusOK, sanitizedConfig)
	}
}
