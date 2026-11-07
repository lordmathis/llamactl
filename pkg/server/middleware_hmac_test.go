package server_test

import (
	"context"
	"encoding/json"
	"llamactl/pkg/auth"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/server"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestAuthStore(t *testing.T) database.AuthStore {
	t.Helper()
	db, err := database.Open(&database.Config{Path: ":memory:", MaxOpenConnections: 1})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}

func doInferenceRequest(t *testing.T, mw *server.APIAuthMiddleware, token string) int {
	t.Helper()
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mw.InferenceAuthMiddleware()(ok).ServeHTTP(rec, req)
	return rec.Code
}

// A pre-fingerprint key must authenticate through the legacy Argon2 scan and
// gain a fingerprint, so it never needs the scan again.
func TestInferenceAuthLegacyKeyUpgradesOnFirstUse(t *testing.T) {
	store := newTestAuthStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	plainText, err := auth.GenerateKey("llamactl")
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	keyHash, err := auth.HashKey(plainText)
	if err != nil {
		t.Fatalf("hashing key: %v", err)
	}
	legacyKey := &auth.APIKey{
		KeyHash:        keyHash,
		Name:           "legacy",
		UserID:         "system",
		PermissionMode: auth.PermissionModeAllowAll,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.CreateKey(ctx, legacyKey, nil); err != nil {
		t.Fatalf("creating legacy key: %v", err)
	}

	mw, err := server.NewAPIAuthMiddleware(config.AuthConfig{RequireInferenceAuth: true}, store)
	if err != nil {
		t.Fatalf("NewAPIAuthMiddleware: %v", err)
	}

	if code := doInferenceRequest(t, mw, plainText); code != http.StatusOK {
		t.Fatalf("first request with legacy key: status %d, want 200", code)
	}

	secret, err := store.GetOrCreateHMACKey(ctx)
	if err != nil {
		t.Fatalf("getting HMAC secret: %v", err)
	}
	upgraded, err := store.GetKeyByHMAC(ctx, auth.KeyFingerprint(secret, plainText))
	if err != nil || upgraded == nil {
		t.Fatalf("legacy key not upgraded after first use: key=%v err=%v", upgraded, err)
	}

	pending, err := store.GetPendingLegacyKeys(ctx)
	if err != nil {
		t.Fatalf("GetPendingLegacyKeys: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("pending legacy keys after upgrade = %d, want 0", len(pending))
	}

	if code := doInferenceRequest(t, mw, plainText); code != http.StatusOK {
		t.Fatalf("second request with upgraded key: status %d, want 200", code)
	}
}

// A fingerprint-bearing key must authenticate without any valid Argon2 hash:
// if the fast path ever fell through to the legacy scan, the garbage hash
// would fail verification and this request would get a 401.
func TestInferenceAuthFingerprintKeySkipsArgon2(t *testing.T) {
	store := newTestAuthStore(t)
	ctx := context.Background()
	now := time.Now().Unix()

	plainText, err := auth.GenerateKey("llamactl")
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}
	secret, err := store.GetOrCreateHMACKey(ctx)
	if err != nil {
		t.Fatalf("getting HMAC secret: %v", err)
	}
	dbKey := &auth.APIKey{
		KeyHash:        "not-an-argon2-hash",
		KeyHMAC:        auth.KeyFingerprint(secret, plainText),
		Name:           "fingerprinted",
		UserID:         "system",
		PermissionMode: auth.PermissionModeAllowAll,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.CreateKey(ctx, dbKey, nil); err != nil {
		t.Fatalf("creating fingerprinted key: %v", err)
	}

	mw, err := server.NewAPIAuthMiddleware(config.AuthConfig{RequireInferenceAuth: true}, store)
	if err != nil {
		t.Fatalf("NewAPIAuthMiddleware: %v", err)
	}

	if code := doInferenceRequest(t, mw, plainText); code != http.StatusOK {
		t.Fatalf("request with fingerprinted key: status %d, want 200", code)
	}
}

// A key created through the management endpoint must authenticate against the
// inference middleware without ever touching the legacy path.
func TestHandlerCreatedKeyAuthenticates(t *testing.T) {
	store := newTestAuthStore(t)

	h, err := server.NewHandler(nil, nil, config.AppConfig{}, store, nil)
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}

	req := httptest.NewRequest("POST", "/api/v1/auth/keys",
		strings.NewReader(`{"name":"prod","permission_mode":"allow_all"}`))
	rec := httptest.NewRecorder()
	h.CreateKey()(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("CreateKey: status %d, body %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding CreateKey response: %v", err)
	}
	if resp.Key == "" {
		t.Fatal("CreateKey returned an empty key")
	}

	mw, err := server.NewAPIAuthMiddleware(config.AuthConfig{RequireInferenceAuth: true}, store)
	if err != nil {
		t.Fatalf("NewAPIAuthMiddleware: %v", err)
	}
	if code := doInferenceRequest(t, mw, resp.Key); code != http.StatusOK {
		t.Fatalf("request with handler-created key: status %d, want 200", code)
	}
	if pending, err := store.GetPendingLegacyKeys(context.Background()); err != nil || len(pending) != 0 {
		t.Fatalf("handler-created key landed in legacy set: pending=%d err=%v", len(pending), err)
	}
}
