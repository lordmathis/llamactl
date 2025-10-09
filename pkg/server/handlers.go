package server

import (
	"llamactl/pkg/config"
	"llamactl/pkg/manager"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

type Handler struct {
	InstanceManager manager.InstanceManager
	cfg             config.AppConfig
	httpClient      *http.Client
	remoteProxies   map[string]*httputil.ReverseProxy // Cache of remote proxies by instance name
	remoteProxiesMu sync.RWMutex
}

func NewHandler(im manager.InstanceManager, cfg config.AppConfig) *Handler {
	return &Handler{
		InstanceManager: im,
		cfg:             cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		remoteProxies: make(map[string]*httputil.ReverseProxy),
	}
}
