package instance

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// newLocalProxyOptions targets the given test backend; the options port *is* the proxy target.
func newLocalProxyOptions(t *testing.T, backendURL string) *Options {
	t.Helper()

	u, err := url.Parse(backendURL)
	if err != nil {
		t.Fatalf("failed to parse backend URL: %v", err)
	}
	host, portStr, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("failed to split backend host/port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse backend port: %v", err)
	}

	return &Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Host:  host,
				Port:  port,
			},
		},
	}
}

// retargetProxy aims inst at a test backend via the public options path.
func retargetProxy(t *testing.T, inst *Instance, backendURL string) {
	t.Helper()
	inst.SetOptions(newLocalProxyOptions(t, backendURL))
}

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

		retargetProxy(t, inst, blockingBackend.URL)

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

		retargetProxy(t, inst, blockingBackend.URL)

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

func newProxyTestConfig(t *testing.T, nodes map[string]config.NodeConfig) *config.AppConfig {
	t.Helper()
	return &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{Command: "llama-server", HealthPath: "/health"},
		},
		Instances: config.InstancesConfig{LogsDir: t.TempDir()},
		Nodes:     nodes,
		LocalNode: "main",
	}
}

func newRemoteProxyOptions(nodeNames ...string) *Options {
	nodes := make(map[string]struct{}, len(nodeNames))
	for _, name := range nodeNames {
		nodes[name] = struct{}{}
	}
	return &Options{
		Nodes: nodes,
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
			},
		},
	}
}

// newCredentialedRequest carries the controller's own credentials.
func newCredentialedRequest(target string) *http.Request {
	req := httptest.NewRequest("GET", target, nil)
	req.Header.Set("Authorization", "Bearer management-key")
	req.Header.Set("Proxy-Authorization", "Basic proxy-secret")
	req.Header.Set("Cookie", "llamactl_session=top-secret")
	req.Header.Set("X-API-Key", "inference-key")
	return req
}

// newCapturingBackend reports the headers of the first request it receives.
func newCapturingBackend(t *testing.T) (*httptest.Server, <-chan http.Header) {
	t.Helper()
	captured := make(chan http.Header, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case captured <- r.Header.Clone():
		default:
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(backend.Close)
	return backend, captured
}

func receiveHeaders(t *testing.T, captured <-chan http.Header) http.Header {
	t.Helper()
	select {
	case h := <-captured:
		return h
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for backend to receive a request")
		return nil
	}
}

func assertNoControllerCredentials(t *testing.T, h http.Header) {
	t.Helper()
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-API-Key"} {
		if got := h.Get(name); got != "" {
			t.Errorf("backend received controller credential %s: %q", name, got)
		}
	}
}

func TestScrubForwardedAuthHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer x")
	h.Set("Proxy-Authorization", "Basic y")
	h.Add("Cookie", "a=1")
	h.Add("Cookie", "b=2")
	h.Set("X-API-Key", "key")
	h.Set("Content-Type", "application/json")

	ScrubForwardedAuthHeaders(h)

	assertNoControllerCredentials(t, h)
	if h.Get("Content-Type") != "application/json" {
		t.Errorf("non-credential header must survive the scrub, got Content-Type %q", h.Get("Content-Type"))
	}
}

