package server_test

import (
	"io"
	"llamactl/pkg/config"
	"llamactl/pkg/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newLimitTestHandler(t *testing.T, maxBodyMB int) *server.Handler {
	t.Helper()
	cfg := config.AppConfig{}
	cfg.Server.MaxRequestBodyMB = maxBodyMB
	h, err := server.NewHandler(nil, nil, cfg, newTestAuthStore(t), nil)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	return h
}

func bigKeyJSON(size int) string {
	return `{"name":"k","permission_mode":"allow_all","pad":"` + strings.Repeat("a", size) + `"}`
}

func TestRequestBodyLimit_RejectsOversizedContentLength(t *testing.T) {
	h := newLimitTestHandler(t, 1)
	req := httptest.NewRequest("POST", "/api/v1/auth/keys",
		strings.NewReader(bigKeyJSON(2*1024*1024)))
	rec := httptest.NewRecorder()
	h.CreateKey()(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413, body %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "request_too_large") {
		t.Fatalf("body %s, want request_too_large error code", rec.Body.String())
	}
}

func TestRequestBodyLimit_RejectsOversizedChunkedBody(t *testing.T) {
	h := newLimitTestHandler(t, 1)
	// The wrapped reader hides the length so the chunked path is exercised.
	body := struct{ io.Reader }{strings.NewReader(bigKeyJSON(2 * 1024 * 1024))}
	req := httptest.NewRequest("POST", "/api/v1/auth/keys", body)
	rec := httptest.NewRecorder()
	h.CreateKey()(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413, body %s", rec.Code, rec.Body.String())
	}
}

func TestRequestBodyLimit_AllowsUnderLimitBody(t *testing.T) {
	h := newLimitTestHandler(t, 1)
	req := httptest.NewRequest("POST", "/api/v1/auth/keys",
		strings.NewReader(`{"name":"k","permission_mode":"allow_all"}`))
	rec := httptest.NewRecorder()
	h.CreateKey()(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201, body %s", rec.Code, rec.Body.String())
	}
}

func TestRequestBodyLimit_DisabledAllowsOversizedBody(t *testing.T) {
	h := newLimitTestHandler(t, -1)
	req := httptest.NewRequest("POST", "/api/v1/auth/keys",
		strings.NewReader(bigKeyJSON(2*1024*1024)))
	rec := httptest.NewRecorder()
	h.CreateKey()(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d, want 201, body %s", rec.Code, rec.Body.String())
	}
}

func TestOpenAIProxy_RejectsOversizedBody(t *testing.T) {
	h := newLimitTestHandler(t, 1)
	req := httptest.NewRequest("POST", "/v1/chat/completions",
		strings.NewReader(bigKeyJSON(2*1024*1024)))
	rec := httptest.NewRecorder()
	h.OpenAIProxy()(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d, want 413, body %s", rec.Code, rec.Body.String())
	}
}
