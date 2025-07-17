package llamactl

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "llamactl/docs"
)

func SetupRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Define routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/server", func(r chi.Router) {
			r.Get("/help", handler.HelpHandler())
			r.Get("/version", handler.VersionHandler())
			r.Get("/devices", handler.ListDevicesHandler())
		})

		// Instance management endpoints
		// r.Get("/instances", handler.ListInstances())                    // List all instances
		// r.Post("/instances", handler.CreateInstance())                  // Create and start new instance
		// r.Get("/instances/{id}", handler.GetInstance())                 // Get instance details
		// r.Put("/instances/{id}", handler.UpdateInstance())              // Update instance configuration
		// r.Delete("/instances/{id}", handler.DeleteInstance())           // Stop and remove instance
		// r.Post("/instances/{id}/start", handler.StartInstance())        // Start stopped instance
		// r.Post("/instances/{id}/stop", handler.StopInstance())          // Stop running instance
		// r.Post("/instances/{id}/restart", handler.RestartInstance())    // Restart instance
		// r.Get("/instances/{id}/logs", handler.GetInstanceLogs())        // Get instance logs
	})

	return r
}
