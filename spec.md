# SQLite Database Persistence

This document describes the SQLite database persistence implementation for llamactl.

## Overview

Llamactl uses SQLite3 for persisting instance configurations and state. The database provides:
- Reliable instance persistence across restarts
- Automatic migration from legacy JSON files
- Prepared for future multi-user features

## Database Schema

### `instances` Table

Stores all instance configurations and state.

```sql
CREATE TABLE IF NOT EXISTS instances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,

    backend_type TEXT NOT NULL CHECK(backend_type IN ('llama_cpp', 'mlx_lm', 'vllm')),
    backend_config_json TEXT NOT NULL,

    status TEXT NOT NULL CHECK(status IN ('stopped', 'running', 'failed', 'restarting', 'shutting_down')) DEFAULT 'stopped',

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    auto_restart INTEGER NOT NULL DEFAULT 0,
    max_restarts INTEGER NOT NULL DEFAULT -1,
    restart_delay INTEGER NOT NULL DEFAULT 0,
    on_demand_start INTEGER NOT NULL DEFAULT 0,
    idle_timeout INTEGER NOT NULL DEFAULT 0,
    docker_enabled INTEGER NOT NULL DEFAULT 0,
    command_override TEXT,

    nodes TEXT,
    environment TEXT,

    owner_user_id TEXT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_name ON instances(name);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);
CREATE INDEX IF NOT EXISTS idx_instances_backend_type ON instances(backend_type);
```

## Architecture

### Database Layer (`pkg/database`)

The `database.DB` type implements the `Database` interface:

```go
// Database interface defines persistence operations
type Database interface {
    Save(inst *instance.Instance) error
    Delete(name string) error
    LoadAll() ([]*instance.Instance, error)
    Close() error
}

type DB struct {
    *sql.DB
    config *Config
}

// Database interface methods
func (db *DB) Save(inst *instance.Instance) error
func (db *DB) Delete(name string) error
func (db *DB) LoadAll() ([]*instance.Instance, error)
func (db *DB) Close() error

// Internal CRUD methods
func (db *DB) Create(ctx context.Context, inst *instance.Instance) error
func (db *DB) Update(ctx context.Context, inst *instance.Instance) error
func (db *DB) GetByName(ctx context.Context, name string) (*instance.Instance, error)
func (db *DB) GetAll(ctx context.Context) ([]*instance.Instance, error)
func (db *DB) DeleteInstance(ctx context.Context, name string) error
```

**Key points:**
- No repository pattern - DB directly implements persistence
- Simple, direct architecture with minimal layers
- Helper methods for row conversion are private to database package

### Manager Integration

Manager accepts a `Database` via dependency injection:

```go
func New(globalConfig *config.AppConfig, db database.Database) InstanceManager
```

Main creates the database, runs migrations, and injects it:

```go
// Initialize database
db, err := database.Open(&database.Config{
    Path:               cfg.Database.Path,
    MaxOpenConnections: cfg.Database.MaxOpenConnections,
    MaxIdleConnections: cfg.Database.MaxIdleConnections,
    ConnMaxLifetime:    cfg.Database.ConnMaxLifetime,
})

// Run database migrations
if err := database.RunMigrations(db); err != nil {
    log.Fatalf("Failed to run database migrations: %v", err)
}

// Migrate from JSON files if needed (one-time migration)
if err := migrateFromJSON(&cfg, db); err != nil {
    log.Printf("Warning: Failed to migrate from JSON: %v", err)
}

instanceManager := manager.New(&cfg, db)
```

### JSON Migration (`cmd/server/migrate_json.go`)

One-time migration utility that runs in main:

```go
func migrateFromJSON(cfg *config.AppConfig, db database.Database) error
```

**Note:** This migration code is temporary and can be removed in a future version (post-1.0) once most users have migrated from JSON to SQLite.

Handles:
- Automatic one-time JSON→SQLite migration
- Archiving old JSON files after migration

## Configuration

```yaml
database:
  path: "llamactl.db"  # Relative to data_dir or absolute
  max_open_connections: 25
  max_idle_connections: 5
  connection_max_lifetime: "5m"
```

Environment variables:
- `LLAMACTL_DATABASE_PATH`
- `LLAMACTL_DATABASE_MAX_OPEN_CONNECTIONS`
- `LLAMACTL_DATABASE_MAX_IDLE_CONNECTIONS`
- `LLAMACTL_DATABASE_CONN_MAX_LIFETIME`

## JSON Migration

On first startup with existing JSON files:

1. Database is created with schema
2. All JSON files are loaded and migrated to database
3. Original JSON files are moved to `{instances_dir}/json_archive/`
4. Subsequent startups use only the database

**Error Handling:**
- Failed migrations block application startup
- Original JSON files are preserved for rollback
- No fallback to JSON after migration

## Data Mapping

**Direct mappings:**
- `Instance.Name` → `instances.name`
- `Instance.Created` → `instances.created_at` (Unix timestamp)
- `Instance.Status` → `instances.status`

**Backend configuration:**
- `BackendOptions.BackendType` → `instances.backend_type`
- Typed backend options (LlamaServerOptions, etc.) → `instances.backend_config_json` (marshaled via MarshalJSON)

**Common options:**
- Boolean pointers (`*bool`) → INTEGER (0/1)
- Integer pointers (`*int`) → INTEGER
- `nil` values use column DEFAULT values
- `Nodes` map → `instances.nodes` (JSON array)
- `Environment` map → `instances.environment` (JSON object)

## Migrations

Uses `golang-migrate/migrate/v4` with embedded SQL files:

```
pkg/database/
├── database.go      # Database interface and DB type
├── migrations.go    # Migration runner
├── instances.go     # Instance CRUD operations
└── migrations/
    ├── 001_initial_schema.up.sql
    └── 001_initial_schema.down.sql

cmd/server/
└── migrate_json.go  # Temporary JSON→SQLite migration (can be removed post-1.0)
```

Migration files are embedded at compile time using `go:embed`.

## Testing

Tests use in-memory SQLite databases (`:memory:`) for speed, except when testing persistence across connections.

```go
appConfig.Database.Path = ":memory:"  // Fast in-memory database
```

For cross-connection persistence tests:
```go
appConfig.Database.Path = tempDir + "/test.db"  // File-based
```
