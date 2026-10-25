package models

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseHFRepoDirName(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		wantRepo string
	}{
		{
			name:     "simple repo",
			dirName:  "models--bartowski--Llama-3.2-3B-GGUF",
			wantRepo: "bartowski/Llama-3.2-3B-GGUF",
		},
		{
			name:     "nested repo",
			dirName:  "models--bartowski--Qwen--Qwen3-30B-GGUF",
			wantRepo: "bartowski/Qwen/Qwen3-30B-GGUF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := parseHFRepoDirName(tt.dirName)
			if repo != tt.wantRepo {
				t.Errorf("parseHFRepoDirName(%q) = %q, want %q", tt.dirName, repo, tt.wantRepo)
			}
		})
	}
}

func TestScanCache(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("LLAMA_CACHE", tmpDir)

	fm := NewFileManager(tmpDir)
	hubRoot := fm.HFHubRoot()

	repoDir := filepath.Join(hubRoot, "models--bartowski--Llama-3.2-3B-GGUF")
	blobsDir := filepath.Join(repoDir, "blobs")
	snapshotDir := filepath.Join(repoDir, "snapshots", "abc123def456789012345678901234567890abcd")
	refsDir := filepath.Join(repoDir, "refs")

	os.MkdirAll(blobsDir, 0755)
	os.MkdirAll(snapshotDir, 0755)
	os.MkdirAll(refsDir, 0755)

	os.WriteFile(filepath.Join(blobsDir, "blob1"), []byte("fake model data"), 0644)
	os.WriteFile(filepath.Join(blobsDir, "blob2"), []byte("fake mmproj data"), 0644)
	os.WriteFile(filepath.Join(refsDir, "main"), []byte("abc123def456789012345678901234567890abcd"), 0644)

	os.Symlink("../../blobs/blob1", filepath.Join(snapshotDir, "model-Q4_K_M.gguf"))
	os.Symlink("../../blobs/blob2", filepath.Join(snapshotDir, "mmproj-F16.gguf"))

	models, err := ScanCache(tmpDir, "test-node")
	if err != nil {
		t.Fatalf("ScanCache failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	model := models[0]
	if model.Node != "test-node" {
		t.Errorf("node = %q, want %q", model.Node, "test-node")
	}

	if model.Repo != "bartowski/Llama-3.2-3B-GGUF" {
		t.Errorf("repo = %q, want %q", model.Repo, "bartowski/Llama-3.2-3B-GGUF")
	}

	if len(model.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(model.Files))
	}

	hasGGUF := false
	hasMMProj := false
	for _, file := range model.Files {
		if file.Type == "gguf" {
			hasGGUF = true
		}
		if file.Type == "mmproj" {
			hasMMProj = true
		}
	}

	if !hasGGUF {
		t.Error("expected GGUF file in results")
	}
	if !hasMMProj {
		t.Error("expected MMProj file in results")
	}
}

func TestScanCache_HardLinks(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("LLAMA_CACHE", tmpDir)

	fm := NewFileManager(tmpDir)
	hubRoot := fm.HFHubRoot()

	repoDir := filepath.Join(hubRoot, "models--bartowski--Llama-3.2-3B-GGUF")
	snapshotDir := filepath.Join(repoDir, "snapshots", "abc123def")
	refsDir := filepath.Join(repoDir, "refs")

	os.MkdirAll(snapshotDir, 0755)
	os.MkdirAll(refsDir, 0755)

	// Create regular files instead of symlinks
	os.WriteFile(filepath.Join(snapshotDir, "model-Q4_K_M.gguf"), []byte("fake model data"), 0644)
	os.WriteFile(filepath.Join(refsDir, "main"), []byte("abc123def"), 0644)

	models, err := ScanCache(tmpDir, "test-node")
	if err != nil {
		t.Fatalf("ScanCache failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	model := models[0]
	if len(model.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(model.Files))
	}

	if model.Files[0].Name != "model-Q4_K_M.gguf" {
		t.Errorf("file name = %q, want %q", model.Files[0].Name, "model-Q4_K_M.gguf")
	}

	// Verify size is correct even for hard link
	if model.SizeBytes != int64(len("fake model data")) {
		t.Errorf("size = %d, want %d", model.SizeBytes, len("fake model data"))
	}
}

func TestScanCache_NonexistentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("LLAMA_CACHE", tmpDir)

	models, err := ScanCache("/nonexistent/path/12345", "test-node")
	if err != nil {
		t.Errorf("expected no error for nonexistent directory, got %v", err)
	}

	if len(models) != 0 {
		t.Errorf("expected empty slice, got %d models", len(models))
	}
}

func TestClassifyHFModelFileType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"model-Q4_K_M.gguf", "gguf"},
		{"model.gguf", "gguf"},
		{"mmproj-F16.gguf", "mmproj"},
		{"model-mmproj-Q4.gguf", "mmproj"},
		{"preset.ini", "preset"},
		{"README.md", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := classifyHFModelFileType(tt.filename)
			if result != tt.expected {
				t.Errorf("classifyHFModelFileType(%q) = %q, want %q", tt.filename, result, tt.expected)
			}
		})
	}
}

