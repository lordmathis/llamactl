package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type PermissionMode string

const (
	PermissionModeAllowAll    PermissionMode = "allow_all"
	PermissionModePerInstance PermissionMode = "per_instance"
)

type APIKey struct {
	ID             int
	KeyHash        string
	KeyHMAC        string
	Name           string
	UserID         string
	PermissionMode PermissionMode
	ExpiresAt      *int64
	CreatedAt      int64
	UpdatedAt      int64
	LastUsedAt     *int64
}

type KeyPermission struct {
	KeyID      int
	InstanceID int
	CanStart   bool
	CanEvict   bool
}

// GenerateKey generates a cryptographically secure API key with the given prefix
func GenerateKey(prefix string) (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex (64 characters)
	hexStr := hex.EncodeToString(bytes)

	return fmt.Sprintf("%s-%s", prefix, hexStr), nil
}

// KeyFingerprint returns the hex HMAC-SHA256 of a token
func KeyFingerprint(secret []byte, token string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
