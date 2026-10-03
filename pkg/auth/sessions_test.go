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

func TestSessionStoreExpiry(t *testing.T) {
	store := NewSessionStore(-time.Second)

	sess := store.Create("user-1", "Alice", "alice@example.com")
	if store.Get(sess.ID) != nil {
		t.Error("expected expired session to be rejected")
	}
}

func TestSessionStoreSweep(t *testing.T) {
	// Sessions that expired without being revisited must not linger forever
	expired := NewSessionStore(-time.Second)
	expired.Create("user-1", "Alice", "alice@example.com")
	expired.sweep(time.Now())
	if n := len(expired.sessions); n != 0 {
		t.Errorf("sweep left %d expired sessions behind", n)
	}

	// Live sessions survive
	live := NewSessionStore(time.Hour)
	sess := live.Create("user-2", "Bob", "bob@example.com")
	live.sweep(time.Now())
	if live.Get(sess.ID) == nil {
		t.Error("expected live session to survive the sweep")
	}
}
