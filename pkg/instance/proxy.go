package instance

import (
	"fmt"
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

	targetURL *url.URL
	apiKey    string // For remote instances

	responseHeaders map[string]string

	mu sync.RWMutex

	proxy     *httputil.ReverseProxy
	proxyOnce sync.Once
	proxyErr  error

	lastRequestTime  atomic.Int64
	inflightRequests atomic.Int32
	timeProvider     TimeProvider
}

// newProxy creates a new Proxy for the given instance
func newProxy(instance *Instance) (*proxy, error) {

	p := &proxy{
		instance:     instance,
		timeProvider: realTimeProvider{},
	}

	var err error

	options := instance.GetOptions()
	if options == nil {
		return nil, fmt.Errorf("instance %s has no options set", instance.Name)
	}

	if instance.IsRemote() {

		// Take the first remote node as the target for now
		var nodeName string
		for node := range options.Nodes {
			nodeName = node
			break
		}

		if nodeName == "" {
			return nil, fmt.Errorf("instance %s has no remote nodes defined", p.instance.Name)
		}

		node, ok := p.instance.globalNodesConfig[nodeName]
		if !ok {
			return nil, fmt.Errorf("remote node %s is not defined", nodeName)
		}

		p.targetURL, err = url.Parse(node.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to parse target URL for remote instance %s: %w", p.instance.Name, err)
		}

		p.apiKey = node.APIKey
	} else {
		// Get host/port from process
		host := p.instance.options.GetHost()
		port := p.instance.options.GetPort()
		if port == 0 {
			return nil, fmt.Errorf("instance %s has no port assigned", p.instance.Name)
		}
		p.targetURL, err = url.Parse(fmt.Sprintf("http://%s:%d", host, port))
		if err != nil {
			return nil, fmt.Errorf("failed to parse target URL for instance %s: %w", p.instance.Name, err)
		}

		// Get response headers from backend config
		p.responseHeaders = options.BackendOptions.GetResponseHeaders(p.instance.globalBackendSettings)
	}

	return p, nil

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

	proxy := httputil.NewSingleHostReverseProxy(p.targetURL)

	// Modify the request before sending it to the backend
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Add API key header for remote instances
		if p.instance.IsRemote() && p.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+p.apiKey)
		}

		// Update last request time
		p.updateLastRequestTime()
	}

	if !p.instance.IsRemote() {
		// Add custom headers to the request
		proxy.ModifyResponse = func(resp *http.Response) error {
			// Remove CORS headers from backend response to avoid conflicts
			// llamactl will add its own CORS headers
			resp.Header.Del("Access-Control-Allow-Origin")
			resp.Header.Del("Access-Control-Allow-Methods")
			resp.Header.Del("Access-Control-Allow-Headers")
			resp.Header.Del("Access-Control-Allow-Credentials")
			resp.Header.Del("Access-Control-Max-Age")
			resp.Header.Del("Access-Control-Expose-Headers")

			for key, value := range p.responseHeaders {
				resp.Header.Set(key, value)
			}
			return nil
		}
	}

	return proxy, nil
}

// serveHTTP handles HTTP requests with inflight tracking and shutting down state checks
func (p *proxy) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	// Check if instance is shutting down
	status := p.instance.GetStatus()
	if status == ShuttingDown {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Instance is shutting down"))
		return fmt.Errorf("instance is shutting down")
	}

	// Get the reverse proxy
	reverseProxy, err := p.get()
	if err != nil {
		return err
	}

	// Track inflight requests
	p.incInflightRequests()
	defer p.decInflightRequests()

	// Serve the request
	reverseProxy.ServeHTTP(w, r)
	return nil
}

// clear resets the proxy, allowing it to be recreated when options change.
func (p *proxy) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.proxy = nil
	p.proxyErr = nil
	p.proxyOnce = sync.Once{}
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

// incInflightRequests increments the inflight request counter
func (p *proxy) incInflightRequests() {
	p.inflightRequests.Add(1)
}

// decInflightRequests decrements the inflight request counter
func (p *proxy) decInflightRequests() {
	p.inflightRequests.Add(-1)
}

// getInflightRequests returns the current number of inflight requests
func (p *proxy) getInflightRequests() int32 {
	return p.inflightRequests.Load()
}
