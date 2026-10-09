package rest

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/mux"
	"github.com/kubilitics/kubilitics-backend/internal/auth"
	"github.com/kubilitics/kubilitics-backend/internal/auth/oidc"
	"github.com/kubilitics/kubilitics-backend/internal/config"
	"github.com/kubilitics/kubilitics-backend/internal/repository"
)

// OIDCHandler handles OIDC authentication flows
type OIDCHandler struct {
	provider *oidc.Provider
	cfg      *config.Config
	repo     *repository.SQLiteRepository

	// nonceMu guards nonceStore. The OIDC nonce is tracked independently from
	// the provider's internal state store so that the ID token's "nonce"
	// claim can be checked against the value this SP actually sent, which
	// prevents a captured/replayed ID token (signed by the real IdP, for a
	// different login attempt) from being accepted here.
	// See: https://openid.net/specs/openid-connect-core-1_0.html#NonceNotes
	nonceMu    sync.Mutex
	nonceStore map[string]string // state -> nonce
}

// NewOIDCHandler creates a new OIDC handler
func NewOIDCHandler(cfg *config.Config, repo *repository.SQLiteRepository) (*OIDCHandler, error) {
	if !cfg.OIDCEnabled {
		return nil, nil // OIDC not enabled
	}

	provider, err := oidc.NewProvider(context.Background(), cfg, repo)
	if err != nil {
		return nil, err
	}

	return &OIDCHandler{
		provider:   provider,
		cfg:        cfg,
		repo:       repo,
		nonceStore: make(map[string]string),
	}, nil
}

// generateNonce returns a random, URL-safe nonce value used for OIDC ID
// token replay protection (see NonceNotes in the OIDC Core spec).
func generateNonce() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// RegisterRoutes registers OIDC routes
func (h *OIDCHandler) RegisterRoutes(router *mux.Router) {
	if h == nil {
		return // OIDC not enabled
	}
	router.HandleFunc("/auth/oidc/login", h.Login).Methods("GET")
	router.HandleFunc("/auth/oidc/callback", h.Callback).Methods("GET")
}

// Login initiates OIDC login flow
func (h *OIDCHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.cfg.AuthMode == "disabled" {
		respondError(w, http.StatusBadRequest, "Authentication is disabled")
		return
	}

	state, err := h.provider.GenerateState()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate state")
		return
	}

	nonce, err := generateNonce()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate nonce")
		return
	}
	h.nonceMu.Lock()
	h.nonceStore[state] = nonce
	h.nonceMu.Unlock()

	authURL := h.provider.AuthCodeURL(state)
	// Attach the nonce to the authorization request so the IdP echoes it back
	// in the ID token; verified in Callback below.
	if parsed, parseErr := url.Parse(authURL); parseErr == nil {
		q := parsed.Query()
		q.Set("nonce", nonce)
		parsed.RawQuery = q.Encode()
		authURL = parsed.String()
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback handles OIDC callback
func (h *OIDCHandler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.cfg.AuthMode == "disabled" {
		respondError(w, http.StatusBadRequest, "Authentication is disabled")
		return
	}

	// Validate state
	state := r.URL.Query().Get("state")

	// Pop the nonce we stored for this state before consuming the state via
	// ValidateState (which deletes it from the provider's store), so a given
	// state/nonce pair can only ever be used once.
	h.nonceMu.Lock()
	expectedNonce, haveNonce := h.nonceStore[state]
	delete(h.nonceStore, state)
	h.nonceMu.Unlock()

	if !h.provider.ValidateState(state) {
		respondError(w, http.StatusBadRequest, "Invalid state parameter")
		return
	}

	// Check for error from OIDC provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		respondError(w, http.StatusBadRequest, "OIDC error: "+errParam)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		respondError(w, http.StatusBadRequest, "Authorization code not provided")
		return
	}

	ctx := r.Context()

	// Exchange code for tokens
	token, err := h.provider.ExchangeCode(ctx, code)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to exchange code: "+err.Error())
		return
	}

	// Verify ID token
	idToken, claims, err := h.provider.VerifyIDToken(ctx, token)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to verify ID token: "+err.Error())
		return
	}
	
	// Verify the nonce echoed back in the ID token matches the one we sent,
	// preventing a previously-issued, validly-signed ID token from a
	// different login attempt from being replayed against this callback.
	if !haveNonce {
		respondError(w, http.StatusBadRequest, "Missing nonce for session")
		return
	}
	actualNonce, _ := claims["nonce"].(string)
	if actualNonce == "" || actualNonce != expectedNonce {
		respondError(w, http.StatusBadRequest, "Invalid nonce: possible ID token replay")
		return
	}

	// Extract subject for user ID
	sub := idToken.Subject
	if sub == "" {
		if s, ok := claims["sub"].(string); ok {
			sub = s
		}
	}

	// Get user info
	userInfo, err := h.provider.GetUserInfo(ctx, token)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user info: "+err.Error())
		return
	}

	// Map OIDC user to kubilitics user
	kubiliticsUser, err := h.provider.MapOIDCUserToKubiliticsUser(ctx, claims, userInfo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to map user: "+err.Error())
		return
	}

	// Ensure user ID uses sub
	if sub != "" {
		kubiliticsUser.ID = fmt.Sprintf("oidc-%s", sub)
	}
	
	// Check if user exists by username or ID
	existingUser, err := h.repo.GetUserByUsername(ctx, kubiliticsUser.Username)
	if err != nil || existingUser == nil {
		// Try by ID (for OIDC users)
		existingUser, err = h.repo.GetUserByID(ctx, kubiliticsUser.ID)
		if err != nil || existingUser == nil {
			// Create OIDC user (no password)
			if err := h.repo.CreateUser(ctx, kubiliticsUser); err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to create user: "+err.Error())
				return
			}
			existingUser = kubiliticsUser
		}
	} else {
		// Update role if changed
		if existingUser.Role != kubiliticsUser.Role {
			_ = h.repo.UpdateUserRole(ctx, existingUser.ID, kubiliticsUser.Role)
			existingUser.Role = kubiliticsUser.Role
		}
	}

	// Issue kubilitics JWT tokens
	accessToken, err := auth.IssueAccessToken(h.cfg.AuthJWTSecret, existingUser.ID, existingUser.Username, existingUser.Role)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to issue token")
		return
	}
	refreshToken, err := auth.IssueRefreshToken(h.cfg.AuthJWTSecret, existingUser.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to issue token")
		return
	}

	// Return tokens (frontend will handle redirect)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(auth.AccessTokenExpiry.Seconds()),
		"token_type":    "Bearer",
	})
}
