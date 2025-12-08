package main

import (
	"context"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/manager"
	"llamactl/pkg/server"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// version is set at build time using -ldflags "-X main.version=1.0.0"
var version string = "unknown"
var commitHash string = "unknown"
var buildTime string = "unknown"

// @title llamactl API
// @version 1.0
// @description llamactl is a control server for managing Llama Server instances.
// @license.name MIT License
// @license.url https://opensource.org/license/mit/
// @basePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func main() {

	// --version flag to print the version
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("llamactl version: %s\n", version)
		fmt.Printf("Commit hash: %s\n", commitHash)
		fmt.Printf("Build time: %s\n", buildTime)
		return
	}

	configPath := os.Getenv("LLAMACTL_CONFIG_PATH")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Error loading config: %v\nUsing default configuration.", err)
	}

	// Set version information
	cfg.Version = version
	cfg.CommitHash = commitHash
	cfg.BuildTime = buildTime

	// Create data directory if it doesn't exist
	if cfg.Instances.AutoCreateDirs {
		// Create the main data directory
		if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
			log.Printf("Error creating data directory %s: %v\nData persistence may not be available.", cfg.DataDir, err)
		}

		// Create instances directory
		if err := os.MkdirAll(cfg.Instances.InstancesDir, 0755); err != nil {
			log.Printf("Error creating instances directory %s: %v\nPersistence will not be available.", cfg.Instances.InstancesDir, err)
		}

		// Create logs directory
		if err := os.MkdirAll(cfg.Instances.Logging.LogsDir, 0755); err != nil {
			log.Printf("Error creating log directory %s: %v\nInstance logs will not be available.", cfg.Instances.Logging.LogsDir, err)
		}
	}

	// Initialize database
	db, err := database.Open(&database.Config{
		Path:               cfg.Database.Path,
		MaxOpenConnections: cfg.Database.MaxOpenConnections,
		MaxIdleConnections: cfg.Database.MaxIdleConnections,
		ConnMaxLifetime:    cfg.Database.ConnMaxLifetime,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Run database migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Migrate from JSON files if needed (one-time migration)
	if err := migrateFromJSON(&cfg, db); err != nil {
		log.Printf("Warning: Failed to migrate from JSON: %v", err)
	}

	// Initialize the instance manager with dependency injection
	instanceManager := manager.New(&cfg, db)

	// Create a new handler with the instance manager
	handler := server.NewHandler(instanceManager, cfg, db)

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
			log.Printf("Error starting server: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-stop
	fmt.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down server: %v\n", err)
	} else {
		fmt.Println("Server shut down gracefully.")
	}

	// Stop all instances and cleanup
	instanceManager.Shutdown()

	if err := db.Close(); err != nil {
		log.Printf("Error closing database: %v\n", err)
	}

	fmt.Println("Exiting llamactl.")
}
