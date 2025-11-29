package database

import (
	"database/sql"
	"fmt"
	"llamactl/pkg/instance"
	"log"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database defines the interface for instance persistence operations
type Database interface {
	Save(inst *instance.Instance) error
	Delete(name string) error
	LoadAll() ([]*instance.Instance, error)
	Close() error
}

// Config contains database configuration settings
type Config struct {
	// Database file path (relative to data_dir or absolute)
	Path string

	// Connection settings
	MaxOpenConnections int
	MaxIdleConnections int
	ConnMaxLifetime    time.Duration
}

// DB wraps the database connection with configuration
type DB struct {
	*sql.DB
	config *Config
}

// Open creates a new database connection with the provided configuration
func Open(config *Config) (*DB, error) {
	if config == nil {
		return nil, fmt.Errorf("database config cannot be nil")
	}

	if config.Path == "" {
		return nil, fmt.Errorf("database path cannot be empty")
	}

	// Ensure the database directory exists
	dbDir := filepath.Dir(config.Path)
	if dbDir != "." && dbDir != "/" {
		// Directory will be created by the manager if auto_create_dirs is enabled
		log.Printf("Database will be created at: %s", config.Path)
	}

	// Open SQLite database with proper options
	// - _journal_mode=WAL: Write-Ahead Logging for better concurrency
	// - _busy_timeout=5000: Wait up to 5 seconds if database is locked
	// - _foreign_keys=1: Enable foreign key constraints (for future use)
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=1", config.Path)

	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConnections > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConnections)
	}
	if config.MaxIdleConnections > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConnections)
	}
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	// Verify database connection
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Database connection established: %s", config.Path)

	return &DB{
		DB:     sqlDB,
		config: config,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.DB != nil {
		log.Println("Closing database connection")
		return db.DB.Close()
	}
	return nil
}

// HealthCheck verifies the database is accessible
func (db *DB) HealthCheck() error {
	if db.DB == nil {
		return fmt.Errorf("database connection is nil")
	}
	return db.DB.Ping()
}
