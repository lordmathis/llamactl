package main

import (
	"fmt"
	llamactl "llamactl/pkg"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	handler := llamactl.NewHandler(instanceManager, config)

	// Setup the router with the handler
	r := llamactl.SetupRouter(handler)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler: r,
	}

	go func() {
		fmt.Printf("Llamactl server listening on %s:%d\n", config.Server.Host, config.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-stop
	fmt.Println("Shutting down server...")

	if err := server.Close(); err != nil {
		fmt.Printf("Error shutting down server: %v\n", err)
	} else {
		fmt.Println("Server shut down gracefully.")
	}

	// Wait for all instances to stop
	instanceManager.Shutdown()

	fmt.Println("Exiting llamactl.")
}
