package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"log"
	"time"
)

// instanceRow represents a row in the instances table
type instanceRow struct {
	ID                int
	Name              string
	BackendType       string
	BackendConfigJSON string
	Status            string
	CreatedAt         int64
	UpdatedAt         int64
	AutoRestart       int
	MaxRestarts       int
	RestartDelay      int
	OnDemandStart     int
	IdleTimeout       int
	DockerEnabled     int
	CommandOverride   sql.NullString
	Nodes             sql.NullString
	Environment       sql.NullString
	OwnerUserID       sql.NullString
}

// Create inserts a new instance into the database
func (db *DB) Create(ctx context.Context, inst *instance.Instance) error {
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
			name, backend_type, backend_config_json, status,
			created_at, updated_at,
			auto_restart, max_restarts, restart_delay,
			on_demand_start, idle_timeout, docker_enabled,
			command_override, nodes, environment, owner_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.DB.ExecContext(ctx, query,
		row.Name, row.BackendType, row.BackendConfigJSON, row.Status,
		row.CreatedAt, row.UpdatedAt,
		row.AutoRestart, row.MaxRestarts, row.RestartDelay,
		row.OnDemandStart, row.IdleTimeout, row.DockerEnabled,
		row.CommandOverride, row.Nodes, row.Environment, row.OwnerUserID,
	)

	if err != nil {
		return fmt.Errorf("failed to insert instance: %w", err)
	}

	return nil
}

