package models

import (
	"os"
	"path/filepath"
	"testing"
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
