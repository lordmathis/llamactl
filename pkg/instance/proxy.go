package instance

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type TimeProvider interface {
	Now() time.Time
}

type realTimeProvider struct{}

func (realTimeProvider) Now() time.Time {
	return time.Now()
}

// Controller credentials: they must never be forwarded to a backend or node.
var forwardedAuthHeaders = []string{
	"Authorization", "Proxy-Authorization", "Cookie", "X-API-Key",
}

// ScrubForwardedAuthHeaders removes controller credentials before forwarding.
func ScrubForwardedAuthHeaders(h http.Header) {
	for _, name := range forwardedAuthHeaders {
		h.Del(name)
	}
}

// Immutable once stored; SetOptions swaps in a new generation, never mutates one.
type proxyGeneration struct {
	reverseProxy *httputil.ReverseProxy
}

type proxy struct {
	instance *Instance

	mu  sync.Mutex // serializes generation builds; gen itself is atomic
	gen atomic.Pointer[proxyGeneration]

	lastRequestTime  atomic.Int64
	inflightRequests atomic.Int32
	timeProvider     TimeProvider
}

func newProxy(instance *Instance) *proxy {
	return &proxy{
		instance:     instance,
		timeProvider: realTimeProvider{},
	}
}

func (p *proxy) get() (*httputil.ReverseProxy, error) {
	if g := p.gen.Load(); g != nil {
		return g.reverseProxy, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if g := p.gen.Load(); g != nil {
		return g.reverseProxy, nil
	}

	g, err := p.build()
	if err != nil {
		return nil, err
	}
	p.gen.Store(g)
	return g.reverseProxy, nil
}

func (p *proxy) build() (*proxyGeneration, error) {
	options := p.instance.GetOptions()
	if options == nil {
		return nil, fmt.Errorf("instance %s has no options set", p.instance.Name)
	}

	targetURL, apiKey, err := p.resolveTarget(options)
	if err != nil {
		return nil, err
	}

	isRemote := p.instance.IsRemote()
	responseHeaders := options.BackendOptions.GetResponseHeaders(p.instance.globalBackendSettings)

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Scrub before the node credential is set, or the scrub would remove it.
		ScrubForwardedAuthHeaders(req.Header)

		if isRemote && apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}

		p.updateLastRequestTime()
	}

	if !isRemote {
		proxy.ModifyResponse = func(resp *http.Response) error {
			// llamactl adds its own CORS headers; backend ones would conflict.
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
	}

	return &proxyGeneration{reverseProxy: proxy}, nil
}

func (p *proxy) resolveTarget(options *Options) (*url.URL, string, error) {
	if !p.instance.IsRemote() {
		host := options.BackendOptions.GetHost()
		port := options.BackendOptions.GetPort()
		if port == 0 {
			return nil, "", fmt.Errorf("instance %s has no port assigned", p.instance.Name)
		}
		targetURL, err := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse target URL for instance %s: %w", p.instance.Name, err)
		}
		return targetURL, "", nil
	}

	// Sorted pick: map range order is random and the target must be stable across rebuilds.
	nodeNames := make([]string, 0, len(options.Nodes))
	for name := range options.Nodes {
		nodeNames = append(nodeNames, name)
	}
	slices.Sort(nodeNames)

	if len(nodeNames) == 0 {
		return nil, "", fmt.Errorf("instance %s has no remote nodes defined", p.instance.Name)
	}

	node, ok := p.instance.globalNodesConfig[nodeNames[0]]
	if !ok {
		return nil, "", fmt.Errorf("remote node %s is not defined", nodeNames[0])
	}

	targetURL, err := url.Parse(node.Address)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse target URL for remote instance %s: %w", p.instance.Name, err)
	}
	return targetURL, node.APIKey, nil
}

func (p *proxy) serveHTTP(w http.ResponseWriter, r *http.Request) error {
	reverseProxy, err := p.get()
	if err != nil {
		return err
	}

	isHealth := p.isHealthRequest(r)
	if !isHealth {
		p.incInflightRequests()
		defer p.decInflightRequests()
	}

	reverseProxy.ServeHTTP(w, r)
	return nil
}

func (p *proxy) isHealthRequest(r *http.Request) bool {
	return r.URL.Path == p.instance.GetHealthPath()
}

// In-flight requests keep the generation they captured.
func (p *proxy) clear() {
	p.gen.Store(nil)
}

func (p *proxy) updateLastRequestTime() {
	p.lastRequestTime.Store(p.timeProvider.Now().Unix())
}

func (p *proxy) getLastRequestTime() int64 {
	return p.lastRequestTime.Load()
}

func (p *proxy) shouldTimeout() bool {
	if !p.instance.IsRunning() {
		return false
	}

	options := p.instance.GetOptions()
	if options == nil || options.IdleTimeout == nil || *options.IdleTimeout <= 0 {
		return false
	}

	idleTimeoutSeconds := int64(*options.IdleTimeout * 60)
	return (p.timeProvider.Now().Unix() - p.lastRequestTime.Load()) > idleTimeoutSeconds
}

func (p *proxy) setTimeProvider(tp TimeProvider) {
	p.timeProvider = tp
}

func (p *proxy) incInflightRequests() {
	p.inflightRequests.Add(1)
}

func (p *proxy) decInflightRequests() {
	p.inflightRequests.Add(-1)
}

func (p *proxy) getInflightRequests() int32 {
	return p.inflightRequests.Load()
}
