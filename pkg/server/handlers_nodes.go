package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NodeResponse represents a sanitized node configuration for API responses
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
// @Router /nodes [get]
func (h *Handler) ListNodes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Convert to sanitized response format (map of name -> NodeResponse)
		nodeResponses := make(map[string]NodeResponse, len(h.cfg.Nodes))
		for name, node := range h.cfg.Nodes {
			nodeResponses[name] = NodeResponse{
				Address: node.Address,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodeResponses); err != nil {
			http.Error(w, "Failed to encode nodes: "+err.Error(), http.StatusInternalServerError)
			return
		}
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
// @Router /nodes/{name} [get]
func (h *Handler) GetNode() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" {
			http.Error(w, "Node name cannot be empty", http.StatusBadRequest)
			return
		}

		nodeConfig, exists := h.cfg.Nodes[name]
		if !exists {
			http.Error(w, "Node not found", http.StatusNotFound)
			return
		}

		// Convert to sanitized response format
		nodeResponse := NodeResponse{
			Address: nodeConfig.Address,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(nodeResponse); err != nil {
			http.Error(w, "Failed to encode node: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
