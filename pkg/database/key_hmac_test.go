package database

import (
	"context"
	"llamactl/pkg/auth"
	"testing"
	"time"
)

func TestMigration003DownAndUp(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	if _, err := db.GetOrCreateHMACKey(ctx); err != nil {
		t.Fatalf("seeding secret: %v", err)
	}

	migrator, err := newMigrator(db)
	if err != nil {
		t.Fatalf("creating migrator: %v", err)
	}
	if err := migrator.Steps(-1); err != nil {
		t.Fatalf("stepping down from version 3: %v", err)
	}
	if err := migrator.Steps(1); err != nil {
		t.Fatalf("stepping back up to version 3: %v", err)
	}

	if _, err := db.GetOrCreateHMACKey(ctx); err != nil {
		t.Fatalf("secret machinery broken after down/up: %v", err)
	}
}

func TestGetOrCreateHMACKeyIdempotent(t *testing.T) {
	db := newTestDB(t)

	first, err := db.GetOrCreateHMACKey(context.Background())
	if err != nil {
		t.Fatalf("first GetOrCreateHMACKey: %v", err)
	}
	second, err := db.GetOrCreateHMACKey(context.Background())
	if err != nil {
		t.Fatalf("second GetOrCreateHMACKey: %v", err)
	}

	if string(first) != string(second) {
		t.Fatal("second call returned a different secret; existing keys would become unverifiable")
	}
	if len(first) != 32 {
		t.Errorf("secret length = %d bytes, want 32", len(first))
	}
}

func TestCreateKeyAllowsEmptyFingerprint(t *testing.T) {
	db := newTestDB(t)
	now := time.Now().Unix()

	for _, name := range []string{"legacy-a", "legacy-b"} {
		key := &auth.APIKey{
			KeyHash:        "hash-" + name,
			Name:           name,
			UserID:         "system",
			PermissionMode: auth.PermissionModeAllowAll,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := db.CreateKey(context.Background(), key, nil); err != nil {
			t.Fatalf("creating fingerprint-less key %q: %v", name, err)
		}
	}
}

func TestGetKeyByHMAC(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().Unix()

	active := &auth.APIKey{
		KeyHash:        "hash",
		KeyHMAC:        "fp-active",
		Name:           "active",
		UserID:         "system",
		PermissionMode: auth.PermissionModeAllowAll,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.CreateKey(ctx, active, nil); err != nil {
		t.Fatalf("creating active key: %v", err)
	}

	expired := &auth.APIKey{
		KeyHash:        "hash",
		KeyHMAC:        "fp-expired",
		Name:           "expired",
		UserID:         "system",
		PermissionMode: auth.PermissionModeAllowAll,
		ExpiresAt:      &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.CreateKey(ctx, expired, nil); err != nil {
		t.Fatalf("creating expired key: %v", err)
	}

	key, err := db.GetKeyByHMAC(ctx, "fp-active")
	if err != nil || key == nil {
		t.Fatalf("fingerprint lookup for active key: key=%v err=%v", key, err)
	}
	if key.Name != "active" {
		t.Errorf("lookup returned key %q, want %q", key.Name, "active")
	}

	key, err = db.GetKeyByHMAC(ctx, "fp-expired")
	if err != nil {
		t.Fatalf("expired lookup: %v", err)
	}
	if key != nil {
		t.Error("expired key must not match a fingerprint lookup")
	}

	key, err = db.GetKeyByHMAC(ctx, "fp-unknown")
	if err != nil {
		t.Fatalf("unknown lookup: %v", err)
	}
	if key != nil {
		t.Error("unknown fingerprint must return nil, nil")
	}
}

func TestPendingLegacyKeysShrinkOnUpgrade(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().Unix()

	legacy := &auth.APIKey{
		KeyHash:        "hash",
		Name:           "legacy",
		UserID:         "system",
		PermissionMode: auth.PermissionModeAllowAll,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.CreateKey(ctx, legacy, nil); err != nil {
		t.Fatalf("creating legacy key: %v", err)
	}
	upgraded := &auth.APIKey{
		KeyHash:        "hash",
		KeyHMAC:        "fp-upgraded",
		Name:           "upgraded",
		UserID:         "system",
		PermissionMode: auth.PermissionModeAllowAll,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.CreateKey(ctx, upgraded, nil); err != nil {
		t.Fatalf("creating upgraded key: %v", err)
	}

	pending, err := db.GetPendingLegacyKeys(ctx)
	if err != nil {
		t.Fatalf("GetPendingLegacyKeys: %v", err)
	}
	if len(pending) != 1 || pending[0].Name != "legacy" {
		t.Fatalf("pending keys = %v, want only the fingerprint-less legacy key", pending)
	}

	if err := db.SetKeyHMAC(ctx, legacy.ID, "fp-legacy"); err != nil {
		t.Fatalf("SetKeyHMAC: %v", err)
	}

	pending, err = db.GetPendingLegacyKeys(ctx)
	if err != nil {
		t.Fatalf("GetPendingLegacyKeys after upgrade: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending keys after upgrade = %d, want 0", len(pending))
	}
}