// --- DeleteModel fixtures ---

func newDeleteTestManager(t *testing.T) (*Manager, *FileManager) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	fm := NewFileManager(cache)
	m := NewManager(cache, time.Minute, "test")
	t.Cleanup(m.Close)
	return m, fm
}

func writeRefFile(t *testing.T, repoDir, name, commit string) string {
	t.Helper()
	refPath := filepath.Join(repoDir, "refs", name)
	if err := os.MkdirAll(filepath.Dir(refPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(refPath, []byte(commit), 0644); err != nil {
		t.Fatal(err)
	}
	return refPath
}

func writeBlobFile(t *testing.T, repoDir, name string, content []byte) string {
	t.Helper()
	blobPath := filepath.Join(repoDir, "blobs", name)
	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blobPath, content, 0644); err != nil {
		t.Fatal(err)
	}
	return blobPath
}

func symlinkSnapshotEntry(t *testing.T, snapshotDir, name, target string) {
	t.Helper()
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(snapshotDir, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(rel, filepath.Join(snapshotDir, name)); err != nil {
		t.Fatal(err)
	}
}

func mustNotExist(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("%s still present after delete", what)
	}
}

func mustExist(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("%s missing after delete: %v", what, err)
	}
}

func TestDeleteModel_SharedCommit(t *testing.T) {
	m, fm := newDeleteTestManager(t)
	repoDir := fm.HFRepoDir("org/model")

	blob1 := writeBlobFile(t, repoDir, strings.Repeat("a1", 32), []byte("one"))
	blob2 := writeBlobFile(t, repoDir, strings.Repeat("b2", 32), []byte("two"))
	snapshotDir := filepath.Join(repoDir, "snapshots", testCommit)
	symlinkSnapshotEntry(t, snapshotDir, "model.gguf", blob1)
	symlinkSnapshotEntry(t, snapshotDir, "mmproj.gguf", blob2)
	mainRef := writeRefFile(t, repoDir, "main", testCommit)
	stableRef := writeRefFile(t, repoDir, "stable", testCommit)

	if err := m.DeleteModel("org/model", "main"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}

	mustNotExist(t, mainRef, "refs/main")
	mustExist(t, stableRef, "refs/stable")
	mustExist(t, snapshotDir, "shared snapshot")
	mustExist(t, blob1, "blob1")
	mustExist(t, blob2, "blob2")
}

