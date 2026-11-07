package database

import (
	"context"
	"database/sql"
	"fmt"
	"llamactl/pkg/auth"
	"time"
)

// scanKey reads one api_keys row; shared by every key query.
func scanKey(scan func(dest ...any) error) (*auth.APIKey, error) {
	var key auth.APIKey
	var expiresAt, lastUsedAt sql.NullInt64

	if err := scan(
		&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.PermissionMode,
		&expiresAt, &key.CreatedAt, &key.UpdatedAt, &lastUsedAt,
	); err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Int64
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Int64
	}

	return &key, nil
}

// CreateKey inserts a new API key with permissions (transactional)
func (db *sqliteDB) CreateKey(ctx context.Context, key *auth.APIKey, permissions []auth.KeyPermission) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// NULL, never "", so the unique partial index tolerates fingerprint-less rows
	var keyHMAC sql.NullString
	if key.KeyHMAC != "" {
		keyHMAC = sql.NullString{String: key.KeyHMAC, Valid: true}
	}

	// Insert the API key
	query := `
		INSERT INTO api_keys (key_hash, key_hmac, name, user_id, permission_mode, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAt sql.NullInt64
	if key.ExpiresAt != nil {
		expiresAt = sql.NullInt64{Int64: *key.ExpiresAt, Valid: true}
	}

	result, err := tx.ExecContext(ctx, query,
		key.KeyHash, keyHMAC, key.Name, key.UserID, key.PermissionMode,
		expiresAt, key.CreatedAt, key.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert API key: %w", err)
	}

	keyID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	key.ID = int(keyID)

	// Insert permissions if per-instance mode
	if key.PermissionMode == auth.PermissionModePerInstance {
		for _, perm := range permissions {
			query := `
				INSERT INTO key_permissions (key_id, instance_id, can_start, can_evict)
				VALUES (?, ?, ?, ?)
			`
			_, err := tx.ExecContext(ctx, query, key.ID, perm.InstanceID, perm.CanStart, perm.CanEvict)
			if err != nil {
				return fmt.Errorf("failed to insert permission for instance %d: %w", perm.InstanceID, err)
			}
		}
	}

	return tx.Commit()
}

// GetKeyByID retrieves an API key by ID
func (db *sqliteDB) GetKeyByID(ctx context.Context, id int) (*auth.APIKey, error) {
	query := `
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE id = ?
	`

	key, err := scanKey(db.QueryRowContext(ctx, query, id).Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to query API key: %w", err)
	}

	return key, nil
}

// ListKeys retrieves all API keys, newest first
func (db *sqliteDB) ListKeys(ctx context.Context) ([]*auth.APIKey, error) {
	query := `
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var keys []*auth.APIKey
	for rows.Next() {
		key, err := scanKey(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate API keys: %w", err)
	}

	return keys, nil
}

// GetKeyByHMAC retrieves a non-expired API key by fingerprint.
func (db *sqliteDB) GetKeyByHMAC(ctx context.Context, fingerprint string) (*auth.APIKey, error) {
	query := `
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE key_hmac = ? AND (expires_at IS NULL OR expires_at > ?)
	`

	key, err := scanKey(db.QueryRowContext(ctx, query, fingerprint, time.Now().Unix()).Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query API key by fingerprint: %w", err)
	}

	return key, nil
}

// GetPendingLegacyKeys retrieves active keys that have no fingerprint yet and
// still verify through the Argon2 fallback.
func (db *sqliteDB) GetPendingLegacyKeys(ctx context.Context) ([]*auth.APIKey, error) {
	query := `
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE key_hmac IS NULL AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query pending legacy API keys: %w", err)
	}
	defer rows.Close()

	var keys []*auth.APIKey
	for rows.Next() {
		key, err := scanKey(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending legacy API key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pending legacy API keys: %w", err)
	}

	return keys, nil
}

// SetKeyHMAC stores the fingerprint of a key verified through the legacy path,
// so it never needs Argon2 again. The old key_hash is kept for v0.22 rollback.
func (db *sqliteDB) SetKeyHMAC(ctx context.Context, keyID int, fingerprint string) error {
	query := `UPDATE api_keys SET key_hmac = ?, updated_at = ? WHERE id = ?`

	_, err := db.ExecContext(ctx, query, fingerprint, time.Now().Unix(), keyID)
	if err != nil {
		return fmt.Errorf("failed to store API key fingerprint: %w", err)
	}

	return nil
}

// DeleteKey removes an API key (cascades to permissions)
func (db *sqliteDB) DeleteKey(ctx context.Context, id int) error {
	query := `DELETE FROM api_keys WHERE id = ?`

	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// TouchKey updates the last_used_at timestamp
func (db *sqliteDB) TouchKey(ctx context.Context, id int) error {
	query := `UPDATE api_keys SET last_used_at = ?, updated_at = ? WHERE id = ?`

	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last used timestamp: %w", err)
	}

	return nil
}
