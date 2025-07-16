package main

import (
	"net/http"
	"os"
	"os/exec"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var instances map[string]*exec.Cmd = make(map[string]*exec.Cmd)

func execLLama(model string) *exec.Cmd {
	llamaCmd := exec.Command("llama", "server", "--model", model, "--port", "8080")
	return llamaCmd
}

func launchHandler(w http.ResponseWriter, r *http.Request) {
	model := chi.URLParam(r, "model")
	if model == "" {
		http.Error(w, "Model parameter is required", http.StatusBadRequest)
		return
	}

	cmd := execLLama(model)
	if err := cmd.Start(); err != nil {
		http.Error(w, "Failed to start llama server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	instances[model] = cmd
	w.Write([]byte("Llama server started for model: " + model))
}

func stopHandler(w http.ResponseWriter, r *http.Request) {
	model := chi.URLParam(r, "model")
	if model == "" {
		http.Error(w, "Model parameter is required", http.StatusBadRequest)
		return
	}

	cmd, exists := instances[model]
	if !exists {
		http.Error(w, "No running instance for model: "+model, http.StatusNotFound)
		return
	}

	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		http.Error(w, "Failed to stop llama server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	delete(instances, model)
	w.Write([]byte("Llama server stopped for model: " + model))
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	r.Post("/launch/{model}", launchHandler)
	r.Post("/stop/{model}", stopHandler)
	http.ListenAndServe(":3000", r)
}
