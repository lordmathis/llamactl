package models

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHFRepoDirName(t *testing.T) {
	fm := NewFileManager("/cache")

	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "simple repo",
			repo:     "bartowski/Llama-3.2-3B-GGUF",
			expected: "models--bartowski--Llama-3.2-3B-GGUF",
		},
		{
			name:     "nested repo",
			repo:     "bartowski/Qwen/Qwen3-30B-GGUF",
			expected: "models--bartowski--Qwen--Qwen3-30B-GGUF",
		},
		{
			name:     "org model",
			repo:     "org/model",
			expected: "models--org--model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.HFRepoDirName(tt.repo)
			if result != tt.expected {
				t.Errorf("HFRepoDirName(%q) = %q, want %q", tt.repo, result, tt.expected)
			}
		})
	}
}

func TestHFValidateBlobPath(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	fm := NewFileManager(cache)

	blobDir := fm.HFRepoDir("org/model") + "/blobs"

	tests := []struct {
		name      string
		path      string
		wantValid bool
	}{
		{"hex blob under repo", blobDir + "/" + strings.Repeat("ab", 32), true},
		{"digest-named blob", blobDir + "/etag-digest-name", true},
		{"traversal out of blobs dir", blobDir + "/../evil", false},
		{"nested under blobs dir", blobDir + "/a/b", false},
		{"blobs dir itself", blobDir, false},
		{"escape of hub root", fm.HFHubRoot() + "/../outside/hub", false},
		{"path under a different root", filepath.Join(t.TempDir(), "models--org--model", "blobs", "x"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fm.HFValidateBlobPath(tt.path)
			if tt.wantValid && err != nil {
				t.Errorf("HFValidateBlobPath(%q) = %v, want nil", tt.path, err)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("HFValidateBlobPath(%q) = nil, want error", tt.path)
			}
		})
	}
}

func TestHFCreateSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	blobPath := filepath.Join(tmpDir, "blobs", "abc123")
	snapshotPath := filepath.Join(tmpDir, "snapshots", "def456", "model.gguf")

	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		t.Fatalf("failed to create blob dir: %v", err)
	}
	if err := os.WriteFile(blobPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create blob file: %v", err)
	}

	if err := fm.HFCreateSymlink(blobPath, snapshotPath); err != nil {
		t.Fatalf("HFCreateSymlink failed: %v", err)
	}

	link, err := os.Readlink(snapshotPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}

	expectedRel, _ := filepath.Rel(filepath.Dir(snapshotPath), blobPath)
	if link != expectedRel {
		t.Errorf("symlink target = %q, want %q", link, expectedRel)
	}
}

func TestHFCreateSymlinkFallback(t *testing.T) {
	tmpDir := t.TempDir()
	fm := NewFileManager(tmpDir)

	blobPath := filepath.Join(tmpDir, "blobs", strings.Repeat("ab", 32))
	snapshotDir := filepath.Join(tmpDir, "snapshots", testCommit)
	snapshotPath := filepath.Join(snapshotDir, "model.gguf")

	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blobPath, []byte("blob-bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := fm.HFCreateSymlinkFallback(blobPath, snapshotPath); err != nil {
		t.Fatalf("HFCreateSymlinkFallback: %v", err)
	}

	got, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("hard-linked snapshot entry missing: %v", err)
	}
	if string(got) != "blob-bytes" {
		t.Errorf("snapshot entry = %q, want %q", got, "blob-bytes")
	}

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "model.gguf" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("snapshot dir entries = %v, want exactly [model.gguf]", names)
	}
}

func TestHFHubRoot(t *testing.T) {
	fm := NewFileManager("/cache")
	t.Setenv("LLAMA_CACHE", "")

	result := fm.HFHubRoot()
	if result == "" {
		t.Error("HFHubRoot returned empty string")
	}
}

func TestHFPathTraversalPrevention(t *testing.T) {
	fm := NewFileManager("/cache")

	malicious := []string{"../etc/passwd", "../../root", "org/../etc"}
	for _, input := range malicious {
		result := fm.HFRepoDirName(input)
		if containsPathTraversal(result) {
			t.Errorf("HFRepoDirName(%q) = %q contains path traversal", input, result)
		}
	}
}

func containsPathTraversal(path string) bool {
	return len(path) >= 2 && (path[0:2] == ".." ||
		len(path) >= 3 && (path[0:3] == "/.." || path[0:3] == "\\.." ||
			contains(path, "/../") || contains(path, "\\..")))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
