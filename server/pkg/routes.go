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
		r.Route("/instances", func(r chi.Router) {
			r.Get("/", handler.ListInstances())   // List all instances
			r.Post("/", handler.CreateInstance()) // Create and start new instance

			r.Route("/{name}", func(r chi.Router) {
				// Instance management
				r.Get("/", handler.GetInstance())             // Get instance details
				r.Put("/", handler.UpdateInstance())          // Update instance configuration
				r.Delete("/", handler.DeleteInstance())       // Stop and remove instance
				r.Post("/start", handler.StartInstance())     // Start stopped instance
				r.Post("/stop", handler.StopInstance())       // Stop running instance
				r.Post("/restart", handler.RestartInstance()) // Restart instance
				// r.Get("/logs", handler.GetInstanceLogs())        // Get instance logs

				// Llama.cpp server proxy endpoints (proxied to the actual llama.cpp server)
				// r.Get("/health", handler.ProxyHealthCheck())         // Health check
				// r.Post("/completion", handler.ProxyCompletion())     // Text completion
				// r.Post("/tokenize", handler.ProxyTokenize())         // Tokenize text
				// r.Post("/detokenize", handler.ProxyDetokenize())     // Detokenize tokens
				// r.Post("/apply-template", handler.ProxyApplyTemplate()) // Apply chat template
				// r.Post("/embedding", handler.ProxyEmbedding())       // Generate embeddings
				// r.Post("/reranking", handler.ProxyReranking())       // Rerank documents
				// r.Post("/rerank", handler.ProxyRerank())             // Rerank documents (alias)
				// r.Post("/infill", handler.ProxyInfill())             // Code infilling
				// r.Get("/props", handler.ProxyGetProps())             // Get server properties
				// r.Post("/props", handler.ProxySetProps())            // Set server properties
				// r.Post("/embeddings", handler.ProxyEmbeddings())     // Non-OpenAI embeddings
				// r.Get("/slots", handler.ProxyGetSlots())             // Get slots state
				// r.Get("/metrics", handler.ProxyGetMetrics())         // Prometheus metrics
				// r.Post("/slots/{slot_id}", handler.ProxySlotAction()) // Slot actions (save/restore/erase)
				// r.Get("/lora-adapters", handler.ProxyGetLoraAdapters()) // Get LoRA adapters
				// r.Post("/lora-adapters", handler.ProxySetLoraAdapters()) // Set LoRA adapters

				// OpenAI-compatible endpoints (proxied to the actual llama.cpp server)
				// r.Post("/v1/completions", handler.ProxyV1Completions()) // OpenAI completions
				// r.Post("/v1/chat/completions", handler.ProxyV1ChatCompletions()) // OpenAI chat completions
				// r.Post("/v1/embeddings", handler.ProxyV1Embeddings())   // OpenAI embeddings
				// r.Post("/v1/rerank", handler.ProxyV1Rerank())           // OpenAI rerank
				// r.Post("/v1/reranking", handler.ProxyV1Reranking())     // OpenAI reranking
			})
		})
	})

	// OpenAI-compatible endpoints (model name in request body determines routing)
	// r.Post("/v1/completions", handler.OpenAICompletions())         // Route based on model name in request
	// r.Post("/v1/chat/completions", handler.OpenAIChatCompletions()) // Route based on model name in request
	// r.Post("/v1/embeddings", handler.OpenAIEmbeddings())           // Route based on model name in request (if supported)
	// r.Post("/v1/rerank", handler.OpenAIRerank())                   // Route based on model name in request (if supported)
	// r.Post("/v1/reranking", handler.OpenAIReranking())             // Route based on model name in request (if supported)

	return r
}
