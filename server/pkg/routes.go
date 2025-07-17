package llamactl

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Define routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/server/help", HelpHandler)
		r.Get("/server/version", VersionHandler)
		r.Get("/server/devices", ListDevicesHandler)

		// Launch and stop handlers
		// r.Post("/server/launch/{model}", launchHandler)
		// r.Post("/server/stop/{model}", stopHandler)
	})

	return r
}
