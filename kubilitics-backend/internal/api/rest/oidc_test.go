package rest

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	oidcpkg "github.com/kubilitics/kubilitics-backend/internal/auth/oidc"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/models"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
)

// ---- Minimal RS256 JWT + mock OIDC IdP helpers (no live IdP; no go-jose/jwt dependency) ----

type testOIDCIdP struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	kid    string
	// nextTokenResponse, when set, lets a test override what the mock
	// token endpoint returns (e.g. to simulate a failure).
	tokenHandler func(w http.ResponseWriter, r *http.Request)
}

func base64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func signRS256JWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]interface{}) string {
	t.Helper()
	header := map[string]interface{}{"alg": "RS256", "typ": "JWT", "kid": kid}
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	cb, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	signingInput := base64url(hb) + "." + base64url(cb)
	hashed := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return signingInput + "." + base64url(sig)
}

// newTestOIDCIdP spins up an httptest server implementing just enough of the
// OIDC discovery + token + userinfo + jwks surface for
// github.com/coreos/go-oidc/v3 to drive a full, real Provider through
// discovery, ID token verification (real RS256 signature check against the
// published JWKS) and UserInfo — without any network dependency on a real
// IdP. This lets the handler tests exercise the actual provider code path
// end-to-end instead of stubbing it out.
func newTestOIDCIdP(t *testing.T) *testOIDCIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	idp := &testOIDCIdP{key: key, kid: "test-kid-1"}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		doc := map[string]interface{}{
			"issuer":                                idp.server.URL,
			"authorization_endpoint":                idp.server.URL + "/authorize",
			"token_endpoint":                         idp.server.URL + "/token",
			"userinfo_endpoint":                      idp.server.URL + "/userinfo",
			"jwks_uri":                                idp.server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		nBytes := idp.key.PublicKey.N.Bytes()
		eBytes := big.NewInt(int64(idp.key.PublicKey.E)).Bytes()
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"use": "sig",
					"kid": idp.kid,
					"alg": "RS256",
					"n":   base64url(nBytes),
					"e":   base64url(eBytes),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if idp.tokenHandler != nil {
			idp.tokenHandler(w, r)
			return
		}
		idp.defaultTokenHandler(t, w, r)
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz != "Bearer test-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sub":   "user-sub-123",
			"email": "ssouser@example.com",
		})
	})

	idp.server = httptest.NewServer(mux)
	t.Cleanup(idp.server.Close)
	return idp
}

// defaultTokenHandler mints a real, signed ID token whose "nonce" claim
// echoes back the nonce supplied on the /authorize redirect via the "code"
// (we smuggle the test's desired nonce through the authorization code value
// itself, since this mock never actually drives a browser through
// /authorize).
func (idp *testOIDCIdP) defaultTokenHandler(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	code := r.PostFormValue("code")
	if code == "bad-code" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		return
	}

	// code is of the form "<nonce>" or "<nonce>|<sub>|<role-group>" set up by
	// the test via encodeTestAuthCode.
	nonce, sub, group := decodeTestAuthCode(code)

	claims := map[string]interface{}{
		"iss":   idp.server.URL,
		"sub":   sub,
		"aud":   "test-client-id",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"nonce": nonce,
		"email": "ssouser@example.com",
	}
	if group != "" {
		claims["groups"] = []string{group}
	}
	idToken := signRS256JWT(t, idp.key, idp.kid, claims)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": "test-access-token",
		"token_type":   "Bearer",
		"id_token":     idToken,
		"expires_in":   3600,
	})
}

// encodeTestAuthCode / decodeTestAuthCode let the test control what subject
// and nonce the mock token endpoint embeds in the minted ID token, keyed off
// the authorization "code" value the handler passes through unmodified from
// the request.
func encodeTestAuthCode(nonce, sub, group string) string {
	return nonce + "|" + sub + "|" + group
}

