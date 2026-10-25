package models

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var repoNamePattern = regexp.MustCompile(
	`^[A-Za-z0-9][A-Za-z0-9._-]*(/[A-Za-z0-9][A-Za-z0-9._-]*)+$`)

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

	m := &Manager{
		cacheDir:    cacheDir,
		jobStore:    jobStore,
		downloader:  downloader,
		fileManager: fileManager,
	}
	// No download can be live at construction, so any scratch file left
	// behind by a previous process is stale and safe to remove.
	m.sweepStaleTempFiles()
	return m
}

func (m *Manager) sweepStaleTempFiles() {
	_ = filepath.WalkDir(m.fileManager.HFHubRoot(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("stale temp sweep: %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.Contains(name, ".tmp-") && !strings.Contains(name, ".lnk-") {
			return nil
		}
		if err := os.Remove(path); err != nil {
			log.Printf("stale temp sweep: failed to remove %s: %v", path, err)
			return nil
		}
		log.Printf("stale temp sweep: removed %s", path)
		return nil
	})
}

func (m *Manager) StartDownload(repo, tag string, format ModelFormat) (string, error) {
	if format == "" {
		format = FormatGGUF
	}
	if !IsValidFormat(format) {
		return "", fmt.Errorf("invalid format %q (must be %q or %q)", format, FormatGGUF, FormatSafetensors)
	}
	if !repoNamePattern.MatchString(repo) {
		return "", fmt.Errorf("repo must be in format 'org/model'")
	}

	if tag == "" {
		tag = "main"
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

	plan, err := m.downloader.BuildDownloadPlan(job.Repo, commit, entries, "", job.Tag, format)
	if err != nil {
		log.Printf("[%s] Failed to build download plan: %v", job.ID, err)
		m.jobStore.Fail(job.ID, err.Error())
		return
	}
	if format != FormatSafetensors && plan.MainGGUF == nil {
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
		if !m.downloader.BlobValid(task) {
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
		if m.downloader.BlobValid(task) {
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
		m.jobStore.SetModelPath(job.ID, filepath.Join(m.fileManager.HFRepoDir(job.Repo), "snapshots", commit))
	} else if plan.MainGGUF != nil {
		m.jobStore.SetModelPath(job.ID, plan.MainGGUF.SnapshotPath)
	} else {
		m.jobStore.Fail(job.ID, "no main GGUF file in download plan")
		return
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
