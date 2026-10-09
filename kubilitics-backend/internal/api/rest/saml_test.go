package rest

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	htmlpkg "html"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	samllib "github.com/crewjam/saml"
	"github.com/crewjam/saml/logger"
	"github.com/crewjam/saml/testsaml"
	samlpkg "github.com/kubilitics/kubilitics-backend/internal/auth/saml"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
)

// ---- Mock IdP built on crewjam/saml's own saml.IdentityProvider ----
//
// This drives the handler under test against a real, locally-run SAML
// Identity Provider (self-signed cert, real XML-DSig signing, real SAML
// AuthnRequest/Response parsing) instead of a live/hosted IdP, exactly as
// the crewjam/saml project's own tests do (see identity_provider_test.go).

// genSelfSignedCert creates a throwaway self-signed RSA certificate for test
// use as either the SP's or a mock IdP's signing credential.
func genSelfSignedCert(t *testing.T, cn string) (*rsa.PrivateKey, *x509.Certificate, []byte, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatalf("generate serial: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return key, cert, certPEM, keyPEM
}

// staticSPProvider always returns the same SP metadata, regardless of the
// requested service provider ID — sufficient for a single-SP test IdP.
type staticSPProvider struct{ ed *samllib.EntityDescriptor }

func (s staticSPProvider) GetServiceProvider(_ *http.Request, _ string) (*samllib.EntityDescriptor, error) {
	return s.ed, nil
}

// staticSessionProvider always returns the same (already-authenticated)
// session — this test IdP never shows a login prompt.
type staticSessionProvider struct{ session *samllib.Session }

func (s staticSessionProvider) GetSession(_ http.ResponseWriter, _ *http.Request, _ *samllib.IdpAuthnRequest) *samllib.Session {
	return s.session
}

// testSAMLEnv wires up a real crewjam/saml IdentityProvider as a mock IdP, a
// real samlpkg.Provider/SAMLHandler as the SP under test, and the plumbing to
// drive a full, signed SSO round trip between them without any network
// dependency on a hosted IdP.
type testSAMLEnv struct {
	idp           *samllib.IdentityProvider
	idpServer     *httptest.Server
	handler       *SAMLHandler
	cfg           *config.Config
	repo          *repository.SQLiteRepository
	samlUserEmail string
}

func newTestSAMLEnv(t *testing.T) *testSAMLEnv {
	t.Helper()

	idpKey, idpCert, _, _ := genSelfSignedCert(t, "test-idp")
	_, spCert, spCertPEM, spKeyPEM := genSelfSignedCert(t, "test-sp")
	_ = spCert

	idp := &samllib.IdentityProvider{
		Key:         idpKey,
		Certificate: idpCert,
		Logger:      logger.DefaultLogger,
		MetadataURL: url.URL{Path: "/metadata"},
		SSOURL:      url.URL{Path: "/sso"},
		LogoutURL:   url.URL{Path: "/slo"},
	}
	idpServer := httptest.NewServer(idp.Handler())
	t.Cleanup(idpServer.Close)

	base, err := url.Parse(idpServer.URL)
	if err != nil {
		t.Fatalf("parse idp server url: %v", err)
	}
	idp.MetadataURL = *base.ResolveReference(&url.URL{Path: "/metadata"})
	idp.SSOURL = *base.ResolveReference(&url.URL{Path: "/sso"})
	idp.LogoutURL = *base.ResolveReference(&url.URL{Path: "/slo"})

	repo := setupSAMLTestRepo(t)

	cfg := &config.Config{
		AuthMode:           "jwt",
		AuthJWTSecret:       "test-secret-key-for-jwt-token-generation",
		SAMLEnabled:        true,
		SAMLIdpMetadataURL: idp.MetadataURL.String(),
		SAMLCertificate:    string(spCertPEM),
		SAMLPrivateKey:     string(spKeyPEM),
	}

	provider, err := samlpkg.NewProvider(context.Background(), cfg, repo)
	if err != nil {
		t.Fatalf("failed to build test SAML provider against mock IdP: %v", err)
	}

	sp := provider.GetServiceProvider()
	spMetadata := sp.ServiceProvider.Metadata()

	samlUserEmail := "samluser@example.com"
	idp.ServiceProviderProvider = staticSPProvider{ed: spMetadata}
	idp.SessionProvider = staticSessionProvider{session: &samllib.Session{
		ID:           "test-session-1",
		CreateTime:   time.Now(),
		ExpireTime:   time.Now().Add(time.Hour),
		Index:        "session-index-1",
		NameID:       samlUserEmail,
		NameIDFormat: "urn:oasis:names:tc:SAML:2.0:nameid-format:emailAddress",
		UserName:     samlUserEmail,
		UserEmail:    samlUserEmail,
		CustomAttributes: []samllib.Attribute{
			{
				Name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
				Values: []samllib.AttributeValue{
					{Type: "xs:string", Value: samlUserEmail},
				},
			},
		},
	}}

	handler := &SAMLHandler{provider: provider, cfg: cfg, repo: repo}

	return &testSAMLEnv{
		idp:           idp,
		idpServer:     idpServer,
		handler:       handler,
		cfg:           cfg,
		repo:          repo,
		samlUserEmail: samlUserEmail,
	}
}

var samlHiddenInputRe = func(name string) *regexp.Regexp {
	return regexp.MustCompile(`name="` + name + `" value="([^"]*)"`)
}

func extractHiddenInput(html, name string) string {
	m := samlHiddenInputRe(name).FindStringSubmatch(html)
	if len(m) != 2 {
		return ""
	}
	// The IdP's response form is rendered with html/template, which
	// HTML-escapes attribute values (e.g. "+" becomes "&#43;"); undo that to
	// recover the original base64 text.
	return htmlpkg.UnescapeString(m[1])
}

// driveLogin calls the handler's Login endpoint and returns the resulting
// redirect URL to the (mock) IdP.
func driveLogin(t *testing.T, env *testSAMLEnv, relayState string) *url.URL {
	t.Helper()
	target := "/auth/saml/login"
	if relayState != "" {
		target += "?relay_state=" + url.QueryEscape(relayState)
	}
	req := httptest.NewRequest("GET", target, nil)
	w := httptest.NewRecorder()
	env.handler.Login(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect from Login, got %d: %s", w.Code, w.Body.String())
	}
	loc := w.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("redirect Location not a valid URL: %v", err)
	}
	return parsed
}

