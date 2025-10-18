package instance

import (
	"fmt"
	"llamactl/pkg/backends"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// TimeProvider interface allows for testing with mock time
type TimeProvider interface {
	Now() time.Time
}

// realTimeProvider implements TimeProvider using the actual time
type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time {
	return time.Now()
}

// proxy manages HTTP reverse proxy and request tracking for an instance.
type proxy struct {
	instance *Instance

	mu              sync.RWMutex
	proxy           *httputil.ReverseProxy
	proxyOnce       sync.Once
	proxyErr        error
	lastRequestTime atomic.Int64
	timeProvider    TimeProvider
}

// newProxy creates a new Proxy for the given instance
func newProxy(instance *Instance) *proxy {
	return &proxy{
		instance:     instance,
		timeProvider: realTimeProvider{},
	}
}

// get returns the reverse proxy for this instance, creating it if needed.
// Uses sync.Once to ensure thread-safe one-time initialization.
func (p *proxy) get() (*httputil.ReverseProxy, error) {
	// sync.Once guarantees buildProxy() is called exactly once
	// Other callers block until first initialization completes
	p.proxyOnce.Do(func() {
		p.proxy, p.proxyErr = p.build()
	})

	return p.proxy, p.proxyErr
}

// build creates the reverse proxy based on instance options
func (p *proxy) build() (*httputil.ReverseProxy, error) {
	options := p.instance.GetOptions()
	if options == nil {
		return nil, fmt.Errorf("instance %s has no options set", p.instance.Name)
	}

	// Remote instances should not use local proxy - they are handled by RemoteInstanceProxy
	if len(options.Nodes) > 0 {
		return nil, fmt.Errorf("instance %s is a remote instance and should not use local proxy", p.instance.Name)
	}

	// Get host/port from process
	host, port := p.instance.getBackendHostPort()

	targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	if err != nil {
		return nil, fmt.Errorf("failed to parse target URL for instance %s: %w", p.instance.Name, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Get response headers from backend config
	var responseHeaders map[string]string
	switch options.BackendType {
	case backends.BackendTypeLlamaCpp:
		responseHeaders = p.instance.globalBackendSettings.LlamaCpp.ResponseHeaders
	case backends.BackendTypeVllm:
		responseHeaders = p.instance.globalBackendSettings.VLLM.ResponseHeaders
	case backends.BackendTypeMlxLm:
		responseHeaders = p.instance.globalBackendSettings.MLX.ResponseHeaders
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

// clear resets the proxy, allowing it to be recreated when options change.
func (p *proxy) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proxy = nil
	p.proxyErr = nil
	p.proxyOnce = sync.Once{} // Reset Once for next GetProxy call
}

// updateLastRequestTime updates the last request access time for the instance
func (p *proxy) updateLastRequestTime() {
	lastRequestTime := p.timeProvider.Now().Unix()
	p.lastRequestTime.Store(lastRequestTime)
}

// getLastRequestTime returns the last request time as a Unix timestamp
func (p *proxy) getLastRequestTime() int64 {
	return p.lastRequestTime.Load()
}

// shouldTimeout checks if the instance should timeout based on idle time
func (p *proxy) shouldTimeout() bool {
	if !p.instance.IsRunning() {
		return false
	}

	options := p.instance.GetOptions()
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

// setTimeProvider sets a custom time provider for testing
func (p *proxy) setTimeProvider(tp TimeProvider) {
	p.timeProvider = tp
}
