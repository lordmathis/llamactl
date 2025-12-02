package database

import (
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations applies all pending database migrations
func RunMigrations(db *sqliteDB) error {
	if db == nil || db.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Create migration source from embedded files
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver for migrations
	dbDriver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator
	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Get current version
	currentVersion, dirty, err := migrator.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("database is in dirty state at version %d - manual intervention required", currentVersion)
	}

	// Run migrations
	log.Printf("Running database migrations (current version: %v)", currentVersionString(currentVersion, err))

	if err := migrator.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("Database schema is up to date")
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get new version
	newVersion, _, err := migrator.Version()
	if err != nil {
		log.Printf("Migrations completed (unable to determine new version: %v)", err)
	} else {
		log.Printf("Migrations completed successfully (new version: %d)", newVersion)
	}

	return nil
}

// currentVersionString returns a string representation of the current version
func currentVersionString(version uint, err error) string {
	if err == migrate.ErrNilVersion {
		return "none"
	}
	return fmt.Sprintf("%d", version)
}
