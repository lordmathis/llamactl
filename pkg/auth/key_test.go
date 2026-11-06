package auth

import "testing"

func TestKeyFingerprint(t *testing.T) {
	secret := []byte("server-secret")

	a := KeyFingerprint(secret, "llamactl-token")
	b := KeyFingerprint(secret, "llamactl-token")

	if a != b {
		t.Fatal("fingerprint must be deterministic to work as an index key")
	}
	if a == KeyFingerprint([]byte("other-secret"), "llamactl-token") {
		t.Error("fingerprint must depend on the server secret")
	}
	if len(a) != 64 {
		t.Errorf("fingerprint length = %d, want 64 hex chars", len(a))
	}
}
