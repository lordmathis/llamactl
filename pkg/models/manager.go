package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// progressChannelBuffer is the buffer size for progress update channels.
	// This prevents blocking on progress updates during file downloads.
	progressChannelBuffer = 100
)

type Manager struct {
	cacheDir   string
	jobs       map[string]*Job
	jobsMutex  sync.RWMutex
	downloader *Downloader
	version    string
}

func NewManager(cacheDir string, timeout time.Duration, version string) *Manager {
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cache", "llama.cpp")
	}

	return &Manager{
		cacheDir:   cacheDir,
		jobs:       make(map[string]*Job),
		downloader: NewDownloader(cacheDir, timeout, version),
		version:    version,
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

	jobID, err := m.generateJobID()
	if err != nil {
		return "", fmt.Errorf("failed to generate job ID: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	newJob := &Job{
		ID:         jobID,
		Repo:       repo,
		Tag:        tag,
		Status:     JobStatusQueued,
		Progress:   Progress{},
		CreatedAt:  time.Now(),
		CancelFunc: cancel,
	}

	m.jobsMutex.Lock()
	m.jobs[jobID] = newJob
	m.jobsMutex.Unlock()

	go m.downloadWorker(ctx, newJob)

	return jobID, nil
}

func (m *Manager) downloadWorker(ctx context.Context, job *Job) {
	m.updateJobStatus(job.ID, JobStatusDownloading)

	manifest, err := m.downloader.FetchManifest(ctx, job.Repo, job.Tag)
	if err != nil {
		m.failJob(job.ID, err.Error())
		return
	}

	if manifest.GGUFFile == nil {
		m.failJob(job.ID, "no GGUF file in manifest")
		return
	}

	// Sanitize filename to prevent path traversal attacks
	safeGGUFFilename := filepath.Base(manifest.GGUFFile.RFilename)

	tmpFiles := []string{}
	defer func() {
		for _, f := range tmpFiles {
			os.Remove(f)
		}
	}()

	tempDest := m.downloader.getCacheFilename(job.Repo, safeGGUFFilename+".tmp")
	tempDest = filepath.Join(m.cacheDir, tempDest)

	progressChan := make(chan int64, progressChannelBuffer)
	go m.trackProgress(job.ID, progressChan)

	tmpFiles = append(tmpFiles, tempDest)

	url := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", job.Repo, manifest.GGUFFile.RFilename)

	m.updateJobCurrentFile(job.ID, safeGGUFFilename)

	contentLength, err := m.downloader.DownloadFile(ctx, url, tempDest, progressChan)
	if err != nil {
		m.failJob(job.ID, fmt.Sprintf("failed to download GGUF file: %v", err))
		return
	}

	close(progressChan)

	// Update total bytes with main file size
	if contentLength > 0 {
		m.addToTotalBytes(job.ID, contentLength)
	}

	splitCount, err := m.downloader.ParseSplitCount(tempDest)
	if err != nil {
		m.failJob(job.ID, fmt.Sprintf("failed to parse split count: %v", err))
		return
	}

	// Download split files in parallel
	if splitCount > 1 {
		var wg sync.WaitGroup
		var splitTempFiles []string
		var splitMutex sync.Mutex
		var firstErr error
		var errOnce sync.Once

		for i := 2; i <= splitCount; i++ {
			wg.Add(1)
			go func(part int) {
				defer wg.Done()

				splitFilename := m.getSplitFilename(safeGGUFFilename, part, splitCount)
				splitTempDest := m.downloader.getCacheFilename(job.Repo, splitFilename+".tmp")
				splitTempDest = filepath.Join(m.cacheDir, splitTempDest)

				splitMutex.Lock()
				splitTempFiles = append(splitTempFiles, splitTempDest)
				splitMutex.Unlock()

				splitURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", job.Repo, splitFilename)

				progressChan := make(chan int64, progressChannelBuffer)
				go m.trackProgress(job.ID, progressChan)

				m.updateJobCurrentFile(job.ID, splitFilename)

				contentLength, err := m.downloader.DownloadFile(ctx, splitURL, splitTempDest, progressChan)
				if err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("failed to download split file %d: %w", part, err)
					})
					close(progressChan)
					return
				}

				close(progressChan)

				// Update total bytes with split file size
				if contentLength > 0 {
					m.addToTotalBytes(job.ID, contentLength)
				}
			}(i)
		}

		wg.Wait()

		// Check for errors
		if firstErr != nil {
			m.failJob(job.ID, firstErr.Error())
			return
		}

		// Add split files to cleanup list
		tmpFiles = append(tmpFiles, splitTempFiles...)
	}

	// Download optional preset.ini
	presetURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/preset.ini", job.Repo)
	presetTempDest := m.downloader.getCacheFilename(job.Repo, "preset.ini.tmp")
	presetTempDest = filepath.Join(m.cacheDir, presetTempDest)

	tmpFiles = append(tmpFiles, presetTempDest)

	if contentLength, err := m.downloader.DownloadFile(ctx, presetURL, presetTempDest, nil); err == nil {
		presetDest := filepath.Join(m.cacheDir, m.downloader.getCacheFilename(job.Repo, "preset.ini"))
		if err := os.Rename(presetTempDest, presetDest); err == nil {
			// Remove from cleanup list on success
			for i, f := range tmpFiles {
				if f == presetTempDest {
					tmpFiles = append(tmpFiles[:i], tmpFiles[i+1:]...)
					break
				}
			}
			if contentLength > 0 {
				m.addToTotalBytes(job.ID, contentLength)
			}
		}
	}

	// Download optional mmproj file
	if manifest.MMProjFile != nil {
		// Sanitize filename to prevent path traversal attacks
		safeMMProjFilename := filepath.Base(manifest.MMProjFile.RFilename)

		mmprojURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", job.Repo, manifest.MMProjFile.RFilename)
		mmprojTempDest := m.downloader.getCacheFilename(job.Repo, safeMMProjFilename+".tmp")
		mmprojTempDest = filepath.Join(m.cacheDir, mmprojTempDest)

		tmpFiles = append(tmpFiles, mmprojTempDest)

		progressChan := make(chan int64, progressChannelBuffer)
		go m.trackProgress(job.ID, progressChan)

		m.updateJobCurrentFile(job.ID, safeMMProjFilename)

		contentLength, err := m.downloader.DownloadFile(ctx, mmprojURL, mmprojTempDest, progressChan)
		if err != nil {
			m.failJob(job.ID, fmt.Sprintf("failed to download mmproj file: %v", err))
			return
		}

		close(progressChan)

		if contentLength > 0 {
			m.addToTotalBytes(job.ID, contentLength)
		}

		mmprojDest := filepath.Join(m.cacheDir, m.downloader.getCacheFilename(job.Repo, safeMMProjFilename))
		if err := os.Rename(mmprojTempDest, mmprojDest); err != nil {
			m.failJob(job.ID, fmt.Sprintf("failed to rename mmproj file: %v", err))
			return
		}

		// Remove from cleanup list on success
		for i, f := range tmpFiles {
			if f == mmprojTempDest {
				tmpFiles = append(tmpFiles[:i], tmpFiles[i+1:]...)
				break
			}
		}
	}

	// Rename main GGUF file
	finalDest := filepath.Join(m.cacheDir, m.downloader.getCacheFilename(job.Repo, safeGGUFFilename))
	if err := os.Rename(tempDest, finalDest); err != nil {
		m.failJob(job.ID, fmt.Sprintf("failed to rename GGUF file: %v", err))
		return
	}

	// Remove from cleanup list
	for i, f := range tmpFiles {
		if f == tempDest {
			tmpFiles = append(tmpFiles[:i], tmpFiles[i+1:]...)
			break
		}
	}

	// Rename split files
	if splitCount > 1 {
		for i := 2; i <= splitCount; i++ {
			splitFilename := m.getSplitFilename(safeGGUFFilename, i, splitCount)
			splitTempDest := filepath.Join(m.cacheDir, m.downloader.getCacheFilename(job.Repo, splitFilename+".tmp"))
			splitDest := filepath.Join(m.cacheDir, m.downloader.getCacheFilename(job.Repo, splitFilename))

			if err := os.Rename(splitTempDest, splitDest); err != nil {
				m.failJob(job.ID, fmt.Sprintf("failed to rename split file %d: %v", i, err))
				return
			}

			// Remove from cleanup list
			for j, f := range tmpFiles {
				if f == splitTempDest {
					tmpFiles = append(tmpFiles[:j], tmpFiles[j+1:]...)
					break
				}
			}
		}
	}

	m.completeJob(job.ID)
}

