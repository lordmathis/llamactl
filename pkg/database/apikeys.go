package database

import (
	"context"
	"database/sql"
	"fmt"
	"llamactl/pkg/auth"
	"time"
)

// CreateKey inserts a new API key with permissions (transactional)
func (db *sqliteDB) CreateKey(ctx context.Context, key *auth.APIKey, permissions []auth.KeyPermission) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert the API key
	query := `
		INSERT INTO api_keys (key_hash, name, user_id, permission_mode, expires_at, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAt sql.NullInt64
	if key.ExpiresAt != nil {
		expiresAt = sql.NullInt64{Int64: *key.ExpiresAt, Valid: true}
	}

	result, err := tx.ExecContext(ctx, query,
		key.KeyHash, key.Name, key.UserID, key.PermissionMode,
		expiresAt, key.Enabled, key.CreatedAt, key.UpdatedAt,
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
				INSERT INTO key_permissions (key_id, instance_id, can_infer)
				VALUES (?, ?, ?)
			`
			_, err := tx.ExecContext(ctx, query, key.ID, perm.InstanceID, perm.CanInfer)
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
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, enabled, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE id = ?
	`

	var key auth.APIKey
	var expiresAt sql.NullInt64
	var lastUsedAt sql.NullInt64

	err := db.QueryRowContext(ctx, query, id).Scan(
		&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.PermissionMode,
		&expiresAt, &key.Enabled, &key.CreatedAt, &key.UpdatedAt, &lastUsedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to query API key: %w", err)
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Int64
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Int64
	}

	return &key, nil
}

// GetUserKeys retrieves all API keys for a user
func (db *sqliteDB) GetUserKeys(ctx context.Context, userID string) ([]*auth.APIKey, error) {
	query := `
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, enabled, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer rows.Close()

	var keys []*auth.APIKey
	for rows.Next() {
		var key auth.APIKey
		var expiresAt sql.NullInt64
		var lastUsedAt sql.NullInt64

		err := rows.Scan(
			&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.PermissionMode,
			&expiresAt, &key.Enabled, &key.CreatedAt, &key.UpdatedAt, &lastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Int64
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Int64
		}

		keys = append(keys, &key)
	}

	return keys, nil
}

// GetActiveKeys retrieves all enabled, non-expired API keys
func (db *sqliteDB) GetActiveKeys(ctx context.Context) ([]*auth.APIKey, error) {
	query := `
		SELECT id, key_hash, name, user_id, permission_mode, expires_at, enabled, created_at, updated_at, last_used_at
		FROM api_keys
		WHERE enabled = 1 AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY created_at DESC
	`

	now := time.Now().Unix()
	rows, err := db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query active API keys: %w", err)
	}
	defer rows.Close()

	var keys []*auth.APIKey
	for rows.Next() {
		var key auth.APIKey
		var expiresAt sql.NullInt64
		var lastUsedAt sql.NullInt64

		err := rows.Scan(
			&key.ID, &key.KeyHash, &key.Name, &key.UserID, &key.PermissionMode,
			&expiresAt, &key.Enabled, &key.CreatedAt, &key.UpdatedAt, &lastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Int64
		}
		if lastUsedAt.Valid {
			key.LastUsedAt = &lastUsedAt.Int64
		}

		keys = append(keys, &key)
	}

	return keys, nil
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
