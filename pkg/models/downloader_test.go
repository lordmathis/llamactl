package models

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestDownloader(t *testing.T) (*Downloader, *FileManager, *JobStore) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	fm := NewFileManager(cache)
	js := NewJobStore()
	t.Cleanup(js.Close)
	return NewDownloader(cache, 30*time.Second, "test", fm, NewProgressTracker(js)), fm, js
}

func TestBuildDownloadPlan_InvalidLFSOID(t *testing.T) {
	d, _, _ := newTestDownloader(t)

	entries := []HFTreeEntry{
		{Path: "config.json", Type: "file", Size: 11},
		{Path: "model.gguf", Type: "file", Size: 100, LFS: &HFLFSInfo{OID: "../../etc/passwd", Size: 100}},
	}

	_, err := d.BuildDownloadPlan("org/model", testCommit, entries, "", "", FormatGGUF)
	if err == nil {
		t.Fatal("expected error for malformed LFS OID, got none")
	}
	if !strings.Contains(err.Error(), "model.gguf") {
		t.Errorf("error %q should name the offending file", err)
	}
}

func TestDownloader_ParseSplitCount(t *testing.T) {
	d := NewDownloader("", 0, "", nil, nil)

	tests := []struct {
		name      string
		filepath  string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single file without split pattern",
			filepath:  "/cache/model-Q4_K_M.gguf",
			wantCount: 1,
		},
		{
			name:      "split file 3 parts",
			filepath:  "/cache/model-Q4_K_M-00001-of-00003.gguf",
			wantCount: 3,
		},
		{
			name:      "split file 10 parts",
			filepath:  "/cache/model-Q8_0-00005-of-00010.gguf",
			wantCount: 10,
		},
		{
			name:      "split file 99 parts",
			filepath:  "/cache/model-00050-of-00099.gguf",
			wantCount: 99,
		},
		{
			name:      "just filename without path",
			filepath:  "model-00001-of-00005.gguf",
			wantCount: 5,
		},
		{
			name:      "filename without split pattern",
			filepath:  "model.gguf",
			wantCount: 1,
		},
		{
			name:      "wrong extension",
			filepath:  "model-00001-of-00003.txt",
			wantCount: 1,
		},
		{
			name:      "partial pattern match",
			filepath:  "model-00001.gguf",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := d.ParseSplitCount(tt.filepath)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got != tt.wantCount {
				t.Errorf("ParseSplitCount(%q) = %d, want %d", tt.filepath, got, tt.wantCount)
			}
		})
	}
}

func TestBuildDownloadPlan(t *testing.T) {
	d := NewDownloader("", 0, "", NewFileManager(t.TempDir()), nil)

	entries := []HFTreeEntry{
		{Path: "model-Q4_K_M.gguf", Type: "file", Size: 100, LFS: &HFLFSInfo{OID: strings.Repeat("a1", 32), Size: 100}},
		{Path: "model-Q8_0.gguf", Type: "file", Size: 200, LFS: &HFLFSInfo{OID: strings.Repeat("b2", 32), Size: 200}},
		{Path: "mmproj-F16.gguf", Type: "file", Size: 50, LFS: &HFLFSInfo{OID: strings.Repeat("c3", 32), Size: 50}},
		{Path: "preset.ini", Type: "file", Size: 10},
	}

	plan, err := d.BuildDownloadPlan("org/model", "commit123", entries, "", "Q4_K_M", FormatGGUF)
	if err != nil {
		t.Fatalf("BuildDownloadPlan: %v", err)
	}

	if plan.MainGGUF == nil {
		t.Fatal("expected MainGGUF to be set")
	}
	if plan.MainGGUF.Filename != "model-Q4_K_M.gguf" {
		t.Errorf("MainGGUF.Filename = %q, want %q", plan.MainGGUF.Filename, "model-Q4_K_M.gguf")
	}
	if plan.MMProj == nil {
		t.Error("expected MMProj to be set")
	}
	if plan.Preset == nil {
		t.Error("expected Preset to be set")
	}
	if len(plan.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(plan.Tasks))
	}
}