// GetByName retrieves an instance by name
func (db *DB) GetByName(ctx context.Context, name string) (*instance.Instance, error) {
	query := `
		SELECT id, name, backend_type, backend_config_json, status,
		       created_at, updated_at,
		       auto_restart, max_restarts, restart_delay,
		       on_demand_start, idle_timeout, docker_enabled,
		       command_override, nodes, environment, owner_user_id
		FROM instances
		WHERE name = ?
	`

	var row instanceRow
	err := db.DB.QueryRowContext(ctx, query, name).Scan(
		&row.ID, &row.Name, &row.BackendType, &row.BackendConfigJSON, &row.Status,
		&row.CreatedAt, &row.UpdatedAt,
		&row.AutoRestart, &row.MaxRestarts, &row.RestartDelay,
		&row.OnDemandStart, &row.IdleTimeout, &row.DockerEnabled,
		&row.CommandOverride, &row.Nodes, &row.Environment, &row.OwnerUserID,
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
func (db *DB) GetAll(ctx context.Context) ([]*instance.Instance, error) {
	query := `
		SELECT id, name, backend_type, backend_config_json, status,
		       created_at, updated_at,
		       auto_restart, max_restarts, restart_delay,
		       on_demand_start, idle_timeout, docker_enabled,
		       command_override, nodes, environment, owner_user_id
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
			&row.ID, &row.Name, &row.BackendType, &row.BackendConfigJSON, &row.Status,
			&row.CreatedAt, &row.UpdatedAt,
			&row.AutoRestart, &row.MaxRestarts, &row.RestartDelay,
			&row.OnDemandStart, &row.IdleTimeout, &row.DockerEnabled,
			&row.CommandOverride, &row.Nodes, &row.Environment, &row.OwnerUserID,
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
func (db *DB) Update(ctx context.Context, inst *instance.Instance) error {
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
			backend_type = ?, backend_config_json = ?, status = ?,
			updated_at = ?,
			auto_restart = ?, max_restarts = ?, restart_delay = ?,
			on_demand_start = ?, idle_timeout = ?, docker_enabled = ?,
			command_override = ?, nodes = ?, environment = ?
		WHERE name = ?
	`

	result, err := db.DB.ExecContext(ctx, query,
		row.BackendType, row.BackendConfigJSON, row.Status,
		row.UpdatedAt,
		row.AutoRestart, row.MaxRestarts, row.RestartDelay,
		row.OnDemandStart, row.IdleTimeout, row.DockerEnabled,
		row.CommandOverride, row.Nodes, row.Environment,
		row.Name,
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
func (db *DB) UpdateStatus(ctx context.Context, name string, status instance.Status) error {
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
func (db *DB) DeleteInstance(ctx context.Context, name string) error {
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
func (db *DB) instanceToRow(inst *instance.Instance) (*instanceRow, error) {
	opts := inst.GetOptions()
	if opts == nil {
		return nil, fmt.Errorf("instance options cannot be nil")
	}

	// Marshal backend options to JSON (this uses the MarshalJSON method which handles typed backends)
	backendJSON, err := json.Marshal(&opts.BackendOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backend options: %w", err)
	}

	// Extract just the backend_options field from the marshaled JSON
	var backendWrapper struct {
		BackendOptions map[string]any `json:"backend_options"`
	}
	if err := json.Unmarshal(backendJSON, &backendWrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backend wrapper: %w", err)
	}

	backendConfigJSON, err := json.Marshal(backendWrapper.BackendOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backend config: %w", err)
	}

	// Convert nodes map to JSON array
	var nodesJSON sql.NullString
	if len(opts.Nodes) > 0 {
		nodesList := make([]string, 0, len(opts.Nodes))
		for node := range opts.Nodes {
			nodesList = append(nodesList, node)
		}
		nodesBytes, err := json.Marshal(nodesList)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal nodes: %w", err)
		}
		nodesJSON = sql.NullString{String: string(nodesBytes), Valid: true}
	}

	// Convert environment map to JSON
	var envJSON sql.NullString
	if len(opts.Environment) > 0 {
		envBytes, err := json.Marshal(opts.Environment)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal environment: %w", err)
		}
		envJSON = sql.NullString{String: string(envBytes), Valid: true}
	}

	// Convert command override
	var cmdOverride sql.NullString
	if opts.CommandOverride != "" {
		cmdOverride = sql.NullString{String: opts.CommandOverride, Valid: true}
	}

	// Convert boolean pointers to integers (0 or 1)
	autoRestart := 0
	if opts.AutoRestart != nil && *opts.AutoRestart {
		autoRestart = 1
	}

	maxRestarts := -1
	if opts.MaxRestarts != nil {
		maxRestarts = *opts.MaxRestarts
	}

	restartDelay := 0
	if opts.RestartDelay != nil {
		restartDelay = *opts.RestartDelay
	}

	onDemandStart := 0
	if opts.OnDemandStart != nil && *opts.OnDemandStart {
		onDemandStart = 1
	}

	idleTimeout := 0
	if opts.IdleTimeout != nil {
		idleTimeout = *opts.IdleTimeout
	}

	dockerEnabled := 0
	if opts.DockerEnabled != nil && *opts.DockerEnabled {
		dockerEnabled = 1
	}

	now := time.Now().Unix()

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
		Name:              inst.Name,
		BackendType:       string(opts.BackendOptions.BackendType),
		BackendConfigJSON: string(backendConfigJSON),
		Status:            statusStr,
		CreatedAt:         inst.Created,
		UpdatedAt:         now,
		AutoRestart:       autoRestart,
		MaxRestarts:       maxRestarts,
		RestartDelay:      restartDelay,
		OnDemandStart:     onDemandStart,
		IdleTimeout:       idleTimeout,
		DockerEnabled:     dockerEnabled,
		CommandOverride:   cmdOverride,
		Nodes:             nodesJSON,
		Environment:       envJSON,
	}, nil
}

// rowToInstance converts a database row to an Instance
func (db *DB) rowToInstance(row *instanceRow) (*instance.Instance, error) {
	// Unmarshal backend config
	var backendConfig map[string]any
	if err := json.Unmarshal([]byte(row.BackendConfigJSON), &backendConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backend config: %w", err)
	}

	// Create backends.Options by marshaling and unmarshaling to trigger the UnmarshalJSON logic
	// This ensures the typed backend fields (LlamaServerOptions, VllmServerOptions, etc.) are populated
	var backendOptions backends.Options
	backendJSON, err := json.Marshal(map[string]any{
		"backend_type":    row.BackendType,
		"backend_options": backendConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backend for unmarshaling: %w", err)
	}

	if err := json.Unmarshal(backendJSON, &backendOptions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal backend options: %w", err)
	}

	// Unmarshal nodes
	var nodes map[string]struct{}
	if row.Nodes.Valid && row.Nodes.String != "" {
		var nodesList []string
		if err := json.Unmarshal([]byte(row.Nodes.String), &nodesList); err != nil {
			return nil, fmt.Errorf("failed to unmarshal nodes: %w", err)
		}
		nodes = make(map[string]struct{}, len(nodesList))
		for _, node := range nodesList {
			nodes[node] = struct{}{}
		}
	}

	// Unmarshal environment
	var environment map[string]string
	if row.Environment.Valid && row.Environment.String != "" {
		if err := json.Unmarshal([]byte(row.Environment.String), &environment); err != nil {
			return nil, fmt.Errorf("failed to unmarshal environment: %w", err)
		}
	}

	// Convert integers to boolean pointers
	autoRestart := row.AutoRestart == 1
	maxRestarts := row.MaxRestarts
	restartDelay := row.RestartDelay
	onDemandStart := row.OnDemandStart == 1
	idleTimeout := row.IdleTimeout
	dockerEnabled := row.DockerEnabled == 1

	// Create instance options
	opts := &instance.Options{
		AutoRestart:     &autoRestart,
		MaxRestarts:     &maxRestarts,
		RestartDelay:    &restartDelay,
		OnDemandStart:   &onDemandStart,
		IdleTimeout:     &idleTimeout,
		DockerEnabled:   &dockerEnabled,
		CommandOverride: row.CommandOverride.String,
		Nodes:           nodes,
		Environment:     environment,
		BackendOptions:  backendOptions,
	}

	// Create instance struct and manually unmarshal fields
	// We do this manually because BackendOptions and Nodes have json:"-" tags
	// and would be lost if we used the marshal/unmarshal cycle
	inst := &instance.Instance{
		Name:    row.Name,
		Created: row.CreatedAt,
	}

	// Create a temporary struct for unmarshaling the status and simple fields
	type instanceAux struct {
		Name    string `json:"name"`
		Created int64  `json:"created"`
		Status  string `json:"status"`
		Options struct {
			AutoRestart     *bool             `json:"auto_restart,omitempty"`
			MaxRestarts     *int              `json:"max_restarts,omitempty"`
			RestartDelay    *int              `json:"restart_delay,omitempty"`
			OnDemandStart   *bool             `json:"on_demand_start,omitempty"`
			IdleTimeout     *int              `json:"idle_timeout,omitempty"`
			DockerEnabled   *bool             `json:"docker_enabled,omitempty"`
			CommandOverride string            `json:"command_override,omitempty"`
			Environment     map[string]string `json:"environment,omitempty"`
		} `json:"options"`
	}

	aux := instanceAux{
		Name:    row.Name,
		Created: row.CreatedAt,
		Status:  row.Status,
	}
	aux.Options.AutoRestart = opts.AutoRestart
	aux.Options.MaxRestarts = opts.MaxRestarts
	aux.Options.RestartDelay = opts.RestartDelay
	aux.Options.OnDemandStart = opts.OnDemandStart
	aux.Options.IdleTimeout = opts.IdleTimeout
	aux.Options.DockerEnabled = opts.DockerEnabled
	aux.Options.CommandOverride = opts.CommandOverride
	aux.Options.Environment = opts.Environment

	instJSON, err := json.Marshal(aux)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal instance: %w", err)
	}

	if err := json.Unmarshal(instJSON, inst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance: %w", err)
	}

	// Manually set the fields that have json:"-" tags by using SetOptions
	// We need to set the whole options object because GetOptions returns a copy
	// and we need to ensure BackendOptions and Nodes (which have json:"-") are set
	inst.SetOptions(opts)

	return inst, nil
}

// Database interface implementation

// Save saves an instance to the database (insert or update)
func (db *DB) Save(inst *instance.Instance) error {
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
func (db *DB) Delete(name string) error {
	ctx := context.Background()
	return db.DeleteInstance(ctx, name)
}

// LoadAll loads all instances from the database
func (db *DB) LoadAll() ([]*instance.Instance, error) {
	ctx := context.Background()
	return db.GetAll(ctx)
}
