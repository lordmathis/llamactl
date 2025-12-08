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
// Migrated files are moved to a .migrated subdirectory to avoid re-importing.
func migrateFromJSON(cfg *config.AppConfig, db database.InstanceStore) error {
	instancesDir := cfg.Instances.InstancesDir
	if instancesDir == "" {
		return nil // No instances directory configured
	}

	// Check if instances directory exists
	if _, err := os.Stat(instancesDir); os.IsNotExist(err) {
		return nil // No instances directory, nothing to migrate
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

	// Create migrated directory
	migratedDir := filepath.Join(instancesDir, ".migrated")
	if err := os.MkdirAll(migratedDir, 0755); err != nil {
		return fmt.Errorf("failed to create migrated directory: %w", err)
	}

	// Migrate each JSON file
	var migrated int
	for _, file := range files {
		if err := migrateJSONFile(file, db); err != nil {
			log.Printf("Failed to migrate %s: %v", file, err)
			continue
		}

		// Move the file to the migrated directory
		destPath := filepath.Join(migratedDir, filepath.Base(file))
		if err := os.Rename(file, destPath); err != nil {
			log.Printf("Warning: Failed to move %s to migrated directory: %v", file, err)
			// Don't fail the migration if we can't move the file
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
