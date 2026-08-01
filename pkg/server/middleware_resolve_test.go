package server

import (
	"context"
	"errors"
	"llamactl/pkg/auth"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"testing"
)

// stubAuthStore satisfies database.AuthStore for ResolveInstancePermission tests.
// Only GetInstancePermission is exercised here; the embedded nil interface makes the
// other methods available for interface satisfaction without implementation.
type stubAuthStore struct {
	database.AuthStore
	perm *auth.KeyPermission
	err  error
}

func (s *stubAuthStore) GetInstancePermission(ctx context.Context, keyID, instanceID int) (*auth.KeyPermission, error) {
	return s.perm, s.err
}

func newResolveMiddleware(store database.AuthStore) *APIAuthMiddleware {
	return NewAPIAuthMiddleware(config.AuthConfig{}, store)
}

func withKey(ctx context.Context, key *auth.APIKey) context.Context {
	return context.WithValue(ctx, apiKeyContextKey, key)
}

func TestResolveInstancePermission(t *testing.T) {
	const instID = 7

	perInstanceKey := &auth.APIKey{ID: 1, PermissionMode: auth.PermissionModePerInstance}
	allowAllKey := &auth.APIKey{ID: 2, PermissionMode: auth.PermissionModeAllowAll}

	tests := []struct {
		name      string
		ctx       context.Context
		store     *stubAuthStore
		wantStart bool
		wantEvict bool
		wantErr   bool
	}{
		{
			name:      "management key (no APIKey in context) short-circuits to all-true",
			ctx:       context.Background(),
			store:     &stubAuthStore{},
			wantStart: true,
			wantEvict: true,
		},
		{
			name:      "allow_all key short-circuits to all-true",
			ctx:       withKey(context.Background(), allowAllKey),
			store:     &stubAuthStore{},
			wantStart: true,
			wantEvict: true,
		},
		{
			name:      "per_instance key both flags true",
			ctx:       withKey(context.Background(), perInstanceKey),
			store:     &stubAuthStore{perm: &auth.KeyPermission{KeyID: 1, InstanceID: instID, CanStart: true, CanEvict: true}},
			wantStart: true,
			wantEvict: true,
		},
		{
			name:      "per_instance key can_start false",
			ctx:       withKey(context.Background(), perInstanceKey),
			store:     &stubAuthStore{perm: &auth.KeyPermission{KeyID: 1, InstanceID: instID, CanStart: false, CanEvict: true}},
			wantStart: false,
			wantEvict: true,
		},
		{
			name:      "per_instance key can_evict false",
			ctx:       withKey(context.Background(), perInstanceKey),
			store:     &stubAuthStore{perm: &auth.KeyPermission{KeyID: 1, InstanceID: instID, CanStart: true, CanEvict: false}},
			wantStart: true,
			wantEvict: false,
		},
		{
			name:    "per_instance key no permission row denied",
			ctx:     withKey(context.Background(), perInstanceKey),
			store:   &stubAuthStore{perm: nil, err: nil},
			wantErr: true,
		},
		{
			name:    "per_instance key db error propagated",
			ctx:     withKey(context.Background(), perInstanceKey),
			store:   &stubAuthStore{perm: nil, err: errors.New("db down")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := newResolveMiddleware(tt.store)
			perm, err := mw.ResolveInstancePermission(tt.ctx, instID)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (perm=%+v)", perm)
				}
				if perm != nil {
					t.Fatalf("expected nil perm on error, got %+v", perm)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if perm == nil {
				t.Fatal("expected non-nil perm")
			}
			if perm.CanStart != tt.wantStart {
				t.Errorf("CanStart = %v, want %v", perm.CanStart, tt.wantStart)
			}
			if perm.CanEvict != tt.wantEvict {
				t.Errorf("CanEvict = %v, want %v", perm.CanEvict, tt.wantEvict)
			}
		})
	}
}

// TestCheckInstancePermissionWrapper verifies the wrapper mirrors ResolveInstancePermission.
func TestCheckInstancePermissionWrapper(t *testing.T) {
	const instID = 3
	perInstanceKey := &auth.APIKey{ID: 1, PermissionMode: auth.PermissionModePerInstance}

	t.Run("allowed returns nil", func(t *testing.T) {
		mw := newResolveMiddleware(&stubAuthStore{perm: &auth.KeyPermission{InstanceID: instID, CanStart: true, CanEvict: true}})
		if err := mw.CheckInstancePermission(withKey(context.Background(), perInstanceKey), instID); err != nil {
			t.Errorf("expected nil error for allowed, got %v", err)
		}
	})

	t.Run("denied returns error", func(t *testing.T) {
		mw := newResolveMiddleware(&stubAuthStore{perm: nil})
		if err := mw.CheckInstancePermission(withKey(context.Background(), perInstanceKey), instID); err == nil {
			t.Errorf("expected error for denied, got nil")
		}
	})
}