// driveIdPSSO feeds the AuthnRequest redirect URL into the mock IdP and
// returns the raw (base64) SAMLResponse + RelayState it produces, exactly as
// a browser auto-submitting the IdP's response form would deliver them to
// the ACS endpoint.
func driveIdPSSO(t *testing.T, env *testSAMLEnv, authnRequestURL *url.URL) (samlResponse, relayState string) {
	t.Helper()
	req := httptest.NewRequest("GET", authnRequestURL.String(), nil)
	req.RemoteAddr = "203.0.113.5:12345"
	w := httptest.NewRecorder()
	env.idp.ServeSSO(w, req)
	if w.Code != 0 && w.Code != http.StatusOK {
		t.Fatalf("mock IdP ServeSSO failed: %d: %s", w.Code, w.Body.String())
	}
	html := w.Body.String()
	samlResponse = extractHiddenInput(html, "SAMLResponse")
	relayState = extractHiddenInput(html, "RelayState")
	if samlResponse == "" {
		t.Fatalf("mock IdP did not produce a SAMLResponse; body: %s", html)
	}
	return samlResponse, relayState
}

func postToACS(env *testSAMLEnv, samlResponse, relayState string) *httptest.ResponseRecorder {
	form := url.Values{}
	form.Set("SAMLResponse", samlResponse)
	if relayState != "" {
		form.Set("RelayState", relayState)
	}
	req := httptest.NewRequest("POST", "/auth/saml/acs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	env.handler.AssertionConsumerService(w, req)
	return w
}

// setupSAMLTestRepo builds on setupTestRepoForAuth's schema, adding the
// tables the SAML ACS/SLO handlers touch (saml_sessions, refresh token
// families for RevokeAllUserTokens).
func setupSAMLTestRepo(t *testing.T) *repository.SQLiteRepository {
	t.Helper()
	repo := setupTestRepoForAuth(t)
	extra := `
		CREATE TABLE IF NOT EXISTS saml_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			saml_session_index TEXT NOT NULL,
			idp_entity_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS refresh_token_families (
			id TEXT PRIMARY KEY,
			family_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			token_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			revoked_at TIMESTAMP
		);
	`
	if err := repo.RunMigrations(extra); err != nil {
		t.Fatalf("failed to run SAML test migrations: %v", err)
	}
	return repo
}

// ---- Tests ----

func TestSAMLHandler_Login_AuthDisabled(t *testing.T) {
	handler := &SAMLHandler{cfg: &config.Config{AuthMode: "disabled"}}
	req := httptest.NewRequest("GET", "/auth/saml/login", nil)
	w := httptest.NewRecorder()
	handler.Login(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSAMLHandler_ACS_AuthDisabled(t *testing.T) {
	handler := &SAMLHandler{cfg: &config.Config{AuthMode: "disabled"}}
	req := httptest.NewRequest("POST", "/auth/saml/acs", strings.NewReader(""))
	w := httptest.NewRecorder()
	handler.AssertionConsumerService(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

// Open-redirect prevention: an attacker-supplied relay_state pointing at an
// external host must never be reflected back into the redirect flow.
func TestSAMLHandler_Login_RejectsAbsoluteRelayState(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	loc := driveLogin(t, env, "https://evil.example.com/phish")
	relayState := loc.Query().Get("RelayState")
	if strings.Contains(relayState, "evil.example.com") {
		t.Fatalf("expected malicious absolute relay_state to be dropped, got RelayState=%q", relayState)
	}
	if relayState != "/" {
		t.Fatalf("expected relay_state to fall back to '/', got %q", relayState)
	}
}

func TestSAMLHandler_Login_PreservesRelativeRelayState(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	loc := driveLogin(t, env, "/dashboard")
	relayState := loc.Query().Get("RelayState")
	if relayState != "/dashboard" {
		t.Fatalf("expected relative relay_state to be preserved, got %q", relayState)
	}
}

// Regression test for the fix to Login(): the AuthnRequest ID stored for
// later InResponseTo validation must be the *real* ID embedded in the
// AuthnRequest actually sent to the IdP, not an unrelated locally-generated
// value (previously GenerateAuthnRequestID() produced a value that was
// never attached to the outgoing request at all).
func TestSAMLHandler_Login_StoresRealAuthnRequestID(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	loc := driveLogin(t, env, "/after-login")
	rawXML, err := testsaml.ParseRedirectRequest(loc)
	if err != nil {
		t.Fatalf("failed to decode AuthnRequest from redirect: %v", err)
	}
	var authnRequest samllib.AuthnRequest
	if err := xml.Unmarshal(rawXML, &authnRequest); err != nil {
		t.Fatalf("failed to parse AuthnRequest XML: %v", err)
	}
	if authnRequest.ID == "" {
		t.Fatal("expected AuthnRequest to have a non-empty ID")
	}

	stored, ok := env.handler.provider.GetAuthnRequest(authnRequest.ID)
	if !ok {
		t.Fatalf("expected the real AuthnRequest ID %q to be stored for later validation", authnRequest.ID)
	}
	if stored.RelayState != "/after-login" {
		t.Fatalf("expected stored RelayState '/after-login', got %q", stored.RelayState)
	}
}

// Full, legitimate SP-initiated SSO round trip: Login -> (mock) IdP -> ACS.
// Exercises the real r.ParseForm() fix (previously the ACS handler never
// actually read the posted SAMLResponse) together with the InResponseTo
// binding check, end to end.
func TestSAMLHandler_ACS_FullRoundTrip_Success(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	loc := driveLogin(t, env, "/post-login")
	samlResponse, relayState := driveIdPSSO(t, env, loc)
	if relayState != "/post-login" {
		t.Fatalf("expected RelayState '/post-login' to round-trip, got %q", relayState)
	}

	w := postToACS(env, samlResponse, relayState)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from ACS, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse ACS response body: %v", err)
	}
	if resp["access_token"] == "" || resp["access_token"] == nil {
		t.Fatal("expected non-empty access_token")
	}
	if resp["refresh_token"] == "" || resp["refresh_token"] == nil {
		t.Fatal("expected non-empty refresh_token")
	}

	expectedUserID := "saml-" + env.samlUserEmail
	user, err := env.repo.GetUserByID(context.Background(), expectedUserID)
	if err != nil || user == nil {
		t.Fatalf("expected SAML user %q to be created, err=%v user=%v", expectedUserID, err, user)
	}
	if user.Username != env.samlUserEmail {
		t.Fatalf("expected username %q, got %q", env.samlUserEmail, user.Username)
	}
}

// Security regression test: the ACS endpoint must reject SAML responses
// that don't carry an InResponseTo matching a request this SP actually
// issued. Before the fix, the handler called
// sp.ServiceProvider.ParseResponse(r, []string{""}) — which, since
// AllowIDPInitiated defaults to false, only ever matched a response whose
// InResponseTo was *empty*, i.e. precisely an unsolicited/IdP-initiated
// response. This drives exactly that case through the mock IdP's
// ServeIDPInitiated path (no prior AuthnRequest at all) to prove such a
// response is now rejected rather than accepted.
func TestSAMLHandler_ACS_RejectsUnsolicitedIdPInitiatedResponse(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	req := httptest.NewRequest("GET", "/idp-initiated", nil)
	req.RemoteAddr = "203.0.113.9:12345"
	w := httptest.NewRecorder()
	env.idp.ServeIDPInitiated(w, req, "unused-sp-id", "/")
	html := w.Body.String()
	samlResponse := extractHiddenInput(html, "SAMLResponse")
	if samlResponse == "" {
		t.Fatalf("mock IdP did not produce an IdP-initiated SAMLResponse; body: %s", html)
	}

	acsResp := postToACS(env, samlResponse, "/")
	if acsResp.Code != http.StatusBadRequest {
		t.Fatalf("expected unsolicited IdP-initiated response to be rejected with 400, got %d: %s", acsResp.Code, acsResp.Body.String())
	}
	if !strings.Contains(acsResp.Body.String(), "InResponseTo") && !strings.Contains(acsResp.Body.String(), "unsolicited") {
		t.Fatalf("expected rejection reason to reference missing InResponseTo, got: %s", acsResp.Body.String())
	}
}

// Security regression test: a SAML response whose InResponseTo references a
// request ID this SP never issued (forged/stale) must be rejected, even
// though it carries a non-empty InResponseTo.
func TestSAMLHandler_ACS_RejectsUnknownRequestID(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	loc := driveLogin(t, env, "/x")
	samlResponse, relayState := driveIdPSSO(t, env, loc)

	decoded, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		t.Fatalf("failed to decode SAMLResponse: %v", err)
	}
	// Extract the real (legitimate) InResponseTo value and substitute a
	// value that was never stored via StoreAuthnRequest. This is caught by
	// our own pre-check (before signature validation would even run), so
	// the resulting XML need not remain well-signed.
	realID := extractAttr(string(decoded), "InResponseTo")
	if realID == "" {
		t.Fatal("expected to find a real InResponseTo value in the mock IdP's response")
	}
	forged := bytes.ReplaceAll(decoded, []byte(realID), []byte("forged-request-id-never-issued"))
	forgedResponse := base64.StdEncoding.EncodeToString(forged)

	w := postToACS(env, forgedResponse, relayState)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected forged InResponseTo to be rejected with 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "does not match a known authentication request") {
		t.Fatalf("expected 'does not match a known authentication request' error, got: %s", w.Body.String())
	}
}

func TestSAMLHandler_ACS_MissingSAMLResponse(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	w := postToACS(env, "", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing SAMLResponse, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSAMLHandler_ACS_MalformedBase64(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	w := postToACS(env, "not-valid-base64!!!", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed base64, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSAMLHandler_Metadata_ReturnsXML(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	req := httptest.NewRequest("GET", "/auth/saml/metadata", nil)
	w := httptest.NewRecorder()
	env.handler.Metadata(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/xml" {
		t.Fatalf("expected Content-Type application/xml, got %q", ct)
	}
	if !strings.Contains(w.Body.String(), "EntityDescriptor") {
		t.Fatalf("expected SP metadata XML to contain EntityDescriptor, got: %s", w.Body.String())
	}
}

func TestSAMLHandler_SingleLogout_UnauthenticatedGET(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	req := httptest.NewRequest("GET", "/auth/saml/slo", nil)
	w := httptest.NewRecorder()
	env.handler.SingleLogout(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated logout, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSAMLHandler_SingleLogout_MalformedPOSTBody(t *testing.T) {
	env := newTestSAMLEnv(t)
	defer env.repo.Close()

	req := httptest.NewRequest("POST", "/auth/saml/slo", strings.NewReader("not-xml"))
	w := httptest.NewRecorder()
	env.handler.SingleLogout(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed logout request body, got %d: %s", w.Code, w.Body.String())
	}
}

// extractAttr does a minimal, dependency-free extraction of an XML
// attribute's value for test assertions/tampering (not for production use).
func extractAttr(xmlStr, attr string) string {
	marker := attr + `="`
	idx := strings.Index(xmlStr, marker)
	if idx < 0 {
		return ""
	}
	start := idx + len(marker)
	end := strings.Index(xmlStr[start:], `"`)
	if end < 0 {
		return ""
	}
	return xmlStr[start : start+end]
}
