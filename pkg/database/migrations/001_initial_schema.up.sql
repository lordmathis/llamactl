-- -----------------------------------------------------------------------------
-- Instances Table: Central configuration and state for LLM backends
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS instances (
    -- Primary identification
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,

    -- Backend configuration
    backend_type TEXT NOT NULL CHECK(backend_type IN ('llama_cpp', 'mlx_lm', 'vllm')),
    backend_config_json TEXT NOT NULL, -- Backend-specific options (150+ fields for llama_cpp, etc.)

    -- Instance state
    status TEXT NOT NULL CHECK(status IN ('stopped', 'running', 'failed', 'restarting', 'shutting_down')) DEFAULT 'stopped',

    -- Timestamps (created_at stored as Unix timestamp for compatibility with existing JSON format)
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    -- Common instance options (extracted from Instance.Options)
    -- NOT NULL with defaults to match config behavior (nil pointers use these defaults)
    auto_restart INTEGER NOT NULL DEFAULT 0,        -- Boolean: Enable automatic restart on failure
    max_restarts INTEGER NOT NULL DEFAULT -1,       -- Maximum restart attempts (-1 = unlimited)
    restart_delay INTEGER NOT NULL DEFAULT 0,       -- Delay between restarts in seconds
    on_demand_start INTEGER NOT NULL DEFAULT 0,     -- Boolean: Enable on-demand instance start
    idle_timeout INTEGER NOT NULL DEFAULT 0,        -- Idle timeout in minutes before auto-stop
    docker_enabled INTEGER NOT NULL DEFAULT 0,      -- Boolean: Run instance in Docker container
    command_override TEXT,                          -- Custom command to override default backend command (nullable)

    -- JSON fields for complex structures (nullable - empty when not set)
    nodes TEXT,                            -- JSON array of node names for remote instances
    environment TEXT,                      -- JSON map of environment variables

    -- Future extensibility hook
    owner_user_id TEXT NULL                -- Future: OIDC user ID for ownership
);

-- -----------------------------------------------------------------------------
-- Indexes for performance
-- -----------------------------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_name ON instances(name);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);
CREATE INDEX IF NOT EXISTS idx_instances_backend_type ON instances(backend_type);
