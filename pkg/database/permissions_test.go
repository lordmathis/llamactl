package database

import (
	"context"
	"llamactl/pkg/auth"
	"testing"
	"time"
)

// newTestDB opens a fresh in-memory database with migrations applied. A single
// open connection guarantees every query shares the same in-memory database.
func newTestDB(t *testing.T) *sqliteDB {
	t.Helper()
	db, err := Open(&Config{Path: ":memory:", MaxOpenConnections: 1})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}

func insertTestInstance(t *testing.T, db *sqliteDB, id int, name string) {
	t.Helper()
	now := time.Now().Unix()
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO instances (id, name, status, created_at, updated_at, options_json) VALUES (?, ?, 'stopped', ?, ?, '{}')`,
		id, name, now, now)
	if err != nil {
		t.Fatalf("insert instance %q: %v", name, err)
	}
}

func TestGetInstancePermission(t *testing.T) {
	db := newTestDB(t)
	insertTestInstance(t, db, 10, "model-a")

	now := time.Now().Unix()
	key := &auth.APIKey{
		KeyHash:        "hash",
		Name:           "k1",
		UserID:         "system",
		PermissionMode: auth.PermissionModePerInstance,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	perms := []auth.KeyPermission{
		{InstanceID: 10, CanStart: true, CanEvict: false},
	}
	if err := db.CreateKey(context.Background(), key, perms); err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	t.Run("existing row returns stored flags", func(t *testing.T) {
		got, err := db.GetInstancePermission(context.Background(), key.ID, 10)
		if err != nil {
			t.Fatalf("GetInstancePermission: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil perm")
		}
		if got.CanStart != true {
			t.Errorf("CanStart = %v, want true", got.CanStart)
		}
		if got.CanEvict != false {
			t.Errorf("CanEvict = %v, want false", got.CanEvict)
		}
	})

	t.Run("missing row returns nil nil", func(t *testing.T) {
		got, err := db.GetInstancePermission(context.Background(), key.ID, 999)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("expected nil perm for missing row, got %+v", got)
		}
	})
}

func TestGetPermissionsReturnsFlags(t *testing.T) {
	db := newTestDB(t)
	insertTestInstance(t, db, 20, "m20")
	insertTestInstance(t, db, 21, "m21")

	now := time.Now().Unix()
	key := &auth.APIKey{
		KeyHash:        "hash",
		Name:           "k2",
		UserID:         "system",
		PermissionMode: auth.PermissionModePerInstance,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	perms := []auth.KeyPermission{
		{InstanceID: 20, CanStart: false, CanEvict: true},
		{InstanceID: 21, CanStart: true, CanEvict: false},
	}
	if err := db.CreateKey(context.Background(), key, perms); err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	got, err := db.GetPermissions(context.Background(), key.ID)
	if err != nil {
		t.Fatalf("GetPermissions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 permissions, got %d", len(got))
	}
	// GetPermissions orders by instance_id.
	if got[0].InstanceID != 20 || got[0].CanStart != false || got[0].CanEvict != true {
		t.Errorf("perm[0] = %+v, want {InstanceID:20 CanStart:false CanEvict:true}", got[0])
	}
	if got[1].InstanceID != 21 || got[1].CanStart != true || got[1].CanEvict != false {
		t.Errorf("perm[1] = %+v, want {InstanceID:21 CanStart:true CanEvict:false}", got[1])
	}
}
