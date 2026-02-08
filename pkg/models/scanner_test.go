package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifestFilename(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantRepo    string
		wantTag     string
		shouldError bool
	}{
		{
			name:     "simple repo with single slash",
			filename: "manifest=bartowski=Llama-3.2-3B-Instruct-GGUF=Q4_K_M.json",
			wantRepo: "bartowski/Llama-3.2-3B-Instruct-GGUF",
			wantTag:  "Q4_K_M",
		},
		{
			name:     "nested repo with multiple slashes",
			filename: "manifest=bartowski=Qwen=Qwen3-30B-GGUF=Q6_K_L.json",
			wantRepo: "bartowski/Qwen/Qwen3-30B-GGUF",
			wantTag:  "Q6_K_L",
		},
		{
			name:     "latest tag",
			filename: "manifest=unsloth=gemma-3-27b-it-GGUF=latest.json",
			wantRepo: "unsloth/gemma-3-27b-it-GGUF",
			wantTag:  "latest",
		},
		{
			name:     "deeply nested repo",
			filename: "manifest=org=team=project=model=GGUF=Q8_0.json",
			wantRepo: "org/team/project/model/GGUF",
			wantTag:  "Q8_0",
		},
		{
			name:        "missing prefix",
			filename:    "bartowski=Model=Q4.json",
			shouldError: true,
		},
		{
			name:        "missing suffix",
			filename:    "manifest=bartowski=Model=Q4",
			shouldError: true,
		},
		{
			name:        "only one part",
			filename:    "manifest=something.json",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, tag, err := parseManifestFilename(tt.filename)

			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}

			if tag != tt.wantTag {
				t.Errorf("tag = %q, want %q", tag, tt.wantTag)
			}
		})
	}
}

func TestParseSplitCount(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantCount int
	}{
		{
			name:      "single file (no split pattern)",
			filename:  "model-Q4_K_M.gguf",
			wantCount: 1,
		},
		{
			name:      "split file 3 parts",
			filename:  "model-Q4_K_M-00001-of-00003.gguf",
			wantCount: 3,
		},
		{
			name:      "split file 10 parts",
			filename:  "model-Q8_0-00005-of-00010.gguf",
			wantCount: 10,
		},
		{
			name:      "split file 99 parts",
			filename:  "model-00050-of-00099.gguf",
			wantCount: 99,
		},
		{
			name:      "no extension",
			filename:  "model-Q4_K_M",
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := parseSplitCount(tt.filename)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if count != tt.wantCount {
				t.Errorf("count = %d, want %d", count, tt.wantCount)
			}
		})
	}
}

func TestScanCache(t *testing.T) {
	// Create temp cache directory
	tmpDir := t.TempDir()

	// Create test manifest and files
	manifest := &Manifest{
		GGUFFile: &FileRef{
			RFilename: "model-Q4_K_M.gguf",
		},
		MMProjFile: &FileRef{
			RFilename: "mmproj-F16.gguf",
		},
	}

	manifestJSON, _ := json.Marshal(manifest)
	manifestPath := filepath.Join(tmpDir, "manifest=bartowski=Llama-3.2-3B-GGUF=Q4_K_M.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create model file
	modelPath := filepath.Join(tmpDir, "bartowski_Llama-3.2-3B-GGUF_model-Q4_K_M.gguf")
	if err := os.WriteFile(modelPath, []byte("fake model data"), 0644); err != nil {
		t.Fatalf("failed to write model file: %v", err)
	}

	// Create mmproj file
	mmprojPath := filepath.Join(tmpDir, "bartowski_Llama-3.2-3B-GGUF_mmproj-F16.gguf")
	if err := os.WriteFile(mmprojPath, []byte("fake mmproj data"), 0644); err != nil {
		t.Fatalf("failed to write mmproj file: %v", err)
	}

	// Scan cache
	models, err := ScanCache(tmpDir)
	if err != nil {
		t.Fatalf("ScanCache failed: %v", err)
	}

	// Verify results
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	model := models[0]
	if model.Repo != "bartowski/Llama-3.2-3B-GGUF" {
		t.Errorf("repo = %q, want %q", model.Repo, "bartowski/Llama-3.2-3B-GGUF")
	}

	if model.Tag != "Q4_K_M" {
		t.Errorf("tag = %q, want %q", model.Tag, "Q4_K_M")
	}

	if len(model.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(model.Files))
	}

	// Check file types
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

func TestScanCache_WithSplitFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manifest for split model
	manifest := &Manifest{
		GGUFFile: &FileRef{
			RFilename: "model-Q4_K_M-00001-of-00003.gguf",
		},
	}

	manifestJSON, _ := json.Marshal(manifest)
	manifestPath := filepath.Join(tmpDir, "manifest=org=model-GGUF=Q4_K_M.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Create split files
	fm := NewFileManager(tmpDir)
	for i := 1; i <= 3; i++ {
		var filename string
		if i == 1 {
			filename = "model-Q4_K_M-00001-of-00003.gguf"
		} else {
			filename = fm.GetSplitFilename("model-Q4_K_M-00001-of-00003.gguf", i, 3)
		}
		filePath := fm.GetPath("org/model-GGUF", filename)
		if err := os.WriteFile(filePath, []byte("fake data"), 0644); err != nil {
			t.Fatalf("failed to write split file %d: %v", i, err)
		}
	}

	// Scan cache
	models, err := ScanCache(tmpDir)
	if err != nil {
		t.Fatalf("ScanCache failed: %v", err)
	}

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	model := models[0]
	if len(model.Files) != 3 {
		t.Errorf("expected 3 split files, got %d", len(model.Files))
	}

	// All files should be type "gguf"
	for _, file := range model.Files {
		if file.Type != "gguf" {
			t.Errorf("expected file type 'gguf', got %q", file.Type)
		}
	}
}

func TestScanCache_NonexistentDirectory(t *testing.T) {
	models, err := ScanCache("/nonexistent/path/12345")
	if err != nil {
		t.Errorf("expected no error for nonexistent directory, got %v", err)
	}

	if len(models) != 0 {
		t.Errorf("expected empty slice, got %d models", len(models))
	}
}

func TestScanCache_IgnoresModelsWithoutFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create manifest but no actual model files
	manifest := &Manifest{
		GGUFFile: &FileRef{
			RFilename: "nonexistent-model.gguf",
		},
	}

	manifestJSON, _ := json.Marshal(manifest)
	manifestPath := filepath.Join(tmpDir, "manifest=org=model=latest.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Scan cache
	models, err := ScanCache(tmpDir)
	if err != nil {
		t.Fatalf("ScanCache failed: %v", err)
	}

	// Should return empty because no files exist
	if len(models) != 0 {
		t.Errorf("expected 0 models (files don't exist), got %d", len(models))
	}
}
