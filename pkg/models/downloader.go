package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func normalizeETag(etag string) string {
	etag = strings.TrimPrefix(etag, "W/")
	etag = strings.Trim(etag, "\"")
	return etag
}

const (
	progressChannelBuffer  = 100
	maxConcurrentDownloads = 5
	jobRetentionDuration   = 24 * time.Hour
	jobCleanupInterval     = 1 * time.Hour
	maxDownloadAttempts    = 3
	maxMetadataBytes       = 32 << 20
)

type ModelFormat string

const (
	FormatGGUF        ModelFormat = "gguf"
	FormatSafetensors ModelFormat = "safetensors"
)

func IsValidFormat(f ModelFormat) bool {
	return f == FormatGGUF || f == FormatSafetensors
}

const (
	hfAPIBase       = "https://huggingface.co"
	hfRefsURLFmt    = "%s/api/models/%s/refs"
	hfTreeURLFmt    = "%s/api/models/%s/tree/%s?recursive=true"
	hfResolveURLFmt = "%s/%s/resolve/%s/%s"
)

var lfsOIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

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
	LFS          bool   `json:"lfs"`
}

type HFDownloadPlan struct {
	Repo     HFRepoInfo       `json:"repo"`
	Tasks    []HFDownloadTask `json:"tasks"`
	MainGGUF *HFDownloadTask  `json:"main_gguf"`
	MMProj   *HFDownloadTask  `json:"mmproj"`
	Preset   *HFDownloadTask  `json:"preset"`
	Format   ModelFormat      `json:"format"`
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
	url := fmt.Sprintf(hfRefsURLFmt, baseURL, escapeHFPath(repo))

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
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataBytes)).Decode(&refsResp); err != nil {
		return "", fmt.Errorf("failed to decode refs response: %w", err)
	}

	// Find the requested branch by name, fall back to first branch
	for _, b := range refsResp.Branches {
		if b.Name == branch {
			return validatedCommit(b.TargetCommit, b.Name)
		}
	}
	if len(refsResp.Branches) > 0 {
		first := refsResp.Branches[0]
		return validatedCommit(first.TargetCommit, first.Name)
	}

	return "", fmt.Errorf("could not resolve commit for branch %s", branch)
}

func validatedCommit(commit, branch string) (string, error) {
	if !commitPattern.MatchString(commit) {
		return "", fmt.Errorf("invalid commit %q for branch %s", commit, branch)
	}
	return commit, nil
}

func (md *Downloader) FetchRepoTree(ctx context.Context, repo, commit string) ([]HFTreeEntry, error) {
	url := fmt.Sprintf(hfTreeURLFmt, md.hfBaseURL(), escapeHFPath(repo), commit)

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
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataBytes)).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode repo tree: %w", err)
	}
	return entries, nil
}

func (md *Downloader) WriteRefFile(repo, branch, commit string) error {
	if !commitPattern.MatchString(commit) {
		return fmt.Errorf("invalid commit %q for ref %s", commit, branch)
	}
	refPath := md.fileManager.HFRefPath(repo, branch)
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		return fmt.Errorf("failed to create refs directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(refPath), "."+filepath.Base(refPath)+".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temp ref file: %w", err)
	}
	if _, err := tmp.WriteString(commit); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("failed to write ref file: %w", err)
	}
	if err := finishTempFile(tmp, refPath); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("failed to publish ref file: %w", err)
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
		return strings.TrimSuffix(v, "/")
	}
	return hfAPIBase
}

