-- -----------------------------------------------------------------------------
-- Instances Table: Central configuration and state for LLM backends
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS instances (
    -- Primary identification
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,

    -- Instance state
    status TEXT NOT NULL CHECK(status IN ('stopped', 'running', 'failed', 'restarting', 'shutting_down')) DEFAULT 'stopped',

    -- Timestamps (created_at stored as Unix timestamp for compatibility with existing JSON format)
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    -- All instance options stored as a single JSON blob
    options_json TEXT NOT NULL,

    -- Future: OIDC user ID for ownership
    owner_user_id TEXT NULL
);

-- -----------------------------------------------------------------------------
-- Indexes for performance
-- -----------------------------------------------------------------------------
CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_name ON instances(name);
CREATE INDEX IF NOT EXISTS idx_instances_status ON instances(status);

-- -----------------------------------------------------------------------------
-- API Keys Table: Database-backed inference API keys
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    user_id TEXT NOT NULL,
    permission_mode TEXT NOT NULL CHECK(permission_mode IN ('allow_all', 'per_instance')) DEFAULT 'per_instance',
    expires_at INTEGER NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_used_at INTEGER NULL
);

-- -----------------------------------------------------------------------------
-- Key Permissions Table: Per-instance permissions for API keys
-- -----------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS key_permissions (
    key_id INTEGER NOT NULL,
    instance_id INTEGER NOT NULL,
    can_infer INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (key_id, instance_id),
    FOREIGN KEY (key_id) REFERENCES api_keys (id) ON DELETE CASCADE,
    FOREIGN KEY (instance_id) REFERENCES instances (id) ON DELETE CASCADE
);

-- -----------------------------------------------------------------------------
-- Indexes for API keys and permissions
-- -----------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);
CREATE INDEX IF NOT EXISTS idx_key_permissions_instance_id ON key_permissions(instance_id);
