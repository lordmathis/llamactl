package server

import (
	"encoding/json"
	"llamactl/pkg/models"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type DownloadRequest struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag,omitempty"`
}

type DownloadResponse struct {
	JobID string `json:"job_id"`
	Repo  string `json:"repo"`
	Tag   string `json:"tag"`
}

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

type ListJobsResponse struct {
	Jobs []JobResponse `json:"jobs"`
}

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

		if !strings.Contains(req.Repo, "/") {
			writeError(w, http.StatusBadRequest, "invalid_request", "repo must be in format 'org/model'")
			return
		}

		if req.Tag == "" {
			req.Tag = "latest"
		}

		jobID, err := h.modelManager.StartDownload(req.Repo, req.Tag)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "download_failed", err.Error())
			return
		}

		response := DownloadResponse{
			JobID: jobID,
			Repo:  req.Repo,
			Tag:   req.Tag,
		}

		writeJSON(w, http.StatusAccepted, response)
	}
}

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
