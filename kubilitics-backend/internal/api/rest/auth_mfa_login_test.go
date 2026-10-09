package rest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/auth/mfa"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
)

// docs/ai/STABILIZATION-PLAN.md Phase 2 item 2: auth.go is the entire
// login/security surface, and its MFA branch (internal/auth/mfa has zero
// test files of its own, and no existing auth_*_test.go file exercises
// Login's MFA path) was completely untested — a regression here is a
// real authentication-bypass or lockout risk, not just a broken feature.

// testMFAEncryptionKey is a valid 32-byte AES-256 key, base64-encoded, as
// mfa.EncryptTOTPSecret/DecryptTOTPSecret require.
func testMFAEncryptionKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate test MFA key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(key)
}

// seedMFAUser creates a user and, if enabled, an encrypted TOTP secret for
// them, returning the plaintext secret (for generating valid test codes).
func seedMFAUser(t *testing.T, repo *repository.SQLiteRepository, encKey string, enabled bool) (*models.User, string) {
	t.Helper()
	password := "Xy9$mK2#pQ7@vN4&wL8*zR5!tB3"
	hashedPassword, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &models.User{
		ID:           "mfa-user-1",
		Username:     "mfauser",
		PasswordHash: hashedPassword,
		Role:         auth.RoleViewer,
	}
	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	plainSecret, _, err := mfa.GenerateTOTPSecret(user.Username)
	if err != nil {
		t.Fatalf("generate TOTP secret: %v", err)
	}
	encrypted, err := mfa.EncryptTOTPSecret(plainSecret, encKey)
	if err != nil {
		t.Fatalf("encrypt TOTP secret: %v", err)
	}
	if err := repo.CreateMFATOTPSecret(context.Background(), &models.MFATOTPSecret{
		UserID:  user.ID,
		Secret:  encrypted,
		Enabled: enabled,
	}); err != nil {
		t.Fatalf("create MFA TOTP secret: %v", err)
	}
	return user, plainSecret
}

func loginRequestFor(username, password, mfaCode string) *http.Request {
	body, _ := json.Marshal(LoginRequest{Username: username, Password: password, MFACode: mfaCode})
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	return req
}

func TestAuthHandler_Login_MFARequired_NoCodeProvided(t *testing.T) {
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	encKey := testMFAEncryptionKey(t)
	cfg := &config.Config{AuthMode: "jwt", AuthJWTSecret: "test-secret-key", MFAEncryptionKey: encKey, MFARequired: true}
	handler := NewAuthHandler(repo, cfg)
	password := "Xy9$mK2#pQ7@vN4&wL8*zR5!tB3"
	_, _ = seedMFAUser(t, repo, encKey, true)

	w := httptest.NewRecorder()
	handler.Login(w, loginRequestFor("mfauser", password, ""))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (MFA-required response, not an error), got %d: %s", w.Code, w.Body.String())
	}
	var resp LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if !resp.MFARequired {
		t.Error("expected MFARequired=true in response")
	}
	if resp.AccessToken != "" {
		t.Error("no access token should be issued before MFA verification")
	}
}

func TestAuthHandler_Login_MFA_ValidTOTPCode_Succeeds(t *testing.T) {
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	encKey := testMFAEncryptionKey(t)
	cfg := &config.Config{AuthMode: "jwt", AuthJWTSecret: "test-secret-key", MFAEncryptionKey: encKey, MFARequired: true}
	handler := NewAuthHandler(repo, cfg)
	password := "Xy9$mK2#pQ7@vN4&wL8*zR5!tB3"
	_, plainSecret := seedMFAUser(t, repo, encKey, true)

	code, err := totp.GenerateCode(plainSecret, time.Now())
	if err != nil {
		t.Fatalf("generate valid TOTP code: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Login(w, loginRequestFor("mfauser", password, code))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with a valid TOTP code, got %d: %s", w.Code, w.Body.String())
	}
	var resp LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected an access token after valid MFA verification")
	}
}

func TestAuthHandler_Login_MFA_InvalidTOTPCode_Rejected(t *testing.T) {
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	encKey := testMFAEncryptionKey(t)
	cfg := &config.Config{AuthMode: "jwt", AuthJWTSecret: "test-secret-key", MFAEncryptionKey: encKey, MFARequired: true}
	handler := NewAuthHandler(repo, cfg)
	password := "Xy9$mK2#pQ7@vN4&wL8*zR5!tB3"
	_, _ = seedMFAUser(t, repo, encKey, true)

	w := httptest.NewRecorder()
	handler.Login(w, loginRequestFor("mfauser", password, "000000"))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for an invalid TOTP code, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuthHandler_Login_MFA_BackupCode_SingleUse(t *testing.T) {
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	encKey := testMFAEncryptionKey(t)
	cfg := &config.Config{AuthMode: "jwt", AuthJWTSecret: "test-secret-key", MFAEncryptionKey: encKey, MFARequired: true}
	handler := NewAuthHandler(repo, cfg)
	password := "Xy9$mK2#pQ7@vN4&wL8*zR5!tB3"
	user, _ := seedMFAUser(t, repo, encKey, true)

	codes, err := mfa.GenerateBackupCodes(1)
	if err != nil {
		t.Fatalf("generate backup codes: %v", err)
	}
	hash, err := mfa.HashBackupCode(codes[0])
	if err != nil {
		t.Fatalf("hash backup code: %v", err)
	}
	if err := repo.CreateMFABackupCodes(context.Background(), user.ID, []string{hash}); err != nil {
		t.Fatalf("store backup code: %v", err)
	}

	// First use: should succeed and consume the code.
	w := httptest.NewRecorder()
	handler.Login(w, loginRequestFor("mfauser", password, codes[0]))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on first backup-code use, got %d: %s", w.Code, w.Body.String())
	}
	var resp LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected an access token after valid backup-code verification")
	}

	// Second use of the same code: must be rejected (single-use).
	w2 := httptest.NewRecorder()
	handler.Login(w2, loginRequestFor("mfauser", password, codes[0]))
	if w2.Code == http.StatusOK {
		t.Error("expected a reused backup code to be rejected, but login succeeded again")
	}
}

func TestAuthHandler_Login_MFA_SecretExistsButNotEnabled(t *testing.T) {
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	encKey := testMFAEncryptionKey(t)
	// Role-level enforcement (MFARequired) is what triggers the check here,
	// independent of whether the user ever finished MFA setup.
	cfg := &config.Config{AuthMode: "jwt", AuthJWTSecret: "test-secret-key", MFAEncryptionKey: encKey, MFARequired: true}
	handler := NewAuthHandler(repo, cfg)
	password := "Xy9$mK2#pQ7@vN4&wL8*zR5!tB3"
	_, plainSecret := seedMFAUser(t, repo, encKey, false) // enabled=false

	code, err := totp.GenerateCode(plainSecret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}

	w := httptest.NewRecorder()
	handler.Login(w, loginRequestFor("mfauser", password, code))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (MFA not set up) when a code is provided for an unenabled secret, got %d: %s", w.Code, w.Body.String())
	}
}
