package database

import (
	"context"
	"llamactl/pkg/auth"
	"testing"
	"time"
)

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
