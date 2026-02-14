package server

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/models"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// DownloadRequest represents the request body for initiating a model download
type DownloadRequest struct {
	Repo string `json:"repo"`
}

// DownloadResponse represents the response after initiating a model download
type DownloadResponse struct {
	JobID string `json:"job_id"`
	Repo  string `json:"repo"`
	Tag   string `json:"tag"`
}

// JobResponse represents the details of a download job for API responses
type JobResponse struct {
	ID          string          `json:"id"`
	Repo        string          `json:"repo"`
	Tag         string          `json:"tag"`
	Status      string          `json:"status"`
	Progress    models.Progress `json:"progress"`
	Error       string          `json:"error,omitempty"`
	CreatedAt   int64           `json:"created_at"`
	CompletedAt *int64          `json:"completed_at,omitempty"`
}

// ListJobsResponse represents the response for listing all download jobs
type ListJobsResponse struct {
	Jobs []JobResponse `json:"jobs"`
}

// forwardToNode forwards an HTTP request to a remote node
func (h *Handler) forwardToNode(nodeName string, w http.ResponseWriter, r *http.Request) bool {
	node, exists := h.cfg.Nodes[nodeName]
	if !exists {
		writeError(w, http.StatusNotFound, "node_not_found", "Node not found")
		return false
	}

	targetURL, err := url.Parse(node.Address)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid_node_address", "Failed to parse node address")
		return false
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if node.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+node.APIKey)
		}
	}

	proxy.ServeHTTP(w, r)
	return true
}

// DownloadModel godoc
// @Summary Download a model from a repository
// @Description Initiates the download of a model from a specified repository and tag. Returns a job ID to track progress.
// @Tags Models
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param node query string false "Node name to forward the request to"
// @Param request body DownloadRequest true "Download request"
// @Success 202 {object} DownloadResponse "Download initiated"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Node not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/download [post]
func (h *Handler) DownloadModel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.URL.Query().Get("node")
		if nodeName != "" {
			if h.forwardToNode(nodeName, w, r) {
				return
			}
			return
		}

		var req DownloadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "Failed to parse request body")
			return
		}

		if req.Repo == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "repo is required")
			return
		}

		// Parse repo and tag from format "org/model:tag"
		repo := req.Repo
		tag := ""
		if colonIdx := strings.LastIndex(req.Repo, ":"); colonIdx != -1 {
			repo = req.Repo[:colonIdx]
			tag = req.Repo[colonIdx+1:]
		}

		if !strings.Contains(repo, "/") {
			writeError(w, http.StatusBadRequest, "invalid_request", "repo must be in format 'org/model' or 'org/model:tag'")
			return
		}

		jobID, err := h.modelManager.StartDownload(repo, tag)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "download_failed", err.Error())
			return
		}

		response := DownloadResponse{
			JobID: jobID,
			Repo:  repo,
			Tag:   tag,
		}

		writeJSON(w, http.StatusAccepted, response)
	}
}

// ListModels godoc
// @Summary List cached models
// @Description Returns a list of all models currently cached on the server. If node parameter is specified, only returns models from that node. If no node is specified, returns models from all nodes aggregated.
// @Tags Models
// @Security ApiKeyAuth
// @Produce json
// @Param node query string false "Node name to query (if not specified, queries all nodes)"
// @Success 200 {object} []models.CachedModel "List of cached models from the specified node or aggregated from all nodes"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models [get]
func (h *Handler) ListModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.URL.Query().Get("node")

		if nodeName != "" {
			if h.forwardToNode(nodeName, w, r) {
				return
			}
			return
		}

		var allModels []models.CachedModel

		localNodeName := h.cfg.LocalNode
		if localNodeName == "" {
			localNodeName = "local"
		}

		localModels, err := h.modelManager.ListCached(localNodeName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scan_failed", err.Error())
			return
		}

		allModels = append(allModels, localModels...)

		for name, node := range h.cfg.Nodes {
			if name == h.cfg.LocalNode {
				continue
			}

			nodeModels, err := h.fetchModelsFromNode(node, name)
			if err != nil {
				continue
			}

			allModels = append(allModels, nodeModels...)
		}

		writeJSON(w, http.StatusOK, allModels)
	}
}

