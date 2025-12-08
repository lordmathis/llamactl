-- Drop API key related indexes and tables first
DROP INDEX IF EXISTS idx_key_permissions_instance_id;
DROP INDEX IF EXISTS idx_api_keys_expires_at;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP TABLE IF EXISTS key_permissions;
DROP TABLE IF EXISTS api_keys;

-- Drop instance related indexes and tables
DROP INDEX IF EXISTS idx_instances_status;
DROP INDEX IF EXISTS idx_instances_name;
DROP TABLE IF EXISTS instances;