func decodeTestAuthCode(code string) (nonce, sub, group string) {
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(code); i++ {
		if code[i] == '|' {
			parts = append(parts, code[start:i])
			start = i + 1
		}
	}
	parts = append(parts, code[start:])
	for len(parts) < 3 {
		parts = append(parts, "")
	}
	return parts[0], parts[1], parts[2]
}

func newTestOIDCHandler(t *testing.T, idp *testOIDCIdP, repo *repository.SQLiteRepository, cfgOverrides func(*config.Config)) *OIDCHandler {
	t.Helper()
	cfg := &config.Config{
		AuthMode:         "jwt",
		AuthJWTSecret:    "test-secret-key-for-jwt-token-generation",
		OIDCEnabled:      true,
		OIDCIssuerURL:    idp.server.URL,
		OIDCClientID:     "test-client-id",
		OIDCClientSecret: "test-client-secret",
		OIDCRedirectURL:  "http://localhost:8190/api/v1/auth/oidc/callback",
		OIDCScopes:       "openid,profile,email",
		OIDCGroupClaim:   "groups",
		OIDCRoleMapping:  `{"admins":"admin"}`,
	}
	if cfgOverrides != nil {
		cfgOverrides(cfg)
	}

	provider, err := oidcpkg.NewProvider(context.Background(), cfg, repo)
	if err != nil {
		t.Fatalf("failed to build test OIDC provider against mock IdP: %v", err)
	}

	return &OIDCHandler{
		provider:   provider,
		cfg:        cfg,
		repo:       repo,
		nonceStore: make(map[string]string),
	}
}

// ---- Tests ----

