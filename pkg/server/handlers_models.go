package server

import (
	"encoding/json"
	"llamactl/pkg/models"
	"net/http"
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

// DownloadModel godoc
// @Summary Download a model from a repository
// @Description Initiates the download of a model from a specified repository and tag. Returns a job ID to track progress.
// @Tags Models
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body DownloadRequest true "Download request"
// @Success 202 {object} DownloadResponse "Download initiated"
// @Failure 400 {string} string "Invalid request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/download [post]
func (h *Handler) DownloadModel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
// @Description Returns a list of all models currently cached on the server
// @Tags Models
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} string "List of cached models"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models [get]
func (h *Handler) ListModels() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models, err := h.modelManager.ListCached()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "scan_failed", err.Error())
			return
		}

		writeJSON(w, http.StatusOK, models)
	}
}

// DeleteModel godoc
// @Summary Delete a cached model
// @Description Deletes a cached model by its repository and optional tag
// @Tags Models
// @Security ApiKeyAuth
// @Param repo query string true "Repository"
// @Param tag query string false "Tag"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Model not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models [delete]
func (h *Handler) DeleteModel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
// @Param id path string true "Job ID"
// @Success 200 {object} JobResponse "Job details"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Job not found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/jobs/{id} [get]
func (h *Handler) GetJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
// @Success 200 {object} ListJobsResponse "List of jobs"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/jobs [get]
func (h *Handler) ListJobs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
// @Param id path string true "Job ID"
// @Success 204 "No Content"
// @Failure 400 {string} string "Invalid request"
// @Failure 404 {string} string "Job not found"
// @Failure 409 {string} string "Cannot cancel job with current status"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/v1/backends/llama-cpp/models/jobs/{id} [delete]
func (h *Handler) CancelJob() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
