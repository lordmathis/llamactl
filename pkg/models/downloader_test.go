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

	plan := d.BuildDownloadPlan("org/model", "commit123", entries, "", "Q4_K_M", FormatGGUF)

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
		{Path: "model.safetensors", Type: "file", Size: 1000, LFS: &HFLFSInfo{OID: "abc123", Size: 1000}},
		{Path: "model-00001-of-00002.safetensors", Type: "file", Size: 500, LFS: &HFLFSInfo{OID: "def456", Size: 500}},
		{Path: "model-00002-of-00002.safetensors", Type: "file", Size: 500, LFS: &HFLFSInfo{OID: "ghi789", Size: 500}},
		{Path: "pytorch_model.bin", Type: "file", Size: 2000, LFS: &HFLFSInfo{OID: "bin123", Size: 2000}},
		{Path: "README.md", Type: "file", Size: 10},
		{Path: ".gitattributes", Type: "file", Size: 5},
	}

	plan := d.BuildDownloadPlan("org/model", "commit123", entries, "", "", FormatSafetensors)

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
		{Path: "pytorch_model.bin", Type: "file", Size: 2000, LFS: &HFLFSInfo{OID: "bin123", Size: 2000}},
		{Path: "model.pth", Type: "file", Size: 1500, LFS: &HFLFSInfo{OID: "pth123", Size: 1500}},
	}

	plan := d.BuildDownloadPlan("org/model", "commit123", entries, "", "", FormatSafetensors)

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
