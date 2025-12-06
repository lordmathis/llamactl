package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"llamactl/pkg/instance"
	"log"
	"time"
)

// instanceRow represents a row in the instances table
type instanceRow struct {
	ID          int
	Name        string
	Status      string
	CreatedAt   int64
	UpdatedAt   int64
	OptionsJSON string
	OwnerUserID sql.NullString
}

// Create inserts a new instance into the database
func (db *sqliteDB) Create(ctx context.Context, inst *instance.Instance) error {
	if inst == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	opts := inst.GetOptions()
	if opts == nil {
		return fmt.Errorf("instance options cannot be nil")
	}

	// Convert instance to database row
	row, err := db.instanceToRow(inst)
	if err != nil {
		return fmt.Errorf("failed to convert instance to row: %w", err)
	}

	// Insert into database
	query := `
		INSERT INTO instances (
			name, status, created_at, updated_at, options_json, owner_user_id
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.DB.ExecContext(ctx, query,
		row.Name, row.Status, row.CreatedAt, row.UpdatedAt, row.OptionsJSON, row.OwnerUserID,
	)

	if err != nil {
		return fmt.Errorf("failed to insert instance: %w", err)
	}

	// Get the auto-generated ID and set it on the instance
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	inst.ID = int(id)

	return nil
}

// GetByName retrieves an instance by name
func (db *sqliteDB) GetByName(ctx context.Context, name string) (*instance.Instance, error) {
	query := `
		SELECT id, name, status, created_at, updated_at, options_json, owner_user_id
		FROM instances
		WHERE name = ?
	`

	var row instanceRow
	err := db.DB.QueryRowContext(ctx, query, name).Scan(
		&row.ID, &row.Name, &row.Status, &row.CreatedAt, &row.UpdatedAt, &row.OptionsJSON, &row.OwnerUserID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query instance: %w", err)
	}

	return db.rowToInstance(&row)
}

// GetAll retrieves all instances from the database
func (db *sqliteDB) GetAll(ctx context.Context) ([]*instance.Instance, error) {
	query := `
		SELECT id, name, status, created_at, updated_at, options_json, owner_user_id
		FROM instances
		ORDER BY created_at ASC
	`

	rows, err := db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query instances: %w", err)
	}
	defer rows.Close()

	var instances []*instance.Instance
	for rows.Next() {
		var row instanceRow
		err := rows.Scan(
			&row.ID, &row.Name, &row.Status, &row.CreatedAt, &row.UpdatedAt, &row.OptionsJSON, &row.OwnerUserID,
		)
		if err != nil {
			log.Printf("Failed to scan instance row: %v", err)
			continue
		}

		inst, err := db.rowToInstance(&row)
		if err != nil {
			log.Printf("Failed to convert row to instance: %v", err)
			continue
		}

		instances = append(instances, inst)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return instances, nil
}

// Update updates an existing instance
func (db *sqliteDB) Update(ctx context.Context, inst *instance.Instance) error {
	if inst == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	opts := inst.GetOptions()
	if opts == nil {
		return fmt.Errorf("instance options cannot be nil")
	}

	// Convert instance to database row
	row, err := db.instanceToRow(inst)
	if err != nil {
		return fmt.Errorf("failed to convert instance to row: %w", err)
	}

	// Update in database
	query := `
		UPDATE instances SET
			status = ?, updated_at = ?, options_json = ?
		WHERE name = ?
	`

	result, err := db.DB.ExecContext(ctx, query,
		row.Status, row.UpdatedAt, row.OptionsJSON, row.Name,
	)

	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", inst.Name)
	}

	return nil
}

// UpdateStatus updates only the status of an instance (optimized operation)
func (db *sqliteDB) UpdateStatus(ctx context.Context, name string, status instance.Status) error {
	// Convert status to string
	statusJSON, err := status.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}
	var statusStr string
	if err := json.Unmarshal(statusJSON, &statusStr); err != nil {
		return fmt.Errorf("failed to unmarshal status string: %w", err)
	}

	query := `
		UPDATE instances SET
			status = ?,
			updated_at = ?
		WHERE name = ?
	`

	result, err := db.DB.ExecContext(ctx, query, statusStr, time.Now().Unix(), name)
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", name)
	}

	return nil
}

// DeleteInstance removes an instance from the database
func (db *sqliteDB) DeleteInstance(ctx context.Context, name string) error {
	query := `DELETE FROM instances WHERE name = ?`

	result, err := db.DB.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found: %s", name)
	}

	return nil
}

// instanceToRow converts an Instance to a database row
func (db *sqliteDB) instanceToRow(inst *instance.Instance) (*instanceRow, error) {
	opts := inst.GetOptions()
	if opts == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	// Marshal options to JSON using the existing MarshalJSON method
	optionsJSON, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal options: %w", err)
	}

	// Convert status to string
	statusJSON, err := inst.GetStatus().MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status: %w", err)
	}
	var statusStr string
	if err := json.Unmarshal(statusJSON, &statusStr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status string: %w", err)
	}

	return &instanceRow{
		Name:        inst.Name,
		Status:      statusStr,
		CreatedAt:   inst.Created,
		UpdatedAt:   time.Now().Unix(),
		OptionsJSON: string(optionsJSON),
	}, nil
}

// rowToInstance converts a database row to an Instance
func (db *sqliteDB) rowToInstance(row *instanceRow) (*instance.Instance, error) {
	// Unmarshal options from JSON using the existing UnmarshalJSON method
	var opts instance.Options
	if err := json.Unmarshal([]byte(row.OptionsJSON), &opts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal options: %w", err)
	}

	// Build complete instance JSON with all fields
	instanceJSON, err := json.Marshal(map[string]any{
		"name":    row.Name,
		"created": row.CreatedAt,
		"status":  row.Status,
		"options": json.RawMessage(row.OptionsJSON),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal instance: %w", err)
	}

	// Unmarshal into a complete Instance
	var inst instance.Instance
	if err := json.Unmarshal(instanceJSON, &inst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	// The UnmarshalJSON doesn't handle BackendOptions and Nodes (they have json:"-" tags)
	// So we need to explicitly set the options again to ensure they're properly set
	inst.SetOptions(&opts)

	return &inst, nil
}

// Database interface implementation

// Save saves an instance to the database (insert or update)
func (db *sqliteDB) Save(inst *instance.Instance) error {
	ctx := context.Background()

	// Try to get existing instance
	existing, err := db.GetByName(ctx, inst.Name)
	if err != nil {
		// Instance doesn't exist, create it
		return db.Create(ctx, inst)
	}

	// Instance exists, update it
	if existing != nil {
		return db.Update(ctx, inst)
	}

	return db.Create(ctx, inst)
}

// Delete removes an instance from the database
func (db *sqliteDB) Delete(name string) error {
	ctx := context.Background()
	return db.DeleteInstance(ctx, name)
}

// LoadAll loads all instances from the database
func (db *sqliteDB) LoadAll() ([]*instance.Instance, error) {
	ctx := context.Background()
	return db.GetAll(ctx)
}
