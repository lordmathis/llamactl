package models

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	cacheDir    string
	jobStore    *JobStore
	downloader  *Downloader
	fileManager *FileManager
}

func NewManager(cacheDir string, timeout time.Duration, version string) *Manager {
	jobStore := NewJobStore()
	fileManager := NewFileManager(cacheDir)
	progressTracker := NewProgressTracker(jobStore)
	downloader := NewDownloader(cacheDir, timeout, version, fileManager, progressTracker)

	return &Manager{
		cacheDir:    cacheDir,
		jobStore:    jobStore,
		downloader:  downloader,
		fileManager: fileManager,
	}
}

func (m *Manager) StartDownload(repo, tag string) (string, error) {
	if repo == "" {
		return "", fmt.Errorf("repo cannot be empty")
	}

	if !strings.Contains(repo, "/") {
		return "", fmt.Errorf("repo must be in format 'org/model'")
	}

	if tag == "" {
		tag = "latest"
	}

	job, err := m.jobStore.Create(repo, tag)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithCancel(context.Background())
	job.CancelFunc = cancel

	go m.downloadWorker(ctx, job)

	return job.ID, nil
}

func (m *Manager) downloadWorker(ctx context.Context, job *Job) {
	m.jobStore.UpdateStatus(job.ID, JobStatusDownloading)

	commit, err := m.downloader.FetchRefs(ctx, job.Repo, job.Tag)
	if err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	entries, err := m.downloader.FetchRepoTree(ctx, job.Repo, commit)
	if err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	plan := m.downloader.BuildDownloadPlan(job.Repo, commit, entries, "", job.Tag)
	if plan.MainGGUF == nil {
		m.jobStore.Fail(job.ID, "no GGUF file found in repo")
		return
	}

	var totalBytes int64
	for i := range plan.Tasks {
		task := &plan.Tasks[i]
		if !m.downloader.BlobExists(task.BlobPath) {
			totalBytes += task.Size
		}
	}
	m.downloader.progressTracker.AddToTotalBytes(job.ID, totalBytes)

	blobDir := filepath.Join(m.fileManager.HFRepoDir(job.Repo), "blobs")
	snapshotDir := filepath.Join(m.fileManager.HFRepoDir(job.Repo), "snapshots", commit)
	refsDir := filepath.Join(m.fileManager.HFRepoDir(job.Repo), "refs")

	for _, dir := range []string{blobDir, snapshotDir, refsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			m.jobStore.Fail(job.ID, fmt.Sprintf("failed to create directory: %v", err))
			return
		}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, maxConcurrentDownloads)

	for i := range plan.Tasks {
		task := &plan.Tasks[i]
		if m.downloader.BlobExists(task.BlobPath) && task.OID != "" {
			if err := m.fileManager.HFCreateSymlink(task.BlobPath, task.SnapshotPath); err != nil {
				log.Printf("Warning: failed to create symlink for %s: %v", task.Filename, err)
			}
			continue
		}
		if task.OID == "" && m.downloader.BlobExists(task.SnapshotPath) {
			continue
		}

		wg.Add(1)
		go func(t *HFDownloadTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := m.downloader.DownloadBlob(ctx, job.ID, t); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to download %s: %w", t.Filename, err)
				}
				mu.Unlock()
				return
			}

			if err := m.fileManager.HFCreateSymlink(t.BlobPath, t.SnapshotPath); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to create symlink for %s: %w", t.Filename, err)
				}
				mu.Unlock()
			}
		}(task)
	}

	wg.Wait()

	if firstErr != nil {
		m.jobStore.Fail(job.ID, firstErr.Error())
		return
	}

	if err := m.downloader.WriteRefFile(job.Repo, job.Tag, commit); err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to write ref file: %v", err))
		return
	}

	job.ModelPath = plan.MainGGUF.SnapshotPath
	m.jobStore.Complete(job.ID)
}

func (m *Manager) GetJob(jobID string) (*Job, error) {
	return m.jobStore.Get(jobID)
}

func (m *Manager) CancelJob(jobID string) error {
	return m.jobStore.Cancel(jobID)
}

func (m *Manager) ListJobs() []*Job {
	return m.jobStore.List()
}

func (m *Manager) DeleteJob(jobID string) error {
	return m.jobStore.Delete(jobID)
}

func (m *Manager) Close() {
	m.jobStore.Close()
}
