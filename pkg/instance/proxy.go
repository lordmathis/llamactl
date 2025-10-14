package instance

import (
	"fmt"
	"llamactl/pkg/backends"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

// Proxy manages HTTP reverse proxy and request tracking for an instance.
type Proxy struct {
	process *Process // Owner reference - Proxy is owned by Process

	mu              sync.RWMutex
	proxy           *httputil.ReverseProxy
	proxyOnce       sync.Once
	proxyErr        error
	lastRequestTime atomic.Int64
	timeProvider    TimeProvider
}

// NewProxy creates a new Proxy for the given process
func NewProxy(process *Process) *Proxy {
	return &Proxy{
		process:      process,
		timeProvider: realTimeProvider{},
	}
}

// GetProxy returns the reverse proxy for this instance, creating it if needed.
// Uses sync.Once to ensure thread-safe one-time initialization.
func (p *Proxy) GetProxy() (*httputil.ReverseProxy, error) {
	// sync.Once guarantees buildProxy() is called exactly once
	// Other callers block until first initialization completes
	p.proxyOnce.Do(func() {
		p.proxy, p.proxyErr = p.buildProxy()
	})

	return p.proxy, p.proxyErr
}

// buildProxy creates the reverse proxy based on instance options
func (p *Proxy) buildProxy() (*httputil.ReverseProxy, error) {
	options := p.process.GetOptions()
	if options == nil {
		return nil, fmt.Errorf("instance %s has no options set", p.process.Name)
	}

	// Remote instances should not use local proxy - they are handled by RemoteInstanceProxy
	if len(options.Nodes) > 0 {
		return nil, fmt.Errorf("instance %s is a remote instance and should not use local proxy", p.process.Name)
	}

	// Get host/port from options
	var host string
	var port int
	switch options.BackendType {
	case backends.BackendTypeLlamaCpp:
		if options.LlamaServerOptions != nil {
			host = options.LlamaServerOptions.Host
			port = options.LlamaServerOptions.Port
		}
	case backends.BackendTypeMlxLm:
		if options.MlxServerOptions != nil {
			host = options.MlxServerOptions.Host
			port = options.MlxServerOptions.Port
		}
	case backends.BackendTypeVllm:
		if options.VllmServerOptions != nil {
			host = options.VllmServerOptions.Host
			port = options.VllmServerOptions.Port
		}
	}

	if host == "" {
		host = "localhost"
	}

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL for instance %s: %w", p.process.Name, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Get response headers from backend config
	var responseHeaders map[string]string
	switch options.BackendType {
	case backends.BackendTypeLlamaCpp:
		responseHeaders = p.process.globalBackendSettings.LlamaCpp.ResponseHeaders
	case backends.BackendTypeVllm:
		responseHeaders = p.process.globalBackendSettings.VLLM.ResponseHeaders
	case backends.BackendTypeMlxLm:
		responseHeaders = p.process.globalBackendSettings.MLX.ResponseHeaders
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove CORS headers from backend response to avoid conflicts
		// llamactl will add its own CORS headers
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Max-Age")
		resp.Header.Del("Access-Control-Expose-Headers")

		for key, value := range responseHeaders {
			resp.Header.Set(key, value)
		}
		return nil
	}

	return proxy, nil
}

// clearProxy resets the proxy, allowing it to be recreated when options change.
// This resets the sync.Once so the next GetProxy call will rebuild the proxy.
func (p *Proxy) clearProxy() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proxy = nil
	p.proxyErr = nil
	p.proxyOnce = sync.Once{} // Reset Once for next GetProxy call
}

// UpdateLastRequestTime updates the last request access time for the instance
func (p *Proxy) UpdateLastRequestTime() {
	lastRequestTime := p.timeProvider.Now().Unix()
	p.lastRequestTime.Store(lastRequestTime)
}

// LastRequestTime returns the last request time as a Unix timestamp
func (p *Proxy) LastRequestTime() int64 {
	return p.lastRequestTime.Load()
}

// ShouldTimeout checks if the instance should timeout based on idle time
func (p *Proxy) ShouldTimeout() bool {
	if !p.process.IsRunning() {
		return false
	}

	options := p.process.GetOptions()
	if options == nil || options.IdleTimeout == nil || *options.IdleTimeout <= 0 {
		return false
	}

	// Check if the last request time exceeds the idle timeout
	lastRequest := p.lastRequestTime.Load()
	idleTimeoutMinutes := *options.IdleTimeout

	// Convert timeout from minutes to seconds for comparison
	idleTimeoutSeconds := int64(idleTimeoutMinutes * 60)

	return (p.timeProvider.Now().Unix() - lastRequest) > idleTimeoutSeconds
}

// SetTimeProvider sets a custom time provider for testing
func (p *Proxy) SetTimeProvider(tp TimeProvider) {
	p.timeProvider = tp
}
