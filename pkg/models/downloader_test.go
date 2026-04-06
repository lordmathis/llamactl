package models

import (
	"testing"
)

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
		{Path: "model-Q4_K_M.gguf", Type: "file", Size: 100, LFS: &HFLFSInfo{OID: "abc123", Size: 100}},
		{Path: "model-Q8_0.gguf", Type: "file", Size: 200, LFS: &HFLFSInfo{OID: "def456", Size: 200}},
		{Path: "mmproj-F16.gguf", Type: "file", Size: 50, LFS: &HFLFSInfo{OID: "ghi789", Size: 50}},
		{Path: "preset.ini", Type: "file", Size: 10},
	}

	plan := d.BuildDownloadPlan("org/model", "commit123", entries, "", "Q4_K_M")

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
