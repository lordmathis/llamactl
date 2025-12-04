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
		SELECT key_id, instance_id, can_infer
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
		err := rows.Scan(&perm.KeyID, &perm.InstanceID, &perm.CanInfer)
		if err != nil {
			return nil, fmt.Errorf("failed to scan key permission: %w", err)
		}
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// HasPermission checks if key has inference permission for instance
func (db *sqliteDB) HasPermission(ctx context.Context, keyID, instanceID int) (bool, error) {
	query := `
		SELECT can_infer 
		FROM key_permissions 
		WHERE key_id = ? AND instance_id = ?
	`

	var canInfer bool
	err := db.QueryRowContext(ctx, query, keyID, instanceID).Scan(&canInfer)
	if err != nil {
		if err == sql.ErrNoRows {
			// No permission record found, deny access
			return false, nil
		}
		return false, fmt.Errorf("failed to check key permission: %w", err)
	}

	return canInfer, nil
}
