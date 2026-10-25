package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llamactl/pkg/config"
	"llamactl/pkg/models"
	"llamactl/pkg/server"
)

// newModelsTestHandler wires a real manager, temp cache, and dead HF endpoint.
func newModelsTestHandler(t *testing.T) (*server.Handler, *models.Manager, string) {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("LLAMA_CACHE", cache)
	t.Setenv("HF_ENDPOINT", "http://127.0.0.1:1")
	mm := models.NewManager(cache, time.Second, "test")
	t.Cleanup(mm.Close)
	h := server.NewHandler(nil, mm, config.AppConfig{LocalNode: "local"}, nil, nil)
	return h, mm, cache
}

func TestDownloadModel_InvalidFormat(t *testing.T) {
	h, mm, _ := newModelsTestHandler(t)

	for _, format := range []string{"bogus", "GGUF"} {
		body := `{"repo":"org/model","format":"` + format + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/models/download", strings.NewReader(body))
		rec := httptest.NewRecorder()
		h.DownloadModel()(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("format %q: status = %d, want 400 (body: %s)", format, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "invalid_request") {
			t.Errorf("format %q: body %q missing invalid_request code", format, rec.Body.String())
		}
	}

	if jobs := mm.ListJobs(); len(jobs) != 0 {
		t.Errorf("expected no jobs created for invalid formats, got %d", len(jobs))
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/models/download", strings.NewReader(`{"repo":"org/model"}`))
	rec := httptest.NewRecorder()
	h.DownloadModel()(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Errorf("control case status = %d, want 202 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestDeleteModel(t *testing.T) {
	h, _, cache := newModelsTestHandler(t)

	request := func(url string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodDelete, url, nil)
		rec := httptest.NewRecorder()
		h.DeleteModel()(rec, req)
		return rec
	}

	t.Run("missing repo", func(t *testing.T) {
		rec := request("/api/v1/models?repo=org/missing")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "model_not_found") {
			t.Errorf("body %q missing model_not_found code", rec.Body.String())
		}
	})

	t.Run("missing ref", func(t *testing.T) {
		fm := models.NewFileManager(cache)
		repoDir := fm.HFRepoDir("org/model")
		if err := os.MkdirAll(filepath.Join(repoDir, "refs"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repoDir, "refs", "main"), []byte("abcdef1234567890abcdef1234567890abcdef12"), 0644); err != nil {
			t.Fatal(err)
		}

		rec := request("/api/v1/models?repo=org/model&tag=other")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "model_not_found") {
			t.Errorf("body %q missing model_not_found code", rec.Body.String())
		}
	})
}
