package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Downloader struct {
	httpClient    *http.Client
	cacheDir      string
	version       string
	manifestCache map[string]*Manifest
	manifestMutex sync.RWMutex
}

type Manifest struct {
	GGUFFile   *FileRef `json:"ggufFile"`
	MMProjFile *FileRef `json:"mmprojFile,omitempty"`
}

type FileRef struct {
	RFilename string `json:"rfilename"`
}

func NewDownloader(cacheDir string, timeout time.Duration, version string) *Downloader {
	if timeout == 0 {
		timeout = 60 * time.Minute
	}
	if version == "" {
		version = "1.0.0"
	}
	return &Downloader{
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		},
		cacheDir:      cacheDir,
		version:       version,
		manifestCache: make(map[string]*Manifest),
	}
}

func (d *Downloader) FetchManifest(ctx context.Context, repo, tag string) (*Manifest, error) {
	if tag == "" {
		tag = "latest"
	}

	// Check cache first
	cacheKey := repo + ":" + tag
	d.manifestMutex.RLock()
	if cached, ok := d.manifestCache[cacheKey]; ok {
		d.manifestMutex.RUnlock()
		return cached, nil
	}
	d.manifestMutex.RUnlock()

	url := fmt.Sprintf("https://huggingface.co/v2/%s/manifests/%s", repo, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "llamactl/"+d.version)
	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("model not found: %s", repo)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	// Cache the manifest
	d.manifestMutex.Lock()
	d.manifestCache[cacheKey] = &manifest
	d.manifestMutex.Unlock()

	return &manifest, nil
}

func (d *Downloader) DownloadFile(ctx context.Context, url, dest string, progress chan<- int64) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "llamactl/"+d.version)
	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return 0, fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	contentLength := resp.ContentLength

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory: %w", err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	reader := &progressReader{
		reader:   resp.Body,
		progress: progress,
	}

	if _, err := io.Copy(f, reader); err != nil {
		return 0, fmt.Errorf("failed to write file: %w", err)
	}

	return contentLength, nil
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

func (d *Downloader) ParseSplitCount(filepath string) (int, error) {
	// Check if the filename follows the split file pattern: name-00001-of-00003.gguf
	// If it does, extract the total count from the filename
	filename := filepath
	if idx := strings.LastIndex(filepath, string(os.PathSeparator)); idx != -1 {
		filename = filepath[idx+1:]
	}

	// Pattern: -XXXXX-of-YYYYY.gguf where YYYYY is the total count
	pattern := `-\d{5}-of-(\d{5})\.gguf$`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(filename)
	if len(matches) == 2 {
		count, err := strconv.Atoi(matches[1])
		if err == nil {
			return count, nil
		}
	}

	// If no split pattern found, assume single file
	return 1, nil
}

func (d *Downloader) getCacheFilename(repo, filename string) string {
	return strings.ReplaceAll(repo, "/", "_") + "_" + filename
}
