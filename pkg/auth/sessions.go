package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session is an authenticated browser session created by OIDC login.
type Session struct {
	ID        string
	Sub       string
	Name      string
	Email     string
	ExpiresAt time.Time
}

// SessionStore keeps sessions in memory. Sessions do not survive restarts;
// users log in again. Expired sessions are removed lazily on access.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

// NewSessionStore creates a store where sessions live for ttl.
func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
}

// RandomToken returns 64 hex characters from crypto/rand. A broken CSPRNG
// is unrecoverable; failing loud beats failing open.
func RandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// Create starts a new session for the given OIDC user and returns it.
func (s *SessionStore) Create(sub, name, email string) *Session {
	id := RandomToken()

	sess := &Session{
		ID:        id,
		Sub:       sub,
		Name:      name,
		Email:     email,
		ExpiresAt: time.Now().Add(s.ttl),
	}

	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	return sess
}

// Get returns the session for id, or nil when it is unknown or expired.
func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()

	if !ok {
		return nil
	}

	if time.Now().After(sess.ExpiresAt) {
		s.Delete(id)
		return nil
	}

	return sess
}

// Delete removes the session for id if present.
func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}