func (m *Manager) getSplitFilename(baseFilename string, part, total int) string {
	ext := filepath.Ext(baseFilename)
	base := strings.TrimSuffix(baseFilename, ext)
	return fmt.Sprintf("%s-%05d-of-%05d%s", base, part, total, ext)
}

func (m *Manager) trackProgress(jobID string, progress <-chan int64) {
	for bytes := range progress {
		m.jobsMutex.Lock()
		if j, ok := m.jobs[jobID]; ok {
			j.Progress.BytesDownloaded += bytes
		}
		m.jobsMutex.Unlock()
	}
}

func (m *Manager) addToTotalBytes(jobID string, bytes int64) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.Progress.TotalBytes += bytes
	}
}

func (m *Manager) updateJobStatus(jobID string, status JobStatus) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.Status = status
	}
}

func (m *Manager) updateJobCurrentFile(jobID string, filename string) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	if job, ok := m.jobs[jobID]; ok {
		job.Progress.CurrentFile = filename
	}
}

func (m *Manager) completeJob(jobID string) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	now := time.Now()
	if job, ok := m.jobs[jobID]; ok {
		job.Status = JobStatusCompleted
		job.CompletedAt = &now
	}
}

func (m *Manager) failJob(jobID string, errMsg string) {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	now := time.Now()
	if job, ok := m.jobs[jobID]; ok {
		job.Status = JobStatusFailed
		job.Error = errMsg
		job.CompletedAt = &now
	}
}

func (m *Manager) GetJob(jobID string) (*Job, error) {
	m.jobsMutex.RLock()
	defer m.jobsMutex.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	jobCopy := *job
	jobCopy.CancelFunc = nil

	return &jobCopy, nil
}

func (m *Manager) CancelJob(jobID string) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	job, exists := m.jobs[jobID]
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

func (m *Manager) ListJobs() []*Job {
	m.jobsMutex.RLock()
	defer m.jobsMutex.RUnlock()

	jobs := make([]*Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobCopy := *job
		jobCopy.CancelFunc = nil
		jobs = append(jobs, &jobCopy)
	}

	return jobs
}

func (m *Manager) DeleteJob(jobID string) error {
	m.jobsMutex.Lock()
	defer m.jobsMutex.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status == JobStatusDownloading || job.Status == JobStatusQueued {
		return fmt.Errorf("cannot delete job with status: %s", job.Status)
	}

	delete(m.jobs, jobID)
	return nil
}

func (m *Manager) generateJobID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
