package models

import (
	"os"
	"path/filepath"
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
