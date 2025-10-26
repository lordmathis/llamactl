package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NodeResponse represents a node configuration in API responses
type NodeResponse struct {
	Address string `json:"address"`
}

// ListNodes godoc
// @Summary List all configured nodes
// @Description Returns a map of all nodes configured in the server (node name -> node config)
// @Tags nodes
// @Security ApiKeyAuth
// @Produces json
// @Success 200 {object} map[string]NodeResponse "Map of nodes"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/nodes [get]
func (h *Handler) ListNodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Convert to sanitized response format (map of name -> NodeResponse)
		nodeResponses := make(map[string]NodeResponse, len(h.cfg.Nodes))
		for name, node := range h.cfg.Nodes {
			nodeResponses[name] = NodeResponse{
				Address: node.Address,
			}
		}

		writeJSON(w, http.StatusOK, nodeResponses)
	}
}

// GetNode godoc
// @Summary Get details of a specific node
// @Description Returns the details of a specific node by name
// @Tags nodes
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Node Name"
// @Success 200 {object} NodeResponse "Node details"
// @Failure 400 {string} string "Invalid name format"
// @Failure 404 {string} string "Node not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/nodes/{name} [get]
func (h *Handler) GetNode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "Node name cannot be empty")
			return
		}

		nodeConfig, exists := h.cfg.Nodes[name]
		if !exists {
			writeError(w, http.StatusNotFound, "not_found", "Node not found")
			return
		}

		// Convert to sanitized response format
		nodeResponse := NodeResponse{
			Address: nodeConfig.Address,
		}

		writeJSON(w, http.StatusOK, nodeResponse)
	}
}