// escapeHFPath percent-escapes each path segment so upstream-controlled repo
// and file names cannot alter the URL structure.
func escapeHFPath(p string) string {
	segments := strings.Split(p, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

func (md *Downloader) setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "llama-cpp/llamactl/"+md.version)
	req.Header.Set("Accept", "application/json")
	if token := md.getHFToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func (md *Downloader) BlobValid(task *HFDownloadTask) bool {
	info, err := os.Stat(task.BlobPath)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	if task.Size > 0 && info.Size() != task.Size {
		return false
	}
	return info.Size() > 0
}

func (md *Downloader) DownloadBlob(ctx context.Context, jobID string, task *HFDownloadTask) error {
	if md.BlobValid(task) {
		log.Printf("Blob %s already valid, skipping download", task.OID)
		return nil
	}
	if err := md.fileManager.HFValidateBlobPath(task.BlobPath); err != nil {
		return fmt.Errorf("refusing to download %s: %w", task.Filename, err)
	}

	progressChan := make(chan int64, progressChannelBuffer)
	go md.progressTracker.Track(jobID, progressChan)
	defer close(progressChan)

	md.progressTracker.UpdateCurrentFile(jobID, task.Filename)

	blobDir := filepath.Dir(task.BlobPath)
	if err := os.MkdirAll(blobDir, 0755); err != nil {
		return fmt.Errorf("failed to create blob directory: %w", err)
	}

	tmp, err := os.CreateTemp(blobDir, "."+filepath.Base(task.BlobPath)+".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer tmp.Close()

	if err := md.downloadWithResume(ctx, jobID, task, tmp, progressChan); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to download blob %s: %w", task.Filename, err)
	}

	if err := finishTempFile(tmp, task.BlobPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to publish blob %s: %w", task.Filename, err)
	}
	return nil
}

func (md *Downloader) downloadWithResume(ctx context.Context, jobID string, task *HFDownloadTask, tmp *os.File, progress chan<- int64) error {
	hasher := sha256.New()

	var lastErr error
	for attempt := 1; attempt <= maxDownloadAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("download cancelled: %w", err)
		}

		offset, err := tmp.Seek(0, io.SeekEnd)
		if err != nil {
			return fmt.Errorf("failed to measure temp file: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.URL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", "llama-cpp/llamactl/"+md.version)
		if token := md.getHFToken(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}

		resp, err := md.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("download cancelled: %w", ctx.Err())
			}
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Printf("[%s] download attempt %d/%d for %s: %v", jobID, attempt, maxDownloadAttempts, task.Filename, lastErr)
			continue
		}

		if offset > 0 && !resumeResponseOK(resp, offset) {
			resp.Body.Close()
			lastErr = fmt.Errorf("server did not honor range request (HTTP %d, Content-Range %q)",
				resp.StatusCode, resp.Header.Get("Content-Range"))
			if err := restartAtZero(tmp, hasher, offset, progress); err != nil {
				return err
			}
			log.Printf("[%s] restarting %s from byte 0: %v", jobID, task.Filename, lastErr)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			err := fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
			resp.Body.Close()
			return err
		}

		if task.Size <= 0 && offset == 0 && resp.ContentLength > 0 {
			md.progressTracker.AddToTotalBytes(jobID, resp.ContentLength)
		}

		var body io.Reader = &progressReader{reader: resp.Body, progress: progress}
		if task.LFS {
			body = io.TeeReader(body, hasher)
		}
		if task.Size > 0 {
			// Allow one byte past the declared size so over-delivery is
			// detected by verifyBlob instead of silently truncated.
			body = io.LimitReader(body, task.Size-offset+1)
		}

		written, copyErr := io.Copy(tmp, body)
		resp.Body.Close()
		if copyErr != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("download cancelled: %w", ctx.Err())
			}
			lastErr = fmt.Errorf("transfer interrupted after %d bytes: %w", written, copyErr)
			log.Printf("[%s] download attempt %d/%d for %s: %v", jobID, attempt, maxDownloadAttempts, task.Filename, lastErr)
			continue
		}

		if err := verifyBlob(task, offset+written, hasher); err != nil {
			return err
		}
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("download did not complete")
	}
	return fmt.Errorf("download failed after %d attempts: %w", maxDownloadAttempts, lastErr)
}

// resumeResponseOK reports whether resp can safely continue from offset.
// A range request must be answered with 206 and a matching Content-Range;
// anything else means the temp prefix cannot be trusted and the download
// must restart from byte 0.
func resumeResponseOK(resp *http.Response, offset int64) bool {
	if resp.StatusCode != http.StatusPartialContent {
		return false
	}
	start, ok := parseContentRangeStart(resp.Header.Get("Content-Range"))
	return ok && start == offset
}

var contentRangeStartPattern = regexp.MustCompile(`^bytes (\d+)-`)

func parseContentRangeStart(header string) (int64, bool) {
	m := contentRangeStartPattern.FindStringSubmatch(header)
	if m == nil {
		return 0, false
	}
	start, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, false
	}
	return start, true
}

func restartAtZero(tmp *os.File, hasher hash.Hash, discarded int64, progress chan<- int64) error {
	if err := tmp.Truncate(0); err != nil {
		return fmt.Errorf("failed to reset temp file: %w", err)
	}
	hasher.Reset()
	if discarded > 0 {
		progress <- -discarded
	}
	return nil
}

func verifyBlob(task *HFDownloadTask, total int64, hasher hash.Hash) error {
	if total == 0 {
		return fmt.Errorf("empty download for %s", task.Filename)
	}
	if task.Size > 0 && total != task.Size {
		return fmt.Errorf("size mismatch for %s: got %d bytes, want %d", task.Filename, total, task.Size)
	}
	if task.LFS {
		got := hex.EncodeToString(hasher.Sum(nil))
		if got != task.OID {
			return fmt.Errorf("hash mismatch for %s: got %s, want %s", task.Filename, got, task.OID)
		}
	}
	return nil
}

