package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

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
	ID          string             `json:"id"`
	Repo        string             `json:"repo"`
	Tag         string             `json:"tag"`
	Status      JobStatus          `json:"status"`
	Progress    Progress           `json:"progress"`
	Error       string             `json:"error,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	CancelFunc  context.CancelFunc `json:"-"`
}

type JobStore struct {
	jobs  map[string]*Job
	mutex sync.RWMutex
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*Job),
	}
}

func (s *JobStore) Create(repo, tag string) (*Job, error) {
	jobID, err := s.generateJobID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate job ID: %w", err)
	}

	job := &Job{
		ID:        jobID,
		Repo:      repo,
		Tag:       tag,
		Status:    JobStatusQueued,
		Progress:  Progress{},
		CreatedAt: time.Now(),
	}

	s.mutex.Lock()
	s.jobs[jobID] = job
	s.mutex.Unlock()

	return job, nil
}

func (s *JobStore) Get(jobID string) (*Job, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	jobCopy := *job
	jobCopy.CancelFunc = nil

	return &jobCopy, nil
}

func (s *JobStore) List() []*Job {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobCopy := *job
		jobCopy.CancelFunc = nil
		jobs = append(jobs, &jobCopy)
	}

	return jobs
}

func (s *JobStore) Delete(jobID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status == JobStatusDownloading || job.Status == JobStatusQueued {
		return fmt.Errorf("cannot delete job with status: %s", job.Status)
	}

	delete(s.jobs, jobID)
	return nil
}

func (s *JobStore) UpdateStatus(jobID string, status JobStatus) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if job, ok := s.jobs[jobID]; ok {
		job.Status = status
	}
}

func (s *JobStore) Complete(jobID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	if job, ok := s.jobs[jobID]; ok {
		job.Status = JobStatusCompleted
		job.CompletedAt = &now
	}
}

func (s *JobStore) Fail(jobID string, errMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	if job, ok := s.jobs[jobID]; ok {
		job.Status = JobStatusFailed
		job.Error = errMsg
		job.CompletedAt = &now
	}
}

func (s *JobStore) Cancel(jobID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled {
		return fmt.Errorf("cannot cancel job with status: %s", job.Status)
	}

	if job.CancelFunc != nil {
		job.CancelFunc()
	}

	now := time.Now()
	job.Status = JobStatusCancelled
	job.CompletedAt = &now

	return nil
}

func (s *JobStore) generateJobID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
