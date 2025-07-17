package main

import (
	"fmt"
	llamactl "llamactl/pkg"
	"net/http"
)

// @title llamactl API
// @version 1.0
// @description llamactl is a control server for managing Llama Server instances.
// @license.name MIT License
// @license.url https://opensource.org/license/mit/
// @basePath /api/v1
func main() {

	// Initialize the instance manager
	instanceManager := llamactl.NewInstanceManager()

	// Create a new handler with the instance manager
	handler := llamactl.NewHandler(instanceManager)

	// Setup the router with the handler
	r := llamactl.SetupRouter(handler)

	// Start the server with the router
	fmt.Println("Starting llamactl on port 8080...")
	http.ListenAndServe(":8080", r)
}
