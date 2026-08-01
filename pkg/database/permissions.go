package database

import (
	"context"
	"database/sql"
	"fmt"
	"llamactl/pkg/auth"
)

// GetPermissions retrieves all permissions for a key
func (db *sqliteDB) GetPermissions(ctx context.Context, keyID int) ([]auth.KeyPermission, error) {
	query := `
		SELECT key_id, instance_id, can_start, can_evict
		FROM key_permissions
		WHERE key_id = ?
		ORDER BY instance_id
	`

	rows, err := db.QueryContext(ctx, query, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query key permissions: %w", err)
	}
	defer rows.Close()

	var permissions []auth.KeyPermission
	for rows.Next() {
		var perm auth.KeyPermission
		err := rows.Scan(&perm.KeyID, &perm.InstanceID, &perm.CanStart, &perm.CanEvict)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// GetInstancePermission retrieves the full permission record for a key/instance pair.
// Returns (nil, nil) if no permission row exists.
func (db *sqliteDB) GetInstancePermission(ctx context.Context, keyID, instanceID int) (*auth.KeyPermission, error) {
	query := `
		SELECT key_id, instance_id, can_start, can_evict
		FROM key_permissions
		WHERE key_id = ? AND instance_id = ?
	`

	var perm auth.KeyPermission
	err := db.QueryRowContext(ctx, query, keyID, instanceID).Scan(
		&perm.KeyID, &perm.InstanceID, &perm.CanStart, &perm.CanEvict,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to check key permission: %w", err)
	}

	return &perm, nil
}
