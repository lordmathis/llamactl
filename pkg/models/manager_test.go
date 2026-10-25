package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testCommit = "abcdef1234567890abcdef1234567890abcdef12"

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type hfFixture struct {
	branches []HFBranch
	tree     []HFTreeEntry
	files    map[string][]byte
	etags    map[string]string
}

// newHFTestServer fakes the refs/tree/resolve endpoints and points HF_ENDPOINT at it.
func newHFTestServer(t *testing.T, fx hfFixture) *httptest.Server {
	t.Helper()

	if fx.branches == nil {
		fx.branches = []HFBranch{{Name: "main", TargetCommit: testCommit}}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/models/") && strings.HasSuffix(r.URL.Path, "/refs"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(HFRefsResponse{Branches: fx.branches})
		case strings.HasPrefix(r.URL.Path, "/api/models/") && strings.Contains(r.URL.Path, "/tree/"):
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(fx.tree)
		case strings.Contains(r.URL.Path, "/resolve/"):
			// {repo}/resolve/{commit}/{path...}
			trimmed := strings.TrimPrefix(r.URL.Path, "/")
			idx := strings.Index(trimmed, "/resolve/")
			filePath := trimmed[idx+len("/resolve/"):]
			if slash := strings.Index(filePath, "/"); slash != -1 {
				filePath = filePath[slash+1:]
			}
			content, ok := fx.files[filePath]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if etag := fx.etags[filePath]; etag != "" {
				w.Header().Set("ETag", etag)
			}
			http.ServeContent(w, r, filepath.Base(filePath), time.Time{}, bytes.NewReader(content))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("HF_ENDPOINT", srv.URL)
	return srv
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	m := NewManager(cache, time.Minute, "test")
	t.Cleanup(m.Close)
	return m
}

func waitForJob(t *testing.T, m *Manager, jobID string) *Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := m.GetJob(jobID)
		if err != nil {
			t.Fatalf("GetJob(%s): %v", jobID, err)
		}
		switch job.Status {
		case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s", jobID)
	return nil
}

func TestStartDownload_InvalidFormat(t *testing.T) {
	m := newTestManager(t)

	for _, format := range []ModelFormat{"bogus", "GGUF"} {
		if _, err := m.StartDownload("org/model", "main", format); err == nil {
			t.Errorf("StartDownload(format=%q) expected error, got none", format)
		}
	}
	if jobs := m.ListJobs(); len(jobs) != 0 {
		t.Errorf("expected 0 jobs after rejected formats, got %d", len(jobs))
	}
}

func TestStartDownload_InvalidRepoName(t *testing.T) {
	m := newTestManager(t)

	for _, repo := range []string{"../etc/passwd", "plainrepo", "org/../model"} {
		if _, err := m.StartDownload(repo, "main", FormatGGUF); err == nil {
			t.Errorf("StartDownload(repo=%q) expected error, got none", repo)
		}
	}
	if jobs := m.ListJobs(); len(jobs) != 0 {
		t.Errorf("expected 0 jobs after rejected repo names, got %d", len(jobs))
	}

	newHFTestServer(t, hfFixture{}) // empty tree: accepted repos fail cleanly downstream
	for _, repo := range []string{"org/model", "a/b/c"} {
		jobID, err := m.StartDownload(repo, "main", FormatGGUF)
		if err != nil {
			t.Errorf("StartDownload(repo=%q) unexpected error: %v", repo, err)
			continue
		}
		job := waitForJob(t, m, jobID)
		if job.Status != JobStatusFailed {
			t.Errorf("repo %q: job status = %s, want failed (empty-tree fixture)", repo, job.Status)
		}
	}
}

func TestDownloadWorker_NoGGUF(t *testing.T) {
	m := newTestManager(t)
	newHFTestServer(t, hfFixture{
		tree: []HFTreeEntry{{Path: "config.json", Type: "file", Size: 11}},
	})

	jobID, err := m.StartDownload("org/model", "main", FormatGGUF)
	if err != nil {
		t.Fatalf("StartDownload: %v", err)
	}
	job := waitForJob(t, m, jobID)
	if job.Status != JobStatusFailed {
		t.Fatalf("job status = %s, want failed", job.Status)
	}
	if !strings.Contains(strings.ToLower(job.Error), "gguf") {
		t.Errorf("job error = %q, want it to mention GGUF", job.Error)
	}
}

func TestFetchRefs(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	d := NewDownloader(cache, time.Second, "test", NewFileManager(cache), nil)

	t.Run("valid commit", func(t *testing.T) {
		newHFTestServer(t, hfFixture{})
		commit, err := d.FetchRefs(context.Background(), "org/model", "main")
		if err != nil {
			t.Fatalf("FetchRefs: %v", err)
		}
		if commit != testCommit {
			t.Errorf("commit = %q, want %q", commit, testCommit)
		}
	})

	t.Run("non-hex commit rejected", func(t *testing.T) {
		newHFTestServer(t, hfFixture{
			branches: []HFBranch{{Name: "main", TargetCommit: "../../evil"}},
		})
		if _, err := d.FetchRefs(context.Background(), "org/model", "main"); err == nil {
			t.Error("expected error for non-hex commit, got none")
		}
	})

	t.Run("non-hex fallback commit rejected", func(t *testing.T) {
		newHFTestServer(t, hfFixture{
			branches: []HFBranch{
				{Name: "dev", TargetCommit: "zzz"},
				{Name: "main", TargetCommit: testCommit},
			},
		})
		if _, err := d.FetchRefs(context.Background(), "org/model", "no-such-branch"); err == nil {
			t.Error("expected error for non-hex fallback commit, got none")
		}
	})
}

func TestDownloadWorker_Safetensors(t *testing.T) {
	m := newTestManager(t)

	configContent := []byte("hello-world") // 11 bytes
	modelContent := bytes.Repeat([]byte("s"), 1000)
	modelOID := sha256hex(modelContent)

	newHFTestServer(t, hfFixture{
		tree: []HFTreeEntry{
			{Path: "config.json", Type: "file", Size: int64(len(configContent))},
			{Path: "model.safetensors", Type: "file", Size: int64(len(modelContent)),
				LFS: &HFLFSInfo{OID: modelOID, Size: int64(len(modelContent))}},
		},
		files: map[string][]byte{
			"config.json":       configContent,
			"model.safetensors": modelContent,
		},
		etags: map[string]string{"config.json": `W/"config-etag-1"`},
	})

	jobID, err := m.StartDownload("org/model", "main", FormatSafetensors)
	if err != nil {
		t.Fatalf("StartDownload: %v", err)
	}
	job := waitForJob(t, m, jobID)
	if job.Status != JobStatusCompleted {
		t.Fatalf("job status = %s (%s)", job.Status, job.Error)
	}

	if want := int64(len(configContent) + len(modelContent)); job.Progress.TotalBytes != want {
		t.Errorf("TotalBytes = %d, want %d (declared sizes, counted once)", job.Progress.TotalBytes, want)
	}

	got, err := os.ReadFile(m.fileManager.HFBlobPath("org/model", modelOID))
	if err != nil {
		t.Fatalf("LFS blob missing: %v", err)
	}
	if !bytes.Equal(got, modelContent) {
		t.Error("LFS blob content mismatch")
	}

	etagBlob := m.fileManager.HFBlobPath("org/model", m.fileManager.DigestHex("config-etag-1"))
	if _, err := os.ReadFile(etagBlob); err != nil {
		t.Errorf("non-LFS blob keyed by etag digest missing: %v", err)
	}

	for _, name := range []string{"config.json", "model.safetensors"} {
		snapPath := m.fileManager.HFSnapshotPath("org/model", testCommit, name)
		if _, err := os.ReadFile(snapPath); err != nil {
			t.Errorf("snapshot entry %s not readable via cache: %v", name, err)
		}
	}

	refData, err := os.ReadFile(m.fileManager.HFRefPath("org/model", "main"))
	if err != nil || string(refData) != testCommit {
		t.Errorf("ref file = %q (%v), want %q", refData, err, testCommit)
	}

	assertBlobDirEntries(t, filepath.Join(m.fileManager.HFRepoDir("org/model"), "blobs"), 2)

	wantModelPath := filepath.Join(m.fileManager.HFRepoDir("org/model"), "snapshots", testCommit)
	if job.ModelPath != wantModelPath {
		t.Errorf("ModelPath = %q, want %q", job.ModelPath, wantModelPath)
	}
}

func TestSweepStaleTempFiles(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	fm := NewFileManager(cache)

	repoDir := fm.HFRepoDir("org/model")
	blobsDir := filepath.Join(repoDir, "blobs")
	refsDir := filepath.Join(repoDir, "refs")
	for _, dir := range []string{blobsDir, refsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	stale := []string{
		filepath.Join(blobsDir, ".model.safetensors.tmp-123"),
		filepath.Join(blobsDir, ".model.safetensors.lnk-ab"),
	}
	for _, p := range stale {
		if err := os.WriteFile(p, []byte("stale"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	keep := []string{
		filepath.Join(blobsDir, "keepblob"),
		filepath.Join(refsDir, "main"),
	}
	if err := os.WriteFile(keep[0], []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep[1], []byte(testCommit), 0644); err != nil {
		t.Fatal(err)
	}

	// regression: marker-named repo files must survive the sweep
	snapshotDir := filepath.Join(repoDir, "snapshots", testCommit)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatal(err)
	}
	blob1 := writeBlobFile(t, repoDir, strings.Repeat("a1", 32), []byte("one"))
	blob2 := writeBlobFile(t, repoDir, strings.Repeat("b2", 32), []byte("two"))
	symlinkSnapshotEntry(t, snapshotDir, "weights.tmp-old.safetensors", blob1)
	symlinkSnapshotEntry(t, snapshotDir, "notes.lnk-v2.gguf", blob2)
	keep = append(keep,
		filepath.Join(snapshotDir, "weights.tmp-old.safetensors"),
		filepath.Join(snapshotDir, "notes.lnk-v2.gguf"),
	)

	m := NewManager(cache, time.Minute, "test")
	t.Cleanup(m.Close)

	for _, p := range stale {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still present after sweep", p)
		}
	}
	for _, p := range keep {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s must survive the sweep: %v", p, err)
		}
	}
}