func TestBuildDownloadPlan_Safetensors(t *testing.T) {
	d := NewDownloader("", 0, "", NewFileManager(t.TempDir()), nil)

	entries := []HFTreeEntry{
		{Path: "config.json", Type: "file", Size: 100},
		{Path: "tokenizer.json", Type: "file", Size: 200},
		{Path: "tokenizer_config.json", Type: "file", Size: 50},
		{Path: "model.safetensors", Type: "file", Size: 1000, LFS: &HFLFSInfo{OID: strings.Repeat("a1", 32), Size: 1000}},
		{Path: "model-00001-of-00002.safetensors", Type: "file", Size: 500, LFS: &HFLFSInfo{OID: strings.Repeat("b2", 32), Size: 500}},
		{Path: "model-00002-of-00002.safetensors", Type: "file", Size: 500, LFS: &HFLFSInfo{OID: strings.Repeat("c3", 32), Size: 500}},
		{Path: "pytorch_model.bin", Type: "file", Size: 2000, LFS: &HFLFSInfo{OID: strings.Repeat("d4", 32), Size: 2000}},
		{Path: "README.md", Type: "file", Size: 10},
		{Path: ".gitattributes", Type: "file", Size: 5},
	}

	plan, err := d.BuildDownloadPlan("org/model", "commit123", entries, "", "", FormatSafetensors)
	if err != nil {
		t.Fatalf("BuildDownloadPlan: %v", err)
	}

	if plan.MainGGUF != nil {
		t.Error("expected MainGGUF to be nil for safetensors plan")
	}
	if plan.Format != FormatSafetensors {
		t.Errorf("Format = %q, want %q", plan.Format, FormatSafetensors)
	}

	filenames := make(map[string]bool)
	for _, task := range plan.Tasks {
		filenames[task.Filename] = true
	}

	for _, name := range []string{"config.json", "tokenizer.json", "tokenizer_config.json", "model.safetensors", "model-00001-of-00002.safetensors", "model-00002-of-00002.safetensors"} {
		if !filenames[name] {
			t.Errorf("expected file %q in plan tasks", name)
		}
	}

	if filenames["pytorch_model.bin"] {
		t.Error("expected pytorch_model.bin to be excluded when safetensors are present")
	}
	if filenames[".gitattributes"] {
		t.Error("expected .gitattributes to be excluded")
	}
	if filenames["README.md"] {
		t.Error("expected README.md to be excluded (not a mandatory or weight file)")
	}
}

func TestBuildDownloadPlan_SafetensorsFallback(t *testing.T) {
	d := NewDownloader("", 0, "", NewFileManager(t.TempDir()), nil)

	entries := []HFTreeEntry{
		{Path: "config.json", Type: "file", Size: 100},
		{Path: "pytorch_model.bin", Type: "file", Size: 2000, LFS: &HFLFSInfo{OID: strings.Repeat("d4", 32), Size: 2000}},
		{Path: "model.pth", Type: "file", Size: 1500, LFS: &HFLFSInfo{OID: strings.Repeat("e5", 32), Size: 1500}},
	}

	plan, err := d.BuildDownloadPlan("org/model", "commit123", entries, "", "", FormatSafetensors)
	if err != nil {
		t.Fatalf("BuildDownloadPlan: %v", err)
	}

	filenames := make(map[string]bool)
	for _, task := range plan.Tasks {
		filenames[task.Filename] = true
	}

	if !filenames["config.json"] {
		t.Error("expected config.json in plan")
	}
	if !filenames["pytorch_model.bin"] {
		t.Error("expected pytorch_model.bin as fallback when no safetensors found")
	}
	if filenames["model.pth"] {
		t.Error("expected model.pth to be excluded (not .bin fallback)")
	}
}

func TestResolveNonLFSOids(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `W/"..\/evil"`)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	d, fm, _ := newTestDownloader(t)

	plan := &HFDownloadPlan{Tasks: []HFDownloadTask{{URL: srv.URL, Filename: "config.json"}}}
	if err := d.ResolveNonLFSOids(context.Background(), plan, "org/model"); err != nil {
		t.Fatalf("ResolveNonLFSOids: %v", err)
	}

	task := &plan.Tasks[0]
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(task.OID) {
		t.Errorf("OID = %q, want 64-char hex digest", task.OID)
	}
	if want := fm.DigestHex(`..\/evil`); task.OID != want {
		t.Errorf("OID = %q, want digest of normalized etag %q", task.OID, want)
	}
	if got := filepath.Base(filepath.Dir(task.BlobPath)); got != "blobs" {
		t.Errorf("BlobPath parent = %q, want \"blobs\"", got)
	}
}