func (h *Handler) fetchModelsFromNode(node config.NodeConfig, nodeName string) ([]models.CachedModel, error) {
	targetURL, err := url.Parse(node.Address)
	if err != nil {
		return nil, err
	}

	reqURL := targetURL.JoinPath("/api/v1/backends/llama-cpp/models")
	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	if node.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+node.APIKey)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var models []models.CachedModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, err
	}

	for i := range models {
		if models[i].Node == "" {
			models[i].Node = nodeName
		}
	}

	return models, nil
}

// DeleteModel godoc
// @Summary Delete a cached model
// @Description Deletes a cached model by its repository and optional tag
// @Tags Models
// @Security ApiKeyAuth
// @Param node query string false "Node name to forward the request to"
// @Param repo query string true "Repository"
// @Param tag query string false "Tag"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Model not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models [delete]
func (h *Handler) DeleteModel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.URL.Query().Get("node")
		if nodeName != "" {
			if h.forwardToNode(nodeName, w, r) {
				return
			}
			return
		}

		repo := r.URL.Query().Get("repo")
		if repo == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "repo is required")
			return
		}

		tag := r.URL.Query().Get("tag")

		err := h.modelManager.DeleteModel(repo, tag)
		if err != nil {
			if strings.Contains(err.Error(), "model not found") {
				writeError(w, http.StatusNotFound, "model_not_found", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "delete_failed", err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// GetJob godoc
// @Summary Get details of a specific download job
// @Description Returns the details of a download job by its ID
// @Tags Models
// @Security ApiKeyAuth
// @Produce json
// @Param node query string false "Node name to forward the request to"
// @Param id path string true "Job ID"
// @Success 200 {object} JobResponse "Job details"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Job not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/jobs/{id} [get]
func (h *Handler) GetJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.URL.Query().Get("node")
		if nodeName != "" {
			if h.forwardToNode(nodeName, w, r) {
				return
			}
			return
		}

		jobID := chi.URLParam(r, "id")
		if jobID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "job ID is required")
			return
		}

		job, err := h.modelManager.GetJob(jobID)
		if err != nil {
			if strings.Contains(err.Error(), "job not found") {
				writeError(w, http.StatusNotFound, "job_not_found", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		response := jobToResponse(job)
		writeJSON(w, http.StatusOK, response)
	}
}

// ListJobs godoc
// @Summary List all model download jobs
// @Description Returns a list of all model download jobs with their details
// @Tags Models
// @Security ApiKeyAuth
// @Produce json
// @Param node query string false "Node name to forward the request to"
// @Success 200 {object} ListJobsResponse "List of jobs"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/jobs [get]
func (h *Handler) ListJobs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.URL.Query().Get("node")
		if nodeName != "" {
			if h.forwardToNode(nodeName, w, r) {
				return
			}
			return
		}

		jobs := h.modelManager.ListJobs()

		response := ListJobsResponse{
			Jobs: make([]JobResponse, len(jobs)),
		}

		for i, job := range jobs {
			response.Jobs[i] = jobToResponse(job)
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// CancelJob godoc
// @Summary Cancel an ongoing model download job
// @Description Cancels a model download job by its ID. Only jobs that are in progress can be cancelled.
// @Tags Models
// @Security ApiKeyAuth
// @Param node query string false "Node name to forward the request to"
// @Param id path string true "Job ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Job not found"
// @Failure 409 {string} string "Cannot cancel job with current status"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/jobs/{id} [delete]
func (h *Handler) CancelJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeName := r.URL.Query().Get("node")
		if nodeName != "" {
			if h.forwardToNode(nodeName, w, r) {
				return
			}
			return
		}

		jobID := chi.URLParam(r, "id")
		if jobID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "job ID is required")
			return
		}

		err := h.modelManager.CancelJob(jobID)
		if err != nil {
			if strings.Contains(err.Error(), "job not found") {
				writeError(w, http.StatusNotFound, "job_not_found", err.Error())
				return
			}
			if strings.Contains(err.Error(), "cannot cancel job with status") {
				writeError(w, http.StatusConflict, "cannot_cancel", err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func jobToResponse(job *models.Job) JobResponse {
	var completedAt *int64
	if job.CompletedAt != nil {
		ts := job.CompletedAt.Unix()
		completedAt = &ts
	}

	return JobResponse{
		ID:          job.ID,
		Repo:        job.Repo,
		Tag:         job.Tag,
		Status:      string(job.Status),
		Progress:    job.Progress,
		Error:       job.Error,
		CreatedAt:   job.CreatedAt.Unix(),
		CompletedAt: completedAt,
	}
}
