package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	f, err := os.Open(filepath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Only read first 1MB (GGUF metadata is at the beginning)
	const maxMetadataSize = 1024 * 1024
	buf := make([]byte, maxMetadataSize)
	n, err := io.ReadAtLeast(f, buf, 1)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}

	strData := string(buf[:n])

	idx := strings.Index(strData, "split.count")
	if idx == -1 {
		return 1, nil
	}

	valueStart := strings.Index(strData[idx:], "=")
	if valueStart == -1 {
		return 1, nil
	}
	valueStart += idx + 1

	valueEnd := strings.IndexAny(strData[valueStart:], " \t\n\r")
	if valueEnd == -1 {
		valueEnd = len(strData)
	} else {
		valueEnd += valueStart
	}

	valueStr := strings.TrimSpace(strData[valueStart:valueEnd])
	count, err := strconv.Atoi(valueStr)
	if err != nil {
		return 1, nil
	}

	return count, nil
}

func (d *Downloader) generateJobID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (d *Downloader) getCacheFilename(repo, filename string) string {
	return strings.ReplaceAll(repo, "/", "_") + "_" + filename
}
