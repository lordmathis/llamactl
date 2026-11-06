DROP INDEX IF EXISTS idx_api_keys_key_hmac;
ALTER TABLE api_keys DROP COLUMN key_hmac;
DROP TABLE IF EXISTS server_secrets;
