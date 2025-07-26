//go:build !test

package webui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed dist/*
var webuiFS embed.FS

func SetupWebUI(r chi.Router) error {
	distFS, err := fs.Sub(webuiFS, "dist")
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(distFS))
	r.Handle("/*", fileServer)
	return nil
}
