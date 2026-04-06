package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	progressChannelBuffer  = 100
	maxConcurrentDownloads = 5
	jobRetentionDuration   = 24 * time.Hour
	jobCleanupInterval     = 1 * time.Hour
)

const (
	hfAPIBase       = "https://huggingface.co"
	hfRefsURLFmt    = "%s/api/models/%s/refs"
	hfTreeURLFmt    = "%s/api/models/%s/tree/%s?recursive=true"
	hfResolveURLFmt = "%s/%s/resolve/%s/%s"
)

type HFBranch struct {
	Name         string `json:"name"`
	TargetCommit string `json:"targetCommit"`
}

type HFRefsResponse struct {
	Branches []HFBranch `json:"branches"`
	Tags     []HFBranch `json:"tags"`
}

type HFLFSInfo struct {
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

type HFTreeEntry struct {
	Path string     `json:"path"`
	Type string     `json:"type"`
	Size int64      `json:"size"`
	LFS  *HFLFSInfo `json:"lfs,omitempty"`
}

type HFRepoInfo struct {
	Repo   string        `json:"repo"`
	Commit string        `json:"commit"`
	Branch string        `json:"branch"`
	Files  []HFTreeEntry `json:"files"`
}

type HFDownloadTask struct {
	URL          string `json:"url"`
	BlobPath     string `json:"blob_path"`
	SnapshotPath string `json:"snapshot_path"`
	OID          string `json:"oid"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	Type         string `json:"type"`
}

type HFDownloadPlan struct {
	Repo     HFRepoInfo       `json:"repo"`
	Tasks    []HFDownloadTask `json:"tasks"`
	MainGGUF *HFDownloadTask  `json:"main_gguf"`
	MMProj   *HFDownloadTask  `json:"mmproj"`
	Preset   *HFDownloadTask  `json:"preset"`
}

type Downloader struct {
	httpClient      *http.Client
	cacheDir        string
	version         string
	fileManager     *FileManager
	progressTracker *ProgressTracker
}

func NewDownloader(cacheDir string, timeout time.Duration, version string, fileManager *FileManager, progressTracker *ProgressTracker) *Downloader {
	if timeout == 0 {
		timeout = 60 * time.Minute
	}
	if version == "" {
		version = "0.1.0"
	}
	return &Downloader{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:          10,
				IdleConnTimeout:       30 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				DisableCompression:    true,
			},
		},
		cacheDir:        cacheDir,
		version:         version,
		fileManager:     fileManager,
		progressTracker: progressTracker,
	}
}

func (md *Downloader) FetchRefs(ctx context.Context, repo, branch string) (string, error) {
	if branch == "" {
		branch = "main"
	}

	baseURL := md.hfBaseURL()
	url := fmt.Sprintf(hfRefsURLFmt, baseURL, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	md.setCommonHeaders(req)

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch refs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("model not found: %s", repo)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to fetch refs: HTTP %d", resp.StatusCode)
	}

	var refsResp HFRefsResponse
	if err := json.NewDecoder(resp.Body).Decode(&refsResp); err != nil {
		return "", fmt.Errorf("failed to decode refs response: %w", err)
	}

	// Find the requested branch by name, fall back to first branch
	for _, b := range refsResp.Branches {
		if b.Name == branch {
			return b.TargetCommit, nil
		}
	}
	if len(refsResp.Branches) > 0 {
		return refsResp.Branches[0].TargetCommit, nil
	}

	return "", fmt.Errorf("could not resolve commit for branch %s", branch)
}

func (md *Downloader) FetchRepoTree(ctx context.Context, repo, commit string) ([]HFTreeEntry, error) {
	url := fmt.Sprintf(hfTreeURLFmt, md.hfBaseURL(), repo, commit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	md.setCommonHeaders(req)

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("model not found: %s", repo)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch repo tree: HTTP %d", resp.StatusCode)
	}

	var entries []HFTreeEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode repo tree: %w", err)
	}
	return entries, nil
}

func (md *Downloader) WriteRefFile(repo, branch, commit string) error {
	refPath := md.fileManager.HFRefPath(repo, branch)

	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpPath := refPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(commit), 0644); err != nil {
		return fmt.Errorf("failed to write ref file: %w", err)
	}

	if err := os.Rename(tmpPath, refPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename ref file: %w", err)
	}
	return nil
}

func (md *Downloader) getHFToken() string {
	token := os.Getenv("HF_TOKEN")
	if len(token) >= 10 && strings.HasPrefix(token, "hf_") {
		return token
	}
	return ""
}

func (md *Downloader) hfBaseURL() string {
	if v := os.Getenv("HF_ENDPOINT"); v != "" {
		return v
	}
	return hfAPIBase
}

func (md *Downloader) setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "llama-cpp/llamactl/"+md.version)
	req.Header.Set("Accept", "application/json")
	if token := md.getHFToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func (md *Downloader) BlobExists(blobPath string) bool {
	info, err := os.Stat(blobPath)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

func (md *Downloader) DownloadBlob(ctx context.Context, jobID string, task *HFDownloadTask) error {
	if md.BlobExists(task.BlobPath) {
		log.Printf("Blob %s already exists, skipping download", task.OID)
		return nil
	}

	progressChan := make(chan int64, progressChannelBuffer)
	go md.progressTracker.Track(jobID, progressChan)
	defer close(progressChan)

	md.progressTracker.UpdateCurrentFile(jobID, task.Filename)

	tmpPath := task.BlobPath + ".tmp"

	existingSize := int64(0)
	if stat, err := os.Stat(tmpPath); err == nil {
		existingSize = stat.Size()
	}
	if existingSize > 0 {
		log.Printf("Resuming blob download from byte %d: %s", existingSize, task.Filename)
	}

	if err := md.downloadBlobFile(ctx, jobID, task.URL, tmpPath, existingSize, task.Size, progressChan); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to download blob %s: %w", task.Filename, err)
	}

	if err := os.Rename(tmpPath, task.BlobPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename blob file: %w", err)
	}
	return nil
}

func (md *Downloader) downloadBlobFile(ctx context.Context, jobID, url, dest string, resumeFrom, totalSize int64, progress chan<- int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "llama-cpp/llamactl/"+md.version)
	if token := md.getHFToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if resumeFrom > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
	}

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && resumeFrom > 0 {
		os.Remove(dest)
		return md.downloadBlobFile(ctx, jobID, url, dest, 0, totalSize, progress)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	if contentLength := resp.ContentLength; contentLength > 0 {
		md.progressTracker.AddToTotalBytes(jobID, contentLength)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	if resumeFrom > 0 {
		if _, err := f.Seek(resumeFrom, 0); err != nil {
			f.Close()
			return fmt.Errorf("failed to seek: %w", err)
		}
	}

	_, copyErr := io.Copy(f, &progressReader{reader: resp.Body, progress: progress})
	f.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to write file: %w", copyErr)
	}
	return nil
}

func (md *Downloader) BuildDownloadPlan(repo, commit string, entries []HFTreeEntry, hfFile, tag string) *HFDownloadPlan {
	baseURL := md.hfBaseURL()

	plan := &HFDownloadPlan{
		Repo: HFRepoInfo{
			Repo:   repo,
			Commit: commit,
			Branch: tag,
			Files:  entries,
		},
	}

	// Partition files
	var allGGUFs []HFTreeEntry
	var mmprojEntry *HFTreeEntry
	var presetEntry *HFTreeEntry

	for i := range entries {
		entry := &entries[i]
		if entry.Type != "file" {
			continue
		}
		lowerPath := strings.ToLower(entry.Path)
		if strings.HasSuffix(lowerPath, ".gguf") {
			if strings.Contains(lowerPath, "mmproj") {
				mmprojEntry = entry
			} else {
				allGGUFs = append(allGGUFs, *entry)
			}
		} else if entry.Path == "preset.ini" {
			presetEntry = entry
		}
	}

	// Select the GGUF(s) to download
	selectedGGUFs := selectGGUFs(allGGUFs, hfFile, tag)

	// Build task list
	var allTasks []HFDownloadTask
	var mainGGUF *HFDownloadTask

	for _, entry := range selectedGGUFs {
		task := md.createDownloadTask(repo, commit, &entry, baseURL)
		allTasks = append(allTasks, task)
		if mainGGUF == nil {
			mainGGUF = &allTasks[len(allTasks)-1]
		}
	}

	if mmprojEntry != nil {
		task := md.createDownloadTask(repo, commit, mmprojEntry, baseURL)
		allTasks = append(allTasks, task)
		plan.MMProj = &allTasks[len(allTasks)-1]
	}

	if presetEntry != nil {
		task := md.createDownloadTask(repo, commit, presetEntry, baseURL)
		allTasks = append(allTasks, task)
		plan.Preset = &allTasks[len(allTasks)-1]
	}

	plan.Tasks = allTasks
	plan.MainGGUF = mainGGUF
	return plan
}

// selectGGUFs picks which GGUF files to include based on explicit file name, tag, or fallback heuristics.
func selectGGUFs(all []HFTreeEntry, hfFile, tag string) []HFTreeEntry {
	if hfFile != "" {
		for _, e := range all {
			if e.Path == hfFile {
				return []HFTreeEntry{e}
			}
		}
		return nil
	}

	if tag != "" {
		tagLower := strings.ToLower(tag)
		var matched []HFTreeEntry
		for _, e := range all {
			if strings.Contains(strings.ToLower(e.Path), tagLower) {
				matched = append(matched, e)
			}
		}
		if len(matched) > 0 {
			return matched
		}
	}

	// Fallback: Q4_K_M, then Q4_0, then first non-shard
	for _, quant := range []string{"q4_k_m", "q4_0"} {
		for _, e := range all {
			if strings.Contains(strings.ToLower(e.Path), quant) {
				return []HFTreeEntry{e}
			}
		}
	}
	for _, e := range all {
		if !isSplitFile(e.Path) {
			return []HFTreeEntry{e}
		}
	}
	return all
}

func (md *Downloader) createDownloadTask(repo, commit string, entry *HFTreeEntry, baseURL string) HFDownloadTask {
	oid := ""
	if entry.LFS != nil {
		oid = entry.LFS.OID
	}
	return HFDownloadTask{
		URL:          fmt.Sprintf(hfResolveURLFmt, baseURL, repo, commit, entry.Path),
		BlobPath:     md.fileManager.HFBlobPath(repo, oid),
		SnapshotPath: md.fileManager.HFSnapshotPath(repo, commit, entry.Path),
		OID:          oid,
		Filename:     entry.Path,
		Size:         entry.Size,
		Type:         classifyFileType(entry.Path),
	}
}

func isSplitFile(filename string) bool {
	matched, _ := regexp.MatchString(`-\d{5}-of-\d{5}\.gguf$`, filename)
	return matched
}

func classifyFileType(path string) string {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".gguf") {
		if strings.Contains(lower, "mmproj") {
			return "mmproj"
		}
		return "gguf"
	}
	if lower == "preset.ini" {
		return "preset"
	}
	return "other"
}

func (md *Downloader) ParseSplitCount(fp string) (int, error) {
	filename := fp
	if idx := strings.LastIndex(fp, string(os.PathSeparator)); idx != -1 {
		filename = fp[idx+1:]
	}

	re := regexp.MustCompile(`-\d{5}-of-(\d{5})\.gguf$`)
	if matches := re.FindStringSubmatch(filename); len(matches) == 2 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count, nil
		}
	}
	return 1, nil
}

type progressReader struct {
	reader   io.Reader
	progress chan<- int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 && pr.progress != nil {
		pr.progress <- int64(n)
	}
	return n, err
}
