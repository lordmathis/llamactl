package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"

	"llamactl/pkg/config"
)

// NodeResponse represents a node configuration in API responses
type NodeResponse struct {
	Address string `json:"address"`
}

// NodeHealthStatus represents the health status of a node
type NodeHealthStatus string

const (
	NodeStatusHealthy     NodeHealthStatus = "healthy"
	NodeStatusUnreachable NodeHealthStatus = "unreachable"
)

// NodeHealthResponse represents the health status of a node in API responses
type NodeHealthResponse struct {
	Status    NodeHealthStatus `json:"status"`
	LatencyMS int64            `json:"latency_ms"` // no omitempty: 0 is meaningful (local node, sub-ms ping)
	Error     string           `json:"error,omitempty"`
	CheckedAt int64            `json:"checked_at"`
}

// ListNodes godoc
// @Summary List all configured nodes
// @Description Returns a map of all nodes configured in the server (node name -> node config)
// @Tags Nodes
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
// @Tags Nodes
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

// GetNodeHealth godoc
// @Summary Check the health of a specific node
// @Description Local node is healthy by definition; remote nodes are pinged. Unreachable nodes get a 200 with status "unreachable", not an error.
// @Tags Nodes
// @Security ApiKeyAuth
// @Produces json
// @Param name path string true "Node Name"
// @Success 200 {object} NodeHealthResponse "Node health status"
// @Failure 400 {string} string "Invalid name format"
// @Failure 404 {string} string "Node not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/nodes/{name}/health [get]
func (h *Handler) GetNodeHealth() http.HandlerFunc {
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

		// this request was served by the local node, so it is up
		if name == h.cfg.LocalNode {
			writeJSON(w, http.StatusOK, NodeHealthResponse{
				Status:    NodeStatusHealthy,
				CheckedAt: time.Now().Unix(),
			})
			return
		}

		writeJSON(w, http.StatusOK, h.checkRemoteNodeHealth(nodeConfig))
	}
}

// checkRemoteNodeHealth pings the node's version endpoint; a 200 also proves
// the configured API key works
func (h *Handler) checkRemoteNodeHealth(node config.NodeConfig) NodeHealthResponse {
	health := NodeHealthResponse{
		Status:    NodeStatusUnreachable,
		CheckedAt: time.Now().Unix(),
	}

	if node.Address == "" {
		health.Error = "no address configured"
		return health
	}

	targetURL, err := url.Parse(node.Address)
	if err != nil {
		health.Error = "invalid node address"
		return health
	}

	reqURL := targetURL.JoinPath("/api/v1/version")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		health.Error = err.Error()
		return health
	}
	if node.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+node.APIKey)
	}

	start := time.Now()
	resp, err := h.httpClient.Do(req)
	if err != nil {
		health.Error = err.Error()
		return health
	}

	health.LatencyMS = time.Since(start).Milliseconds()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		health.Error = fmt.Sprintf("unexpected status code: %d", resp.StatusCode)
		return health
	}

	health.Status = NodeStatusHealthy
	return health
}
