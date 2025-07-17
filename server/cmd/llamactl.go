package main

import (
	llamactl "llamactl/pkg"
	"net/http"
)

// @title Llama Server Control
// @version 1.0
// @description This is a control server for managing Llama Server instances.
// @license.name MIT License
// @license.url https://opensource.org/license/mit/
// @basePath /api/v1
func main() {
	r := llamactl.SetupRouter()
	// Start the server with the router
	http.ListenAndServe(":8080", r)
}