func TestOIDCHandler_Login_AuthDisabled(t *testing.T) {
	// Guard clause must short-circuit before ever touching h.provider.
	handler := &OIDCHandler{cfg: &config.Config{AuthMode: "disabled"}}

	req := httptest.NewRequest("GET", "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOIDCHandler_Callback_AuthDisabled(t *testing.T) {
	handler := &OIDCHandler{cfg: &config.Config{AuthMode: "disabled"}}

	req := httptest.NewRequest("GET", "/auth/oidc/callback?state=x&code=y", nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOIDCHandler_Login_Success_SetsStateAndNonce(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	req := httptest.NewRequest("GET", "/auth/oidc/login", nil)
	w := httptest.NewRecorder()
	handler.Login(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("redirect location not a valid URL: %v", err)
	}
	state := parsed.Query().Get("state")
	nonce := parsed.Query().Get("nonce")
	if state == "" {
		t.Fatal("expected state query param on redirect URL")
	}
	if nonce == "" {
		t.Fatal("expected nonce query param on redirect URL (ID token replay protection)")
	}
	handler.nonceMu.Lock()
	stored, ok := handler.nonceStore[state]
	handler.nonceMu.Unlock()
	if !ok || stored != nonce {
		t.Fatalf("expected nonce %q stored against state %q, got stored=%q ok=%v", nonce, state, stored, ok)
	}
}

func TestOIDCHandler_Callback_InvalidState(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	req := httptest.NewRequest("GET", "/auth/oidc/callback?state=never-issued&code=abc", nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown state, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOIDCHandler_Callback_StateIsSingleUse(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	state, err := handler.provider.GenerateState()
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	handler.nonceStore[state] = "some-nonce"

	// First consumption fails validation for an unrelated reason (missing
	// code) but MUST still consume the state.
	req1 := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state, nil)
	w1 := httptest.NewRecorder()
	handler.Callback(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (missing code) on first use, got %d: %s", w1.Code, w1.Body.String())
	}

	// Second attempt with the same state must be rejected — replay/reuse of
	// a consumed state parameter must not be possible.
	req2 := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&code=whatever", nil)
	w2 := httptest.NewRecorder()
	handler.Callback(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on replayed state, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestOIDCHandler_Callback_ErrorParam(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	state, _ := handler.provider.GenerateState()
	handler.nonceStore[state] = "n"

	req := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&error=access_denied", nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for IdP error param, got %d: %s", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), "access_denied") {
		t.Fatalf("expected error body to mention access_denied, got %s", w.Body.String())
	}
}

func TestOIDCHandler_Callback_MissingCode(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	state, _ := handler.provider.GenerateState()
	handler.nonceStore[state] = "n"

	req := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state, nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing code, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOIDCHandler_Callback_TokenExchangeFailure(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	state, _ := handler.provider.GenerateState()
	handler.nonceStore[state] = "n"

	req := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&code=bad-code", nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for token exchange failure, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOIDCHandler_Callback_NonceMismatchRejected(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	state, _ := handler.provider.GenerateState()
	// Handler expects nonce "expected-nonce" but the mock IdP's token
	// endpoint will mint an ID token with the nonce we smuggle through the
	// auth code: "attacker-nonce". This simulates an attacker replaying a
	// validly-signed ID token issued for a *different* login attempt.
	handler.nonceStore[state] = "expected-nonce"

	code := encodeTestAuthCode("attacker-nonce", "oidc-attacker-sub", "")
	req := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&code="+code, nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for nonce mismatch, got %d: %s", w.Code, w.Body.String())
	}
	if !contains(w.Body.String(), "nonce") {
		t.Fatalf("expected error to mention nonce, got %s", w.Body.String())
	}
}

func TestOIDCHandler_Callback_Success_CreatesUserAndIssuesTokens(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	state, _ := handler.provider.GenerateState()
	nonce := "matching-nonce"
	handler.nonceStore[state] = nonce

	code := encodeTestAuthCode(nonce, "user-sub-123", "")
	req := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&code="+code, nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Fatal("expected non-empty access_token in response")
	}
	if resp["refresh_token"] == "" || resp["refresh_token"] == nil {
		t.Fatal("expected non-empty refresh_token in response")
	}

	user, err := repo.GetUserByID(context.Background(), "oidc-user-sub-123")
	if err != nil || user == nil {
		t.Fatalf("expected OIDC user to be created with ID oidc-user-sub-123, err=%v user=%v", err, user)
	}
	if user.Username != "ssouser@example.com" {
		t.Fatalf("expected username from userinfo email, got %q", user.Username)
	}
	if user.Role != "viewer" {
		t.Fatalf("expected default role 'viewer' (no matching group), got %q", user.Role)
	}

	// The same state/nonce pair must not be usable a second time.
	req2 := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&code="+code, nil)
	w2 := httptest.NewRecorder()
	handler.Callback(w2, req2)
	if w2.Code == http.StatusOK {
		t.Fatalf("expected replay of consumed state to fail, got 200: %s", w2.Body.String())
	}
}

func TestOIDCHandler_Callback_RoleMappingUpdatesExistingUser(t *testing.T) {
	idp := newTestOIDCIdP(t)
	repo := setupTestRepoForAuth(t)
	defer repo.Close()
	handler := newTestOIDCHandler(t, idp, repo, nil)

	// Pre-create the user as a viewer so we can assert the role is promoted
	// to admin via the OIDC group -> role mapping on this callback.
	ctx := context.Background()
	if err := repo.CreateUser(ctx, &models.User{
		ID:       "oidc-user-sub-123",
		Username: "ssouser@example.com",
		Role:     "viewer",
	}); err != nil {
		t.Fatalf("failed to seed existing user: %v", err)
	}

	state, _ := handler.provider.GenerateState()
	nonce := "role-nonce"
	handler.nonceStore[state] = nonce

	// group "admins" maps to role "admin" per OIDCRoleMapping in newTestOIDCHandler.
	code := encodeTestAuthCode(nonce, "user-sub-123", "admins")
	req := httptest.NewRequest("GET", "/auth/oidc/callback?state="+state+"&code="+code, nil)
	w := httptest.NewRecorder()
	handler.Callback(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	user, err := repo.GetUserByID(ctx, "oidc-user-sub-123")
	if err != nil || user == nil {
		t.Fatalf("expected existing user to be found, err=%v user=%v", err, user)
	}
	if user.Role != "admin" {
		t.Fatalf("expected role promoted to 'admin' via group mapping, got %q", user.Role)
	}
}
