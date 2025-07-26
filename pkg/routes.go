package llamactl

import (
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "llamactl/docs"
	"llamactl/webui"
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
		r.Route("/instances", func(r chi.Router) {
			r.Get("/", handler.ListInstances()) // List all instances

			r.Route("/{name}", func(r chi.Router) {
				// Instance management
				r.Get("/", handler.GetInstance())             // Get instance details
				r.Post("/", handler.CreateInstance())         // Create and start new instance
				r.Put("/", handler.UpdateInstance())          // Update instance configuration
				r.Delete("/", handler.DeleteInstance())       // Stop and remove instance
				r.Post("/start", handler.StartInstance())     // Start stopped instance
				r.Post("/stop", handler.StopInstance())       // Stop running instance
				r.Post("/restart", handler.RestartInstance()) // Restart instance
				r.Get("/logs", handler.GetInstanceLogs())     // Get instance logs

				// Llama.cpp server proxy endpoints (proxied to the actual llama.cpp server)
				r.Route("/proxy", func(r chi.Router) {
					r.HandleFunc("/*", handler.ProxyToInstance()) // Proxy all llama.cpp server requests
				})
			})
		})
	})

	// OpenAI-compatible proxy endpoint
	// Handles all POST requests to /v1/*, including:
	//   - /v1/completions
	//   - /v1/chat/completions
	//   - /v1/embeddings
	//   - /v1/rerank
	//   - /v1/reranking
	// The instance/model to use is determined by the request body.
	r.Post("/v1/*", handler.OpenAIProxy())

	// Serve WebUI files
	if err := webui.SetupWebUI(r); err != nil {
		fmt.Printf("Failed to set up WebUI: %v\n", err)
	}

	return r
}
