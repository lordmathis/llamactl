package models

import (
	"testing"
)

func TestDownloader_ParseSplitCount(t *testing.T) {
	d := NewDownloader("", 0, "", nil, nil)

	tests := []struct {
		name         string
		filepath     string
		wantCount    int
		wantErr      bool
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

