package main

import (
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/manager"
	"llamactl/pkg/server"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// version is set at build time using -ldflags "-X main.version=1.0.0"
var version string = "unknown"

// @title llamactl API
// @version 1.0
// @description llamactl is a control server for managing Llama Server instances.
// @license.name MIT License
// @license.url https://opensource.org/license/mit/
// @basePath /api/v1
func main() {

	// --version flag to print the version
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("llamactl version: %s\n", version)
		return
	}

	configPath := os.Getenv("LLAMACTL_CONFIG_PATH")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		fmt.Println("Using default configuration.")
	}

	// Create the data directory if it doesn't exist
	if cfg.Instances.AutoCreateDirs {
		if err := os.MkdirAll(cfg.Instances.InstancesDir, 0755); err != nil {
			fmt.Printf("Error creating config directory %s: %v\n", cfg.Instances.InstancesDir, err)
			fmt.Println("Persistence will not be available.")
		}

		if err := os.MkdirAll(cfg.Instances.LogsDir, 0755); err != nil {
			fmt.Printf("Error creating log directory %s: %v\n", cfg.Instances.LogsDir, err)
			fmt.Println("Instance logs will not be available.")
		}
	}

	// Initialize the instance manager
	instanceManager := manager.NewInstanceManager(cfg.Instances)

	// Create a new handler with the instance manager
	handler := server.NewHandler(instanceManager, cfg)

	// Setup the router with the handler
	r := server.SetupRouter(handler)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: r,
	}

	go func() {
		fmt.Printf("Llamactl server listening on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
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
