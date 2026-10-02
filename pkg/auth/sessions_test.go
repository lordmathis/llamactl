package auth

import (
	"testing"
	"time"
)

func TestSessionStoreLifecycle(t *testing.T) {
	store := NewSessionStore(time.Hour)

	sess := store.Create("user-1", "Alice", "alice@example.com")
	if sess.ID == "" {
		t.Fatal("expected session ID to be set")
	}

	got := store.Get(sess.ID)
	if got == nil {
		t.Fatal("expected session to be found")
	}
	if got.Sub != "user-1" || got.Name != "Alice" || got.Email != "alice@example.com" {
		t.Errorf("unexpected session contents: %+v", got)
	}

	store.Delete(sess.ID)
	if store.Get(sess.ID) != nil {
		t.Error("expected session to be gone after delete")
	}
}

func TestSessionStoreUnknownID(t *testing.T) {
	store := NewSessionStore(time.Hour)
	if store.Get("does-not-exist") != nil {
		t.Error("expected nil for unknown session ID")
	}
}

func TestSessionStoreExpiry(t *testing.T) {
	store := NewSessionStore(-time.Second)

	sess := store.Create("user-1", "Alice", "alice@example.com")
	if store.Get(sess.ID) != nil {
		t.Error("expected expired session to be rejected")
	}
}

func TestSessionIDsAreUnique(t *testing.T) {
	store := NewSessionStore(time.Hour)

	seen := make(map[string]bool)
	for range 100 {
		sess := store.Create("user-1", "Alice", "alice@example.com")
		if seen[sess.ID] {
			t.Fatalf("duplicate session ID generated: %s", sess.ID)
		}
		seen[sess.ID] = true
	}
}