func TestClassifyFileType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"model-Q4_K_M.gguf", "gguf"},
		{"mmproj-F16.gguf", "mmproj"},
		{"model.gguf", "gguf"},
		{"preset.ini", "preset"},
		{"README.md", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := classifyFileType(tt.path)
			if result != tt.expected {
				t.Errorf("classifyFileType(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestBlobValid(t *testing.T) {
	d, fm, _ := newTestDownloader(t)
	blobDir := filepath.Join(fm.HFRepoDir("org/model"), "blobs")
	if err := os.MkdirAll(blobDir, 0755); err != nil {
		t.Fatal(err)
	}

	blobPath := filepath.Join(blobDir, strings.Repeat("ab", 32))
	if err := os.WriteFile(blobPath, bytes.Repeat([]byte("x"), 100), 0644); err != nil {
		t.Fatal(err)
	}
	emptyPath := filepath.Join(blobDir, "empty")
	if err := os.WriteFile(emptyPath, nil, 0644); err != nil {
		t.Fatal(err)
	}
	dirPath := filepath.Join(blobDir, "adirectory")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		task *HFDownloadTask
		want bool
	}{
		{"right size", &HFDownloadTask{BlobPath: blobPath, Size: 100}, true},
		{"unknown size nonempty", &HFDownloadTask{BlobPath: blobPath}, true},
		{"wrong size", &HFDownloadTask{BlobPath: blobPath, Size: 200}, false},
		{"empty file", &HFDownloadTask{BlobPath: emptyPath, Size: 0}, false},
		{"directory", &HFDownloadTask{BlobPath: dirPath}, false},
		{"missing", &HFDownloadTask{BlobPath: filepath.Join(blobDir, "nope")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := d.BlobValid(tt.task); got != tt.want {
				t.Errorf("BlobValid(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func newDownloadTask(fm *FileManager, oid string, content []byte, lfs bool) *HFDownloadTask {
	return &HFDownloadTask{
		Filename: "model.safetensors",
		BlobPath: fm.HFBlobPath("org/model", oid),
		OID:      oid,
		Size:     int64(len(content)),
		LFS:      lfs,
	}
}

func serveContent(t *testing.T, content []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.Write(content)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func assertBlobDirEntries(t *testing.T, dir string, want int) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != want {
		for _, e := range entries {
			t.Logf("entry: %s", e.Name())
		}
		t.Errorf("blob dir has %d entries, want %d", len(entries), want)
	}
}

// waitJobBytes polls until the job's byte counter settles on want; the
// progress tracker applies channel bytes asynchronously.
func waitJobBytes(t *testing.T, js *JobStore, jobID string, want int64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := js.Get(jobID)
		if err != nil {
			t.Fatalf("GetJob: %v", err)
		}
		if job.Progress.BytesDownloaded == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	job, _ := js.Get(jobID)
	t.Fatalf("BytesDownloaded = %d, want %d", job.Progress.BytesDownloaded, want)
}

func TestDownloadBlob_Success(t *testing.T) {
	d, fm, _ := newTestDownloader(t)
	content := bytes.Repeat([]byte("x"), 1000)
	oid := sha256hex(content)
	task := newDownloadTask(fm, oid, content, true)
	task.URL = serveContent(t, content).URL

	if err := d.DownloadBlob(context.Background(), "job", task); err != nil {
		t.Fatalf("DownloadBlob: %v", err)
	}

	got, err := os.ReadFile(task.BlobPath)
	if err != nil {
		t.Fatalf("blob not published: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Error("published blob does not match served bytes")
	}
	assertBlobDirEntries(t, filepath.Dir(task.BlobPath), 1)
}

func TestDownloadBlob_HashMismatch(t *testing.T) {
	d, fm, _ := newTestDownloader(t)
	content := bytes.Repeat([]byte("x"), 1000)
	task := newDownloadTask(fm, sha256hex([]byte("other")), content, true)
	task.URL = serveContent(t, content).URL

	err := d.DownloadBlob(context.Background(), "job", task)
	if err == nil {
		t.Fatal("expected hash mismatch error, got none")
	}
	if !strings.Contains(err.Error(), "hash") {
		t.Errorf("error %q should mention hash", err)
	}
	if _, err := os.Stat(task.BlobPath); !os.IsNotExist(err) {
		t.Error("blob must not be published on hash mismatch")
	}
	assertBlobDirEntries(t, filepath.Dir(task.BlobPath), 0)
}

func TestDownloadBlob_SizeMismatch(t *testing.T) {
	for name, delta := range map[string]int{"over-delivery": 5, "under-delivery": -5} {
		t.Run(name, func(t *testing.T) {
			d, fm, _ := newTestDownloader(t)
			task := newDownloadTask(fm, strings.Repeat("cd", 32), bytes.Repeat([]byte("y"), 100), false)
			served := bytes.Repeat([]byte("y"), 100+delta)
			task.URL = serveContent(t, served).URL

			err := d.DownloadBlob(context.Background(), "job", task)
			if err == nil {
				t.Fatal("expected size mismatch error, got none")
			}
			if !strings.Contains(err.Error(), "size") {
				t.Errorf("error %q should mention size", err)
			}
			if _, err := os.Stat(task.BlobPath); !os.IsNotExist(err) {
				t.Error("blob must not be published on size mismatch")
			}
			assertBlobDirEntries(t, filepath.Dir(task.BlobPath), 0)
		})
	}
}

func TestDownloadBlob_Resume(t *testing.T) {
	d, fm, _ := newTestDownloader(t)
	content := bytes.Repeat([]byte("r"), 1000)
	half := len(content) / 2
	oid := sha256hex(content)
	task := newDownloadTask(fm, oid, content, true)

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n := requests.Add(1); n == 1 {
			w.Header().Set("Content-Length", "1000")
			w.Write(content[:half])
			http.NewResponseController(w).Flush()
			panic(http.ErrAbortHandler) // drop the connection mid-body
		}
		if got := r.Header.Get("Range"); got != fmt.Sprintf("bytes=%d-", half) {
			t.Errorf("Range header = %q, want bytes=%d-", got, half)
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", half, len(content)-1, len(content)))
		w.Header().Set("Content-Length", strconv.Itoa(len(content)-half))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(content[half:])
	}))
	t.Cleanup(srv.Close)
	task.URL = srv.URL

	if err := d.DownloadBlob(context.Background(), "job", task); err != nil {
		t.Fatalf("DownloadBlob: %v", err)
	}

	got, err := os.ReadFile(task.BlobPath)
	if err != nil {
		t.Fatalf("blob not published: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("resumed blob = %d bytes, want full %d-byte content", len(got), len(content))
	}
	assertBlobDirEntries(t, filepath.Dir(task.BlobPath), 1)
}

func TestDownloadBlob_RestartOn200(t *testing.T) {
	d, fm, js := newTestDownloader(t)
	content := bytes.Repeat([]byte("s"), 1000)
	half := len(content) / 2
	oid := sha256hex(content)
	task := newDownloadTask(fm, oid, content, true)

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n := requests.Add(1); n == 1 {
			w.Header().Set("Content-Length", "1000")
			w.Write(content[:half])
			http.NewResponseController(w).Flush()
			panic(http.ErrAbortHandler)
		}
		// Ignores any Range header and always returns the full body.
		w.Header().Set("Content-Length", "1000")
		w.Write(content)
	}))
	t.Cleanup(srv.Close)
	task.URL = srv.URL

	// A real job makes the progress channel observable: the discarded
	// half-copy must be subtracted again, or BytesDownloaded reads 1500.
	job, err := js.Create("org/model", "main")
	if err != nil {
		t.Fatal(err)
	}

	if err := d.DownloadBlob(context.Background(), job.ID, task); err != nil {
		t.Fatalf("DownloadBlob: %v", err)
	}
	if n := requests.Load(); n != 3 {
		t.Errorf("server saw %d requests, want 3 (fail, restart, full)", n)
	}
	waitJobBytes(t, js, job.ID, int64(len(content)))

	got, err := os.ReadFile(task.BlobPath)
	if err != nil {
		t.Fatalf("blob not published: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("blob = %d bytes, want exactly one %d-byte copy", len(got), len(content))
	}
	assertBlobDirEntries(t, filepath.Dir(task.BlobPath), 1)
}

func TestWriteRefFile(t *testing.T) {
	d, fm, _ := newTestDownloader(t)

	if err := d.WriteRefFile("org/model", "main", testCommit); err != nil {
		t.Fatalf("WriteRefFile: %v", err)
	}

	refPath := fm.HFRefPath("org/model", "main")
	got, err := os.ReadFile(refPath)
	if err != nil {
		t.Fatalf("ref file missing: %v", err)
	}
	if string(got) != testCommit {
		t.Errorf("ref file = %q, want %q", got, testCommit)
	}
	assertBlobDirEntries(t, filepath.Dir(refPath), 1) // exactly one entry: no temp leftovers
}

func TestWriteRefFile_InvalidCommit(t *testing.T) {
	d, fm, _ := newTestDownloader(t)

	if err := d.WriteRefFile("org/model", "main", "../evil"); err == nil {
		t.Error("expected error for invalid commit, got none")
	}
	if _, err := os.Stat(fm.HFRefPath("org/model", "main")); !os.IsNotExist(err) {
		t.Error("ref file must not be created for an invalid commit")
	}
}

// A naive url.PathEscape(repo) would escape the "/" separators to %2F and
// break every HF URL; escaping must happen per segment.
func TestEscapeHFPath(t *testing.T) {
	got := escapeHFPath("org/mo del/file name.gguf")
	if want := "org/mo%20del/file%20name.gguf"; got != want {
		t.Errorf("escapeHFPath = %q, want %q", got, want)
	}
}
