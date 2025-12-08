package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2 parameters
	time    uint32 = 1
	memory  uint32 = 64 * 1024 // 64 MB
	threads uint8  = 4
	keyLen  uint32 = 32
	saltLen uint32 = 16
)

// HashKey hashes an API key using Argon2id
func HashKey(plainTextKey string) (string, error) {
	// Generate random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using Argon2id
	hash := argon2.IDKey([]byte(plainTextKey), salt, time, memory, threads, keyLen)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$<base64-salt>$<base64-hash>
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, time, threads, saltB64, hashB64), nil
}

// VerifyKey verifies a plain-text key against an Argon2id hash
func VerifyKey(plainTextKey, hash string) bool {
	// Parse the hash format
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	// Extract parameters
	var version, time, memory, threads int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != 19 {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Compute hash of the provided key
	computedHash := argon2.IDKey([]byte(plainTextKey), salt, uint32(time), uint32(memory), uint8(threads), uint32(len(expectedHash)))

	// Compare hashes using constant-time comparison
	return subtle.ConstantTimeCompare(computedHash, expectedHash) == 1
}
