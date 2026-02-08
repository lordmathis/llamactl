package models

import (
	"testing"
)

func TestFileManager_GetManifestPath(t *testing.T) {
	fm := NewFileManager("/cache")

	tests := []struct {
		name     string
		repo     string
		tag      string
		expected string
	}{
		{
			name:     "simple repo",
			repo:     "bartowski/Llama-3.2-3B-GGUF",
			tag:      "Q4_K_M",
			expected: "/cache/manifest=bartowski=Llama-3.2-3B-GGUF=Q4_K_M.json",
		},
		{
			name:     "nested repo with multiple slashes",
			repo:     "bartowski/Qwen/Qwen3-30B-GGUF",
			tag:      "Q6_K_L",
			expected: "/cache/manifest=bartowski=Qwen=Qwen3-30B-GGUF=Q6_K_L.json",
		},
		{
			name:     "latest tag",
			repo:     "unsloth/gemma-3-27b-it-GGUF",
			tag:      "latest",
			expected: "/cache/manifest=unsloth=gemma-3-27b-it-GGUF=latest.json",
		},
		{
			name:     "deeply nested repo",
			repo:     "org/team/project/model/GGUF",
			tag:      "Q8_0",
			expected: "/cache/manifest=org=team=project=model=GGUF=Q8_0.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.GetManifestPath(tt.repo, tt.tag)
			if result != tt.expected {
				t.Errorf("GetManifestPath(%q, %q) = %q, want %q",
					tt.repo, tt.tag, result, tt.expected)
			}
		})
	}
}

func TestFileManager_GetPath(t *testing.T) {
	fm := NewFileManager("/cache")

	tests := []struct {
		name     string
		repo     string
		filename string
		expected string
	}{
		{
			name:     "simple repo",
			repo:     "bartowski/Llama-3.2-3B-GGUF",
			filename: "model-Q4_K_M.gguf",
			expected: "/cache/bartowski_Llama-3.2-3B-GGUF_model-Q4_K_M.gguf",
		},
		{
			name:     "nested repo with multiple slashes",
			repo:     "bartowski/Qwen/Qwen3-30B-GGUF",
			filename: "model.gguf",
			expected: "/cache/bartowski_Qwen_Qwen3-30B-GGUF_model.gguf",
		},
		{
			name:     "mmproj file",
			repo:     "org/model",
			filename: "mmproj-F16.gguf",
			expected: "/cache/org_model_mmproj-F16.gguf",
		},
		{
			name:     "preset file",
			repo:     "org/model",
			filename: "preset.ini",
			expected: "/cache/org_model_preset.ini",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.GetPath(tt.repo, tt.filename)
			if result != tt.expected {
				t.Errorf("GetPath(%q, %q) = %q, want %q",
					tt.repo, tt.filename, result, tt.expected)
			}
		})
	}
}

func TestFileManager_GetSplitFilename(t *testing.T) {
	fm := NewFileManager("/cache")

	tests := []struct {
		name         string
		baseFilename string
		part         int
		total        int
		expected     string
	}{
		{
			name:         "split 2 of 3",
			baseFilename: "model-Q4_K_M.gguf",
			part:         2,
			total:        3,
			expected:     "model-Q4_K_M-00002-of-00003.gguf",
		},
		{
			name:         "split 5 of 10",
			baseFilename: "model-Q8_0.gguf",
			part:         5,
			total:        10,
			expected:     "model-Q8_0-00005-of-00010.gguf",
		},
		{
			name:         "split 99 of 100",
			baseFilename: "large-model.gguf",
			part:         99,
			total:        100,
			expected:     "large-model-00099-of-00100.gguf",
		},
		{
			name:         "already split filename",
			baseFilename: "model-00001-of-00003.gguf",
			part:         2,
			total:        3,
			expected:     "model-00001-of-00003-00002-of-00003.gguf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.GetSplitFilename(tt.baseFilename, tt.part, tt.total)
			if result != tt.expected {
				t.Errorf("GetSplitFilename(%q, %d, %d) = %q, want %q",
					tt.baseFilename, tt.part, tt.total, result, tt.expected)
			}
		})
	}
}

func TestFileManager_GetETagPath(t *testing.T) {
	fm := NewFileManager("/cache")

	tests := []struct {
		name     string
		repo     string
		filename string
		expected string
	}{
		{
			name:     "gguf file etag",
			repo:     "org/model",
			filename: "model-Q4_K_M.gguf",
			expected: "/cache/org_model_model-Q4_K_M.gguf.etag",
		},
		{
			name:     "mmproj file etag",
			repo:     "org/model",
			filename: "mmproj-F16.gguf",
			expected: "/cache/org_model_mmproj-F16.gguf.etag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.GetETagPath(tt.repo, tt.filename)
			if result != tt.expected {
				t.Errorf("GetETagPath(%q, %q) = %q, want %q",
					tt.repo, tt.filename, result, tt.expected)
			}
		})
	}
}

// TestFileManager_PathTraversalPrevention verifies that path traversal attacks are blocked
func TestFileManager_PathTraversalPrevention(t *testing.T) {
	fm := NewFileManager("/cache")

	t.Run("GetPath prevents traversal", func(t *testing.T) {
		maliciousInputs := []struct {
			repo     string
			filename string
		}{
			{"org/../etc", "passwd"},
			{"org/model", "../../../etc/passwd"},
			{"/etc/passwd", "model.gguf"},
			{"org\\..\\etc", "model.gguf"},
		}

		for _, input := range maliciousInputs {
			result := fm.GetPath(input.repo, input.filename)
			// Verify no ".." sequences remain
			if containsPathTraversal(result) {
				t.Errorf("GetPath(%q, %q) = %q contains path traversal", input.repo, input.filename, result)
			}
		}
	})

	t.Run("GetManifestPath prevents traversal", func(t *testing.T) {
		maliciousInputs := []struct {
			repo string
			tag  string
		}{
			{"org/../etc", "latest"},
			{"org/model", "../../../etc"},
			{"org\\..\\etc", "latest"},
		}

		for _, input := range maliciousInputs {
			result := fm.GetManifestPath(input.repo, input.tag)
			// Verify no ".." sequences remain
			if containsPathTraversal(result) {
				t.Errorf("GetManifestPath(%q, %q) = %q contains path traversal", input.repo, input.tag, result)
			}
		}
	})
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

