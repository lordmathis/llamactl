package models

import (
	"context"
	"fmt"
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
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cache", "llama.cpp")
	}

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

	manifest, err := m.downloader.FetchManifest(ctx, job.Repo, job.Tag)
	if err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	if manifest.GGUFFile == nil {
		m.jobStore.Fail(job.ID, "no GGUF file in manifest")
		return
	}

	cleanup := newTempFileCleanup()
	defer cleanup.cleanupAll()

	safeGGUFFilename := filepath.Base(manifest.GGUFFile.RFilename)

	// Download main GGUF file
	mainTempPath := m.fileManager.GetPath(job.Repo, safeGGUFFilename+".tmp")
	cleanup.add(mainTempPath)
	mainURL := fmt.Sprintf(huggingFaceFileURLFmt, job.Repo, manifest.GGUFFile.RFilename)
	if err := m.downloader.Download(ctx, job.ID, mainURL, mainTempPath, safeGGUFFilename); err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to download GGUF file: %v", err))
		return
	}

	splitCount, err := m.downloader.ParseSplitCount(mainTempPath)
	if err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to parse split count: %v", err))
		return
	}

	// Download split files if any
	if splitCount > 1 {
		if err := m.downloadSplitFiles(ctx, job.ID, job.Repo, safeGGUFFilename, splitCount, cleanup); err != nil {
			m.jobStore.Fail(job.ID, err.Error())
			return
		}
	}

	// Download preset.ini (optional, ignore errors)
	presetTempPath := m.fileManager.GetPath(job.Repo, "preset.ini.tmp")
	cleanup.add(presetTempPath)
	presetURL := fmt.Sprintf(huggingFaceFileURLFmt, job.Repo, "preset.ini")
	if err := m.downloader.Download(ctx, job.ID, presetURL, presetTempPath, "preset.ini"); err == nil {
		presetFinalPath := m.fileManager.GetPath(job.Repo, "preset.ini")
		if err := os.Rename(presetTempPath, presetFinalPath); err == nil {
			cleanup.remove(presetTempPath)
		}
	}

	// Download mmproj file if present in manifest
	if manifest.MMProjFile != nil {
		mmprojFilename := filepath.Base(manifest.MMProjFile.RFilename)
		mmprojTempPath := m.fileManager.GetPath(job.Repo, mmprojFilename+".tmp")
		cleanup.add(mmprojTempPath)
		mmprojURL := fmt.Sprintf(huggingFaceFileURLFmt, job.Repo, manifest.MMProjFile.RFilename)
		if err := m.downloader.Download(ctx, job.ID, mmprojURL, mmprojTempPath, mmprojFilename); err != nil {
			m.jobStore.Fail(job.ID, fmt.Sprintf("failed to download mmproj file: %v", err))
			return
		}
		mmprojFinalPath := m.fileManager.GetPath(job.Repo, mmprojFilename)
		if err := os.Rename(mmprojTempPath, mmprojFinalPath); err != nil {
			m.jobStore.Fail(job.ID, fmt.Sprintf("failed to rename mmproj file: %v", err))
			return
		}
		cleanup.remove(mmprojTempPath)
	}

	// Rename main GGUF to final path
	if err := m.fileManager.RenameToFinal(mainTempPath, job.Repo, safeGGUFFilename, cleanup); err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	// Rename split files to final paths
	if err := m.fileManager.RenameSplitFiles(job.Repo, safeGGUFFilename, splitCount, cleanup); err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	m.jobStore.Complete(job.ID)
}

func (m *Manager) downloadSplitFiles(ctx context.Context, jobID, repo, baseFilename string, splitCount int, cleanup *tempFileCleanup) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// Semaphore to limit concurrent downloads
	sem := make(chan struct{}, maxConcurrentDownloads)

	for i := 2; i <= splitCount; i++ {
		wg.Add(1)
		go func(part int) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }() // Release semaphore

			splitFilename := m.fileManager.GetSplitFilename(baseFilename, part, splitCount)
			tempPath := m.fileManager.GetPath(repo, splitFilename+".tmp")

			mu.Lock()
			cleanup.add(tempPath)
			mu.Unlock()

			url := fmt.Sprintf(huggingFaceFileURLFmt, repo, splitFilename)
			if err := m.downloader.Download(ctx, jobID, url, tempPath, splitFilename); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to download split file %d: %w", part, err)
				}
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return firstErr
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

type tempFileCleanup struct {
	files []string
	mu    sync.Mutex
}

func newTempFileCleanup() *tempFileCleanup {
	return &tempFileCleanup{}
}

func (t *tempFileCleanup) add(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.files = append(t.files, path)
}

func (t *tempFileCleanup) remove(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for i, f := range t.files {
		if f == path {
			t.files = append(t.files[:i], t.files[i+1:]...)
			return
		}
	}
}

func (t *tempFileCleanup) cleanupAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, f := range t.files {
		os.Remove(f)
	}
}
