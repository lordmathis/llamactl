package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const hmacKeyName = "hmac_key"

// GetOrCreateHMACKey returns the server HMAC secret, creating and storing it
// on first use. Losing this secret invalidates every stored fingerprint.
func (db *sqliteDB) GetOrCreateHMACKey(ctx context.Context) ([]byte, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generating HMAC secret: %w", err)
	}

	_, err := db.ExecContext(ctx,
		"INSERT OR IGNORE INTO server_secrets (name, value) VALUES (?, ?)",
		hmacKeyName, hex.EncodeToString(secret))
	if err != nil {
		return nil, fmt.Errorf("storing HMAC secret: %w", err)
	}

	var stored string
	err = db.QueryRowContext(ctx,
		"SELECT value FROM server_secrets WHERE name = ?", hmacKeyName).Scan(&stored)
	if err != nil {
		return nil, fmt.Errorf("reading HMAC secret: %w", err)
	}

	decoded, err := hex.DecodeString(stored)
	if err != nil {
		return nil, fmt.Errorf("decoding HMAC secret: %w", err)
	}
	return decoded, nil
}
