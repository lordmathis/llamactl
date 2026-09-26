package instance

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestProxy_HealthRequestsExcluded(t *testing.T) {
	globalConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command:    "llama-server",
				HealthPath: "/health",
			},
		},
		Instances: config.InstancesConfig{
			LogsDir: t.TempDir(),
		},
		Nodes:     map[string]config.NodeConfig{},
		LocalNode: "main",
	}

	options := &Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  8080,
			},
		},
	}

	t.Run("Normal request increments inflight and updates last request time", func(t *testing.T) {
		inst := New("test-instance", globalConfig, options, nil)
		p := inst.proxy
		initialLastRequest := p.getLastRequestTime()
		
		req := httptest.NewRequest("GET", "/v1/chat/completions", nil)
		w := httptest.NewRecorder()
		
		backendDone := make(chan struct{})
		backendBlock := make(chan struct{})
		
		blockingBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			backendDone <- struct{}{}
			<-backendBlock
			w.WriteHeader(http.StatusOK)
		}))
		defer blockingBackend.Close()
		
		bu, _ := url.Parse(blockingBackend.URL)
		p.mu.Lock()
		p.targetURL = bu
		p.mu.Unlock()
		p.clear()

		go func() {
			p.serveHTTP(w, req)
		}()

		select {
		case <-backendDone:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for backend to be reached")
		}
		
		if p.getInflightRequests() != 1 {
			t.Errorf("Expected 1 inflight request, got %d", p.getInflightRequests())
		}
		
		backendBlock <- struct{}{}
		time.Sleep(50 * time.Millisecond)
		
		if p.getInflightRequests() != 0 {
			t.Errorf("Expected 0 inflight requests after completion, got %d", p.getInflightRequests())
		}
		
		if p.getLastRequestTime() <= initialLastRequest {
			t.Errorf("Expected last request time to be updated")
		}
	})

	t.Run("Health request does NOT increment inflight but UPDATES last request time", func(t *testing.T) {
		inst := New("test-instance-health", globalConfig, options, nil)
		p := inst.proxy
		p.lastRequestTime.Store(0)
		initialLastRequest := p.getLastRequestTime()
		
		backendDone := make(chan struct{})
		backendBlock := make(chan struct{})
		
		blockingBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			backendDone <- struct{}{}
			<-backendBlock
			w.WriteHeader(http.StatusOK)
		}))
		defer blockingBackend.Close()
		
		bu, _ := url.Parse(blockingBackend.URL)
		p.mu.Lock()
		p.targetURL = bu
		p.mu.Unlock()
		p.clear()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		
		go func() {
			p.serveHTTP(w, req)
		}()

		select {
		case <-backendDone:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for backend to be reached")
		}
		
		if p.getInflightRequests() != 0 {
			t.Errorf("Expected 0 inflight requests for health check, got %d", p.getInflightRequests())
		}
		
		backendBlock <- struct{}{}
		time.Sleep(50 * time.Millisecond)
		
		if p.getLastRequestTime() <= initialLastRequest {
			t.Errorf("Expected last request time to be updated for health check (dashboard use case)")
		}
	})
}
