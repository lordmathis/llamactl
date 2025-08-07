package server

import (
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "llamactl/docs"
	"llamactl/webui"
)

func SetupRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Add CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   handler.cfg.Server.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Add API authentication middleware
	authMiddleware := NewAPIAuthMiddleware(handler.cfg.Auth)

	if handler.cfg.Server.EnableSwagger {
		r.Get("/swagger/*", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
	}

	// Define routes
	r.Route("/api/v1", func(r chi.Router) {

		if authMiddleware != nil && handler.cfg.Auth.RequireManagementAuth {
			r.Use(authMiddleware.AuthMiddleware(KeyTypeManagement))
		}

		r.Get("/version", handler.VersionHandler()) // Get server version

		r.Route("/server", func(r chi.Router) {
			r.Get("/help", handler.LlamaServerHelpHandler())
			r.Get("/version", handler.LlamaServerVersionHandler())
			r.Get("/devices", handler.LlamaServerListDevicesHandler())
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

	r.Route(("/v1"), func(r chi.Router) {

		if authMiddleware != nil && handler.cfg.Auth.RequireInferenceAuth {
			r.Use(authMiddleware.AuthMiddleware(KeyTypeInference))
		}

		r.Get(("/models"), handler.OpenAIListInstances()) // List instances in OpenAI-compatible format

		// OpenAI-compatible proxy endpoint
		// Handles all POST requests to /v1/*, including:
		//   - /v1/completions
		//   - /v1/chat/completions
		//   - /v1/embeddings
		//   - /v1/rerank
		//   - /v1/reranking
		// The instance/model to use is determined by the request body.
		r.Post("/*", handler.OpenAIProxy())

	})

	// Serve WebUI files
	if err := webui.SetupWebUI(r); err != nil {
		fmt.Printf("Failed to set up WebUI: %v\n", err)
	}

	return r
}