func TestProxy_LocalScrubsControllerCredentials(t *testing.T) {
	backend, captured := newCapturingBackend(t)
	cfg := newProxyTestConfig(t, nil)

	inst := New("local-scrub", cfg, newLocalProxyOptions(t, backend.URL), nil)

	rec := httptest.NewRecorder()
	if err := inst.ServeHTTP(rec, newCredentialedRequest("/v1/models")); err != nil {
		t.Fatalf("ServeHTTP failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	assertNoControllerCredentials(t, receiveHeaders(t, captured))
}

func TestProxy_RemoteReplacesCredentialsWithNodeKey(t *testing.T) {
	backend, captured := newCapturingBackend(t)
	cfg := newProxyTestConfig(t, map[string]config.NodeConfig{
		"worker": {Address: backend.URL, APIKey: "node-key"},
	})

	inst := New("remote-cred", cfg, newRemoteProxyOptions("worker"), nil)

	rec := httptest.NewRecorder()
	if err := inst.ServeHTTP(rec, newCredentialedRequest("/v1/models")); err != nil {
		t.Fatalf("ServeHTTP failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	h := receiveHeaders(t, captured)
	if got := h.Get("Authorization"); got != "Bearer node-key" {
		t.Errorf("backend Authorization = %q, want node credential", got)
	}
	for _, name := range []string{"Proxy-Authorization", "Cookie", "X-API-Key"} {
		if got := h.Get(name); got != "" {
			t.Errorf("backend received controller credential %s: %q", name, got)
		}
	}
}

func TestProxy_RemoteSelectionIsDeterministicAcrossRebuilds(t *testing.T) {
	aaaBackend, aaaCaptured := newCapturingBackend(t)
	bbbBackend, bbbCaptured := newCapturingBackend(t)
	cfg := newProxyTestConfig(t, map[string]config.NodeConfig{
		"aaa": {Address: aaaBackend.URL, APIKey: "aaa-key"},
		"bbb": {Address: bbbBackend.URL, APIKey: "bbb-key"},
	})

	inst := New("remote-multi", cfg, newRemoteProxyOptions("bbb", "aaa"), nil)

	for range 5 {
		// Fresh options each time, like a real options change would deliver.
		inst.SetOptions(newRemoteProxyOptions("bbb", "aaa"))
		rec := httptest.NewRecorder()
		if err := inst.ServeHTTP(rec, newCredentialedRequest("/v1/models")); err != nil {
			t.Fatalf("ServeHTTP failed: %v", err)
		}
	}

	select {
	case <-bbbCaptured:
		t.Error("proxy targeted node \"bbb\"; lexicographically first node \"aaa\" must always win")
	default:
	}
	h := receiveHeaders(t, aaaCaptured)
	if got := h.Get("Authorization"); got != "Bearer aaa-key" {
		t.Errorf("aaa Authorization = %q, want aaa-key credential", got)
	}
}

func TestProxy_PortChangeRetargetsToNewBackend(t *testing.T) {
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("A"))
	}))
	defer backendA.Close()
	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("B"))
	}))
	defer backendB.Close()

	cfg := newProxyTestConfig(t, nil)
	inst := New("port-change", cfg, newLocalProxyOptions(t, backendA.URL), nil)

	proxyBody := func() string {
		rec := httptest.NewRecorder()
		if err := inst.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/models", nil)); err != nil {
			t.Fatalf("ServeHTTP failed: %v", err)
		}
		return rec.Body.String()
	}

	if body := proxyBody(); body != "A" {
		t.Fatalf("initial proxy target = %q, want backend A", body)
	}

	inst.SetOptions(newLocalProxyOptions(t, backendB.URL))

	if body := proxyBody(); body != "B" {
		t.Fatalf("proxy target after port change = %q, want backend B (stale generation reused)", body)
	}
}

func TestProxy_ResponseHeadersRefreshOnOptionsChange(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := newProxyTestConfig(t, nil)
	cfg.Backends.LlamaCpp.ResponseHeaders = map[string]string{"X-Custom": "one"}
	inst := New("headers", cfg, newLocalProxyOptions(t, backend.URL), nil)

	serve := func() string {
		rec := httptest.NewRecorder()
		if err := inst.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/models", nil)); err != nil {
			t.Fatalf("ServeHTTP failed: %v", err)
		}
		return rec.Header().Get("X-Custom")
	}

	if got := serve(); got != "one" {
		t.Fatalf("X-Custom = %q, want %q", got, "one")
	}

	cfg.Backends.LlamaCpp.ResponseHeaders = map[string]string{"X-Custom": "two"}
	inst.SetOptions(newLocalProxyOptions(t, backend.URL))

	if got := serve(); got != "two" {
		t.Fatalf("X-Custom after options change = %q, want %q (headers captured at construction)", got, "two")
	}
}

func TestProxy_SetOptionsConcurrentWithServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := newProxyTestConfig(t, nil)
	inst := New("race", cfg, newLocalProxyOptions(t, backend.URL), nil)

	// Regression for the sync.Once reset race; must pass under -race.
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			_ = inst.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/models", nil))
		}
	}()
	go func() {
		defer wg.Done()
		// Fresh options each iteration: SetOptions must never re-validate an already-stored pointer.
		for range 500 {
			inst.SetOptions(newLocalProxyOptions(t, backend.URL))
		}
		close(done)
	}()
	wg.Wait()
}

func TestProxy_BuildErrorSurfacesOnRequest(t *testing.T) {
	cfg := newProxyTestConfig(t, nil)
	inst := New("portless", cfg, &Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: "/path/to/model.gguf",
				Port:  0,
			},
		},
	}, nil)

	err := inst.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/v1/models", nil))
	if err == nil {
		t.Fatal("expected build error for port 0, got none")
	}
	if !strings.Contains(err.Error(), "no port assigned") {
		t.Errorf("error = %v, want 'no port assigned'", err)
	}
}