func TestDeleteModel_LastRef(t *testing.T) {
	m, fm := newDeleteTestManager(t)
	repoDir := fm.HFRepoDir("org/model")

	blob1 := writeBlobFile(t, repoDir, strings.Repeat("a1", 32), []byte("one"))
	blob2 := writeBlobFile(t, repoDir, strings.Repeat("b2", 32), []byte("two"))
	snapshotDir := filepath.Join(repoDir, "snapshots", testCommit)
	symlinkSnapshotEntry(t, snapshotDir, "model.gguf", blob1)
	symlinkSnapshotEntry(t, snapshotDir, "mmproj.gguf", blob2)
	mainRef := writeRefFile(t, repoDir, "main", testCommit)

	if err := m.DeleteModel("org/model", "main"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}

	mustNotExist(t, mainRef, "refs/main")
	mustNotExist(t, snapshotDir, "snapshot dir")
	mustNotExist(t, blob1, "blob1")
	mustNotExist(t, blob2, "blob2")

	blobsDir := filepath.Join(repoDir, "blobs")
	entries, err := os.ReadDir(blobsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("blobs dir has %d entries after delete, want 0", len(entries))
	}
}

func TestDeleteModel_EscapingSymlink(t *testing.T) {
	m, fm := newDeleteTestManager(t)
	repoDir := fm.HFRepoDir("org/model")

	victimPath := filepath.Join(t.TempDir(), "victim.txt")
	victimContent := []byte("do not delete me")
	if err := os.WriteFile(victimPath, victimContent, 0644); err != nil {
		t.Fatal(err)
	}

	snapshotDir := filepath.Join(repoDir, "snapshots", testCommit)
	symlinkSnapshotEntry(t, snapshotDir, "evil.gguf", victimPath)
	mainRef := writeRefFile(t, repoDir, "main", testCommit)

	if err := m.DeleteModel("org/model", "main"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}

	got, err := os.ReadFile(victimPath)
	if err != nil {
		t.Fatalf("victim deleted by escaping symlink: %v", err)
	}
	if !bytes.Equal(got, victimContent) {
		t.Error("victim content modified")
	}
	mustNotExist(t, mainRef, "refs/main")
	mustNotExist(t, snapshotDir, "snapshot dir")
}

// Pins the deliberate fail-open choice: an invalid sibling ref is skipped,
// not an error, and it protects nothing.
func TestDeleteModel_InvalidSiblingRefIgnored(t *testing.T) {
	m, fm := newDeleteTestManager(t)
	repoDir := fm.HFRepoDir("org/model")

	blobPath := writeBlobFile(t, repoDir, strings.Repeat("a1", 32), []byte("one"))
	snapshotDir := filepath.Join(repoDir, "snapshots", testCommit)
	symlinkSnapshotEntry(t, snapshotDir, "model.gguf", blobPath)
	mainRef := writeRefFile(t, repoDir, "main", testCommit)
	writeRefFile(t, repoDir, "garbage", "not-a-commit")

	if err := m.DeleteModel("org/model", "main"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}

	mustNotExist(t, mainRef, "refs/main")
	mustNotExist(t, blobPath, "blob (invalid sibling ref must not protect it)")
}

func TestDeleteModel_DanglingRef(t *testing.T) {
	m, fm := newDeleteTestManager(t)
	repoDir := fm.HFRepoDir("org/model")
	mainRef := writeRefFile(t, repoDir, "main", testCommit)
	// snapshots/<commit> intentionally absent.

	if err := m.DeleteModel("org/model", "main"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	mustNotExist(t, mainRef, "refs/main")
}

func TestDeleteModel_HardlinkFallback(t *testing.T) {
	m, fm := newDeleteTestManager(t)
	repoDir := fm.HFRepoDir("org/model")

	blobPath := writeBlobFile(t, repoDir, strings.Repeat("a1", 32), []byte("one"))
	snapshotDir := filepath.Join(repoDir, "snapshots", testCommit)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(snapshotDir, "model.gguf")
	if err := os.Link(blobPath, entryPath); err != nil {
		t.Fatal(err)
	}
	mainRef := writeRefFile(t, repoDir, "main", testCommit)

	if err := m.DeleteModel("org/model", "main"); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}

	mustNotExist(t, entryPath, "hard-linked snapshot entry")
	mustExist(t, blobPath, "hard-linked blob (kept per D6)")
	mustNotExist(t, mainRef, "refs/main")
}
