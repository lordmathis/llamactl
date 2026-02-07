package models

import (
	"context"
	"testing"
	"time"
)

func TestJobStore_JobStateTransitions(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	// Create job
	job, err := store.Create("org/model", "Q4_K_M")
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	if job.Status != JobStatusQueued {
		t.Errorf("initial status = %v, want %v", job.Status, JobStatusQueued)
	}

	// Update to downloading
	store.UpdateStatus(job.ID, JobStatusDownloading)
	retrieved, _ := store.Get(job.ID)
	if retrieved.Status != JobStatusDownloading {
		t.Errorf("status after update = %v, want %v", retrieved.Status, JobStatusDownloading)
	}

	// Complete job
	store.Complete(job.ID)
	retrieved, _ = store.Get(job.ID)
	if retrieved.Status != JobStatusCompleted {
		t.Errorf("status after complete = %v, want %v", retrieved.Status, JobStatusCompleted)
	}

	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should be set after completion")
	}
}

func TestJobStore_FailJob(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	job, _ := store.Create("org/model", "Q4_K_M")

	errorMsg := "download failed: network timeout"
	store.Fail(job.ID, errorMsg)

	retrieved, _ := store.Get(job.ID)
	if retrieved.Status != JobStatusFailed {
		t.Errorf("status = %v, want %v", retrieved.Status, JobStatusFailed)
	}

	if retrieved.Error != errorMsg {
		t.Errorf("error = %q, want %q", retrieved.Error, errorMsg)
	}

	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should be set after failure")
	}
}

func TestJobStore_CancelJob(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	job, _ := store.Create("org/model", "Q4_K_M")
	job.CancelFunc = cancel

	// Update job in store with cancel func
	store.mutex.Lock()
	store.jobs[job.ID].CancelFunc = cancel
	store.mutex.Unlock()

	err := store.Cancel(job.ID)
	if err != nil {
		t.Errorf("Cancel failed: %v", err)
	}

	retrieved, _ := store.Get(job.ID)
	if retrieved.Status != JobStatusCancelled {
		t.Errorf("status = %v, want %v", retrieved.Status, JobStatusCancelled)
	}

	if retrieved.CompletedAt == nil {
		t.Error("CompletedAt should be set after cancellation")
	}

	// Verify context was cancelled
	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("context should be cancelled")
	}
}

func TestJobStore_CannotCancelCompletedJob(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	job, _ := store.Create("org/model", "Q4_K_M")
	store.Complete(job.ID)

	err := store.Cancel(job.ID)
	if err == nil {
		t.Error("expected error when cancelling completed job")
	}
}

func TestJobStore_CannotDeleteActiveJob(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	job, _ := store.Create("org/model", "Q4_K_M")
	store.UpdateStatus(job.ID, JobStatusDownloading)

	err := store.Delete(job.ID)
	if err == nil {
		t.Error("expected error when deleting active job")
	}

	// Should still exist
	_, err = store.Get(job.ID)
	if err != nil {
		t.Error("job should still exist after failed delete")
	}
}

func TestJobStore_DeleteCompletedJob(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	job, _ := store.Create("org/model", "Q4_K_M")
	store.Complete(job.ID)

	err := store.Delete(job.ID)
	if err != nil {
		t.Errorf("failed to delete completed job: %v", err)
	}

	// Should not exist
	_, err = store.Get(job.ID)
	if err == nil {
		t.Error("expected error getting deleted job")
	}
}

func TestJobStore_ListJobs(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	// Create multiple jobs
	job1, _ := store.Create("org/model1", "Q4_K_M")
	job2, _ := store.Create("org/model2", "Q8_0")
	job3, _ := store.Create("org/model3", "latest")

	jobs := store.List()
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}

	// Verify all jobs are present
	ids := make(map[string]bool)
	for _, job := range jobs {
		ids[job.ID] = true
	}

	if !ids[job1.ID] || !ids[job2.ID] || !ids[job3.ID] {
		t.Error("not all jobs returned in list")
	}
}

func TestJobStore_GetReturnsCopy(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	job, _ := store.Create("org/model", "Q4_K_M")

	// Get returns a copy with nil CancelFunc
	retrieved, _ := store.Get(job.ID)
	if retrieved.CancelFunc != nil {
		t.Error("Get should return job with nil CancelFunc")
	}
}

func TestJobStore_CleanupOldJobs(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	// Create and complete a job
	job, _ := store.Create("org/model", "Q4_K_M")
	store.Complete(job.ID)

	// Manually set CompletedAt to old timestamp
	store.mutex.Lock()
	oldTime := time.Now().Add(-25 * time.Hour) // Older than 24h retention
	store.jobs[job.ID].CompletedAt = &oldTime
	store.mutex.Unlock()

	// Trigger cleanup
	store.cleanupOldJobs()

	// Job should be deleted
	_, err := store.Get(job.ID)
	if err == nil {
		t.Error("old job should be deleted by cleanup")
	}
}

func TestJobStore_CleanupDoesNotDeleteActiveJobs(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	// Create active job
	job, _ := store.Create("org/model", "Q4_K_M")
	store.UpdateStatus(job.ID, JobStatusDownloading)

	// Trigger cleanup
	store.cleanupOldJobs()

	// Job should still exist
	_, err := store.Get(job.ID)
	if err != nil {
		t.Error("active job should not be deleted by cleanup")
	}
}

func TestJobStore_CleanupDoesNotDeleteRecentJobs(t *testing.T) {
	store := NewJobStore()
	defer store.Close()

	// Create and complete a job
	job, _ := store.Create("org/model", "Q4_K_M")
	store.Complete(job.ID)

	// Trigger cleanup immediately (job is recent)
	store.cleanupOldJobs()

	// Job should still exist
	_, err := store.Get(job.ID)
	if err != nil {
		t.Error("recent completed job should not be deleted by cleanup")
	}
}
