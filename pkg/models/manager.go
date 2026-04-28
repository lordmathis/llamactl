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

func (m *Manager) StartDownload(repo, tag string, format ModelFormat) (string, error) {
	if repo == "" {
		return "", fmt.Errorf("repo cannot be empty")
	}

	if format == FormatGGUF && !strings.Contains(repo, "/") {
		return "", fmt.Errorf("repo must be in format 'org/model'")
	}

	if !strings.Contains(repo, "/") {
		return "", fmt.Errorf("repo must be in format 'org/model'")
	}

	if tag == "" {
		tag = "main"
	}

	if format == "" {
		format = FormatGGUF
	}

	job, err := m.jobStore.Create(repo, tag)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithCancel(context.Background())
	job.CancelFunc = cancel

	go m.downloadWorker(ctx, job, format)

	return job.ID, nil
}

func (m *Manager) downloadWorker(ctx context.Context, job *Job, format ModelFormat) {
	log.Printf("[%s] Starting download: %s:%s", job.ID, job.Repo, job.Tag)
	m.jobStore.UpdateStatus(job.ID, JobStatusDownloading)

	commit, err := m.downloader.FetchRefs(ctx, job.Repo, job.Tag)
	if err != nil {
		log.Printf("[%s] Failed to fetch refs: %v", job.ID, err)
		m.jobStore.Fail(job.ID, err.Error())
		return
	}
	log.Printf("[%s] Resolved %s to commit %s", job.ID, job.Tag, commit)

	entries, err := m.downloader.FetchRepoTree(ctx, job.Repo, commit)
	if err != nil {
		log.Printf("[%s] Failed to fetch repo tree: %v", job.ID, err)
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	plan := m.downloader.BuildDownloadPlan(job.Repo, commit, entries, "", job.Tag, format)
	if format == FormatGGUF && plan.MainGGUF == nil {
		log.Printf("[%s] Error: No GGUF files found in repo matching criteria", job.ID)
		m.jobStore.Fail(job.ID, "no GGUF file found in repo")
		return
	}
	if format == FormatSafetensors && len(plan.Tasks) == 0 {
		log.Printf("[%s] Error: No safetensors or fallback files found in repo", job.ID)
		m.jobStore.Fail(job.ID, "no safetensors files found in repo")
		return
	}

	if err := m.downloader.ResolveNonLFSOids(ctx, plan, job.Repo); err != nil {
		log.Printf("[%s] Failed to resolve file metadata: %v", job.ID, err)
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	var totalBytes int64
	var tasksToDownload []*HFDownloadTask
	for i := range plan.Tasks {
		task := &plan.Tasks[i]
		if !m.downloader.BlobExists(task.BlobPath) {
			totalBytes += task.Size
			tasksToDownload = append(tasksToDownload, task)
		}
	}
	m.downloader.progressTracker.AddToTotalBytes(job.ID, totalBytes)
	log.Printf("[%s] Plan built: %d files to download (%d bytes)", job.ID, len(tasksToDownload), totalBytes)

	blobDir := filepath.Join(m.fileManager.HFRepoDir(job.Repo), "blobs")
	snapshotDir := filepath.Join(m.fileManager.HFRepoDir(job.Repo), "snapshots", commit)
	refsDir := filepath.Join(m.fileManager.HFRepoDir(job.Repo), "refs")

	for _, dir := range []string{blobDir, snapshotDir, refsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("[%s] Failed to create directory %s: %v", job.ID, dir, err)
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

		// If blob exists, just ensure the symlink is there
		if m.downloader.BlobExists(task.BlobPath) {
			if err := m.fileManager.HFCreateSymlink(task.BlobPath, task.SnapshotPath); err != nil {
				log.Printf("[%s] Warning: failed to create symlink for %s: %v", job.ID, task.Filename, err)
			}
			continue
		}

		wg.Add(1)
		go func(t *HFDownloadTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			log.Printf("[%s] Downloading %s...", job.ID, t.Filename)
			if err := m.downloader.DownloadBlob(ctx, job.ID, t); err != nil {
				log.Printf("[%s] Download failed for %s: %v", job.ID, t.Filename, err)
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
		log.Printf("[%s] Failed to write ref file: %v", job.ID, err)
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to write ref file: %v", err))
		return
	}

	log.Printf("[%s] Download completed successfully", job.ID)
	if plan.Format == FormatSafetensors {
		job.ModelPath = filepath.Join(m.fileManager.HFRepoDir(job.Repo), "snapshots", commit)
	} else {
		job.ModelPath = plan.MainGGUF.SnapshotPath
	}
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
