package server

import (
	"llamactl/pkg/config"
	"llamactl/pkg/manager"
	"net/http"
	"time"
)

type Handler struct {
	InstanceManager manager.InstanceManager
	cfg             config.AppConfig
	httpClient      *http.Client
}

func NewHandler(im manager.InstanceManager, cfg config.AppConfig) *Handler {
	return &Handler{
		InstanceManager: im,
		cfg:             cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
