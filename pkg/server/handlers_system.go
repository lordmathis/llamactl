package server

import (
	"fmt"
	"net/http"
)

// VersionHandler godoc
// @Summary Get llamactl version
// @Description Returns the version of the llamactl command
// @Tags version
// @Security ApiKeyAuth
// @Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /version [get]
func (h *Handler) VersionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		versionInfo := fmt.Sprintf("Version: %s\nCommit: %s\nBuild Time: %s\n", h.cfg.Version, h.cfg.CommitHash, h.cfg.BuildTime)
		writeText(w, http.StatusOK, versionInfo)
	}
}
