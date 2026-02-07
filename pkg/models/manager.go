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

	manifest, manifestData, err := m.downloader.FetchManifest(ctx, job.Repo, job.Tag)
	if err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	if manifest.GGUFFile == nil {
		m.jobStore.Fail(job.ID, "no GGUF file in manifest")
		return
	}

	// Save manifest to cache
	if err := m.downloader.SaveManifest(job.Repo, job.Tag, manifestData); err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to save manifest: %v", err))
		return
	}

	cleanup := newTempFileCleanup()
	defer cleanup.cleanupAll()

	safeGGUFFilename := filepath.Base(manifest.GGUFFile.RFilename)

	// Download main GGUF file
	mainTempPath := m.fileManager.GetPath(job.Repo, safeGGUFFilename+".tmp")
	cleanup.add(mainTempPath)
	mainURL := fmt.Sprintf(huggingFaceFileURLFmt, job.Repo, manifest.GGUFFile.RFilename)
	mainETag, err := m.downloader.Download(ctx, job.ID, mainURL, mainTempPath, safeGGUFFilename)
	if err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to download GGUF file: %v", err))
		return
	}

	splitCount, err := m.downloader.ParseSplitCount(mainTempPath)
	if err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to parse split count: %v", err))
		return
	}

	// Download split files if any
	var splitETags map[string]string
	if splitCount > 1 {
		var err error
		splitETags, err = m.downloadSplitFiles(ctx, job.ID, job.Repo, safeGGUFFilename, splitCount, cleanup)
		if err != nil {
			m.jobStore.Fail(job.ID, err.Error())
			return
		}
	}

	// Download preset.ini (optional, ignore errors)
	presetTempPath := m.fileManager.GetPath(job.Repo, "preset.ini.tmp")
	cleanup.add(presetTempPath)
	presetURL := fmt.Sprintf(huggingFaceFileURLFmt, job.Repo, "preset.ini")
	if presetETag, err := m.downloader.Download(ctx, job.ID, presetURL, presetTempPath, "preset.ini"); err == nil {
		presetFinalPath := m.fileManager.GetPath(job.Repo, "preset.ini")
		if err := os.Rename(presetTempPath, presetFinalPath); err == nil {
			cleanup.remove(presetTempPath)
			// Save ETag for preset file
			m.downloader.SaveETag(job.Repo, "preset.ini", presetETag)
		}
	}

	// Download mmproj file if present in manifest
	var mmprojETag string
	if manifest.MMProjFile != nil {
		mmprojFilename := filepath.Base(manifest.MMProjFile.RFilename)
		mmprojTempPath := m.fileManager.GetPath(job.Repo, mmprojFilename+".tmp")
		cleanup.add(mmprojTempPath)
		mmprojURL := fmt.Sprintf(huggingFaceFileURLFmt, job.Repo, manifest.MMProjFile.RFilename)
		mmprojETag, err = m.downloader.Download(ctx, job.ID, mmprojURL, mmprojTempPath, mmprojFilename)
		if err != nil {
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

	// Save ETag for main GGUF file
	if err := m.downloader.SaveETag(job.Repo, safeGGUFFilename, mainETag); err != nil {
		m.jobStore.Fail(job.ID, fmt.Sprintf("failed to save etag: %v", err))
		return
	}

	// Rename split files to final paths
	if err := m.fileManager.RenameSplitFiles(job.Repo, safeGGUFFilename, splitCount, cleanup); err != nil {
		m.jobStore.Fail(job.ID, err.Error())
		return
	}

	// Save ETags for split files (after rename)
	for splitFilename, etag := range splitETags {
		if err := m.downloader.SaveETag(job.Repo, splitFilename, etag); err != nil {
			m.jobStore.Fail(job.ID, fmt.Sprintf("failed to save split file etag: %v", err))
			return
		}
	}

	// Save ETag for mmproj file (after rename)
	if manifest.MMProjFile != nil {
		mmprojFilename := filepath.Base(manifest.MMProjFile.RFilename)
		if err := m.downloader.SaveETag(job.Repo, mmprojFilename, mmprojETag); err != nil {
			m.jobStore.Fail(job.ID, fmt.Sprintf("failed to save mmproj etag: %v", err))
			return
		}
	}

	m.jobStore.Complete(job.ID)
}

func (m *Manager) downloadSplitFiles(ctx context.Context, jobID, repo, baseFilename string, splitCount int, cleanup *tempFileCleanup) (map[string]string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	etags := make(map[string]string)

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
			etag, err := m.downloader.Download(ctx, jobID, url, tempPath, splitFilename)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("failed to download split file %d: %w", part, err)
				}
				mu.Unlock()
			} else {
				mu.Lock()
				etags[splitFilename] = etag
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return etags, firstErr
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
	files map[string]bool
	mu    sync.Mutex
}

func newTempFileCleanup() *tempFileCleanup {
	return &tempFileCleanup{
		files: make(map[string]bool),
	}
}

func (t *tempFileCleanup) add(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.files[path] = true
}

func (t *tempFileCleanup) remove(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.files, path)
}

func (t *tempFileCleanup) cleanupAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for f := range t.files {
		os.Remove(f)
	}
}
