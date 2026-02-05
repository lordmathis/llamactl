package models

import "time"

type JobStatus string

const (
	JobStatusQueued      JobStatus = "queued"
	JobStatusDownloading JobStatus = "downloading"
	JobStatusCompleted   JobStatus = "completed"
	JobStatusFailed      JobStatus = "failed"
	JobStatusCancelled   JobStatus = "cancelled"
)

type Progress struct {
	BytesDownloaded int64  `json:"bytes_downloaded"`
	TotalBytes      int64  `json:"total_bytes"`
	CurrentFile     string `json:"current_file"`
}

type Job struct {
	ID          string     `json:"id"`
	Repo        string     `json:"repo"`
	Tag         string     `json:"tag"`
	Status      JobStatus  `json:"status"`
	Progress    Progress   `json:"progress"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CancelFunc  CancelFunc
}

type CancelFunc interface {
	Cancel()
}
