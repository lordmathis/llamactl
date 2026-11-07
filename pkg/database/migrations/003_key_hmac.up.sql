-- API key fingerprint lookup
ALTER TABLE api_keys ADD COLUMN key_hmac TEXT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hmac
    ON api_keys(key_hmac) WHERE key_hmac IS NOT NULL;
CREATE TABLE IF NOT EXISTS server_secrets (
    name TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
