// Command dummy-server is a minimal OpenAI-compatible server for testing
// custom backends. Run it from the repo root:
//
//	go run ./examples/dummy-server --port 9000
//
// Endpoints:
//
//	GET  /health                readiness (503 until --delay elapses)
//	GET  /v1/models             canned model list
//	POST /v1/chat/completions   canned response echoing the request model
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

func main() {
	port := flag.Int("port", 0, "port to listen on (required)")
	delay := flag.Duration("delay", 0, "stay unhealthy for this long after startup")
	flag.Parse()

	if *port == 0 {
		log.Fatal("--port is required")
	}

	var ready atomic.Bool
	ready.Store(*delay <= 0)
	if *delay > 0 {
		time.AfterFunc(*delay, func() { ready.Store(true) })
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"object": "list",
			"data":   []map[string]any{{"id": "dummy-model", "object": "model"}},
		})
	})

	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("chat completion for model %q", req.Model)
		writeJSON(w, map[string]any{
			"id":     "chatcmpl-dummy",
			"object": "chat.completion",
			"model":  req.Model,
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]string{"role": "assistant", "content": "Hello from dummy-server"},
				"finish_reason": "stop",
			}},
		})
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Printf("dummy-server listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