// finishTempFile durably publishes a fully written temp file at finalPath.
func finishTempFile(tmp *os.File, finalPath string) error {
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Chmod(tmp.Name(), 0644); err != nil {
		return fmt.Errorf("failed to chmod temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), finalPath); err != nil {
		return fmt.Errorf("failed to rename into place: %w", err)
	}
	return nil
}

func (md *Downloader) BuildDownloadPlan(repo, commit string, entries []HFTreeEntry, hfFile, tag string, format ModelFormat) (*HFDownloadPlan, error) {
	for i := range entries {
		entry := &entries[i]
		if entry.Type != "file" || entry.LFS == nil {
			continue
		}
		if !lfsOIDPattern.MatchString(entry.LFS.OID) {
			return nil, fmt.Errorf("invalid LFS OID for %s: %q", entry.Path, entry.LFS.OID)
		}
	}

	baseURL := md.hfBaseURL()

	plan := &HFDownloadPlan{
		Repo: HFRepoInfo{
			Repo:   repo,
			Commit: commit,
			Branch: tag,
			Files:  entries,
		},
		Format: format,
	}

	if format == FormatSafetensors {
		return md.buildSafetensorsPlan(plan, repo, commit, entries, baseURL), nil
	}

	return md.buildGGUFPlan(plan, repo, commit, entries, hfFile, tag, baseURL), nil
}

func (md *Downloader) FetchFileETag(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HEAD request: %w", err)
	}
	md.setCommonHeaders(req)

	resp, err := md.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch file metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return "", fmt.Errorf("HEAD request failed: HTTP %d", resp.StatusCode)
	}

	etag := resp.Header.Get("ETag")
	if etag == "" {
		return "", fmt.Errorf("no ETag header in response")
	}

	return normalizeETag(etag), nil
}

func (md *Downloader) ResolveNonLFSOids(ctx context.Context, plan *HFDownloadPlan, repo string) error {
	for i := range plan.Tasks {
		task := &plan.Tasks[i]
		if task.OID != "" {
			continue
		}

		etag, err := md.FetchFileETag(ctx, task.URL)
		if err != nil {
			return fmt.Errorf("failed to resolve etag for %s: %w", task.Filename, err)
		}
		task.OID = md.fileManager.DigestHex(etag)
		task.BlobPath = md.fileManager.HFBlobPath(repo, task.OID)
	}
	return nil
}

func (md *Downloader) buildGGUFPlan(plan *HFDownloadPlan, repo, commit string, entries []HFTreeEntry, hfFile, tag, baseURL string) *HFDownloadPlan {
	var allGGUFs []HFTreeEntry
	var mmprojEntry *HFTreeEntry
	var presetEntry *HFTreeEntry
	var jinjaEntries []HFTreeEntry

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
		} else if strings.HasSuffix(lowerPath, ".jinja") {
			jinjaEntries = append(jinjaEntries, *entry)
		}
	}

	selectedGGUFs := selectGGUFs(allGGUFs, hfFile, tag)

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

	for i := range jinjaEntries {
		task := md.createDownloadTask(repo, commit, &jinjaEntries[i], baseURL)
		allTasks = append(allTasks, task)
	}

	plan.Tasks = allTasks
	plan.MainGGUF = mainGGUF
	return plan
}

func (md *Downloader) buildSafetensorsPlan(plan *HFDownloadPlan, repo, commit string, entries []HFTreeEntry, baseURL string) *HFDownloadPlan {
	hasSafetensors := false
	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Path), ".safetensors") {
			hasSafetensors = true
			break
		}
	}

	var selected []HFTreeEntry
	for _, entry := range entries {
		if entry.Type != "file" {
			continue
		}
		lowerPath := strings.ToLower(entry.Path)

		if strings.HasPrefix(entry.Path, ".git/") || entry.Path == ".gitattributes" {
			continue
		}

		if isMandatoryFile(lowerPath) {
			selected = append(selected, entry)
			continue
		}

		if hasSafetensors {
			if strings.HasSuffix(lowerPath, ".safetensors") {
				selected = append(selected, entry)
			}
		} else {
			if strings.HasSuffix(lowerPath, ".bin") {
				selected = append(selected, entry)
			}
		}
	}

	var allTasks []HFDownloadTask
	for _, entry := range selected {
		task := md.createDownloadTask(repo, commit, &entry, baseURL)
		allTasks = append(allTasks, task)
	}

	plan.Tasks = allTasks
	return plan
}

func isMandatoryFile(lowerPath string) bool {
	return strings.HasSuffix(lowerPath, ".json") ||
		strings.HasSuffix(lowerPath, ".txt") ||
		strings.HasSuffix(lowerPath, ".model") ||
		strings.HasSuffix(lowerPath, ".py") ||
		strings.HasSuffix(lowerPath, ".jinja")
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
	lfs := entry.LFS != nil
	oid := ""
	if lfs {
		oid = entry.LFS.OID
	}
	blobPath := ""
	if oid != "" {
		blobPath = md.fileManager.HFBlobPath(repo, oid)
	}
	return HFDownloadTask{
		URL:          fmt.Sprintf(hfResolveURLFmt, baseURL, escapeHFPath(repo), commit, escapeHFPath(entry.Path)),
		BlobPath:     blobPath,
		SnapshotPath: md.fileManager.HFSnapshotPath(repo, commit, entry.Path),
		OID:          oid,
		Filename:     entry.Path,
		Size:         entry.Size,
		Type:         classifyFileType(entry.Path),
		LFS:          lfs,
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
