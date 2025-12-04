package main

import (
	"encoding/json"
	"fmt"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/instance"
	"log"
	"os"
	"path/filepath"
)

// migrateFromJSON migrates instances from JSON files to SQLite database
// This is a one-time migration that runs on first startup with existing JSON files.
func migrateFromJSON(cfg *config.AppConfig, db database.InstanceStore) error {
	instancesDir := cfg.Instances.InstancesDir
	if instancesDir == "" {
		return nil // No instances directory configured
	}

	// Check if instances directory exists
	if _, err := os.Stat(instancesDir); os.IsNotExist(err) {
		return nil // No instances directory, nothing to migrate
	}

	// Check if database is empty (no instances)
	existing, err := db.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to check existing instances: %w", err)
	}

	if len(existing) > 0 {
		return nil // Database already has instances, skip migration
	}

	// Find all JSON files
	files, err := filepath.Glob(filepath.Join(instancesDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list instance files: %w", err)
	}

	if len(files) == 0 {
		return nil // No JSON files to migrate
	}

	log.Printf("Migrating %d instances from JSON to SQLite...", len(files))

	// Migrate each JSON file
	var migrated int
	for _, file := range files {
		if err := migrateJSONFile(file, db); err != nil {
			log.Printf("Failed to migrate %s: %v", file, err)
			continue
		}
		migrated++
	}

	log.Printf("Successfully migrated %d/%d instances to SQLite", migrated, len(files))

	return nil
}

// migrateJSONFile migrates a single JSON file to the database
func migrateJSONFile(filename string, db database.InstanceStore) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var inst instance.Instance
	if err := json.Unmarshal(data, &inst); err != nil {
		return fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	if err := db.Save(&inst); err != nil {
		return fmt.Errorf("failed to save instance to database: %w", err)
	}

	log.Printf("Migrated instance %s from JSON to SQLite", inst.Name)
	return nil
}
