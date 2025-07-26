package main

import (
	"fmt"
	llamactl "llamactl/pkg"
	"net/http"
	"os"
)

// @title llamactl API
// @version 1.0
// @description llamactl is a control server for managing Llama Server instances.
// @license.name MIT License
// @license.url https://opensource.org/license/mit/
// @basePath /api/v1
func main() {

	config, err := llamactl.LoadConfig("")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Using default configuration.")
	}

	// Create the log directory if it doesn't exist
	err = os.MkdirAll(config.Instances.LogDirectory, 0755)
	if err != nil {
		fmt.Printf("Error creating log directory: %v\n", err)
		return
	}

	// Initialize the instance manager
	instanceManager := llamactl.NewInstanceManager(config.Instances)

	// Create a new handler with the instance manager
	handler := llamactl.NewHandler(instanceManager)

	// Setup the router with the handler
	r := llamactl.SetupRouter(handler)

	// Start the server with the router
	fmt.Printf("Starting llamactl on port %d...\n", config.Server.Port)
	http.ListenAndServe(fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port), r)
}
