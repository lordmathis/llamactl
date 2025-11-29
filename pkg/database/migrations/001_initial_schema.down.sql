-- Drop indexes first
DROP INDEX IF EXISTS idx_instances_backend_type;
DROP INDEX IF EXISTS idx_instances_status;
DROP INDEX IF EXISTS idx_instances_name;

-- Drop tables
DROP TABLE IF EXISTS instances;
