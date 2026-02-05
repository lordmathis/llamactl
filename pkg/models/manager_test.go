package models

import (
	"testing"
)

func TestNewManagerDefaultCacheDir(t *testing.T) {
	mgr := NewManager("", 0, "")
	if mgr.cacheDir == "" {
		t.Error("cacheDir should be set to default value")
	}
}

func TestExtractTag(t *testing.T) {
	tests := []struct {
		filename string
		wantTag  string
	}{
		{"Llama-3.2-3B-Instruct-GGUF_Q4_K_M.gguf", "Q4_K_M"},
		{"model_Q8_0.gguf", "Q8_0"},
		{"preset.ini", "latest"},
	}

	for _, tt := range tests {
		got := extractTag(tt.filename)
		if got != tt.wantTag {
			t.Errorf("extractTag(%q) = %q, want %q", tt.filename, got, tt.wantTag)
		}
	}
}

func TestStartDownloadInvalidRepo(t *testing.T) {
	mgr := NewManager("/tmp/test-cache", 0, "")

	_, err := mgr.StartDownload("", "latest")
	if err == nil {
		t.Error("Expected error for empty repo")
	}

	_, err = mgr.StartDownload("invalid-repo", "latest")
	if err == nil {
		t.Error("Expected error for repo without slash")
	}
}

func TestScanCacheEmpty(t *testing.T) {
	mgr := NewManager("/tmp/nonexistent-cache-path-12345", 0, "")
	models, err := mgr.ListCached()
	if err != nil {
		t.Fatalf("ListCached failed: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("Expected empty model list, got %d", len(models))
	}
}
