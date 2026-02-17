package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCAuthenticator handles the OIDC authorization code flow with Keycloak.
type OIDCAuthenticator struct {
	provider     *oidc.Provider
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	sessions     *SessionManager
	orgStore     *OrgStore
	issuerURL    string
	clientID     string
}

// NewOIDCAuthenticator creates an authenticator from the auth config.
// It performs OIDC discovery against the issuer URL.
func NewOIDCAuthenticator(ctx context.Context, cfg AuthConfig, sessions *SessionManager, orgStore *OrgStore) (*OIDCAuthenticator, error) {
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})

	return &OIDCAuthenticator{
		provider:     provider,
		oauth2Config: oauth2Config,
		verifier:     verifier,
		sessions:     sessions,
		orgStore:     orgStore,
		issuerURL:    cfg.IssuerURL,
		clientID:     cfg.ClientID,
	}, nil
}

// LoginHandler initiates the OIDC authorization code flow.
func (a *OIDCAuthenticator) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := randomString(32)
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	verifier := oauth2.GenerateVerifier()

	if err := a.sessions.SetOAuthState(w, r, state, verifier); err != nil {
		http.Error(w, "Failed to save state", http.StatusInternalServerError)
		return
	}

	authURL := a.oauth2Config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, authURL, http.StatusFound)
}

// keycloakClaims represents the JWT claims returned by Keycloak.
type keycloakClaims struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// CallbackHandler handles the OIDC callback after Keycloak authentication.
func (a *OIDCAuthenticator) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify state
	expectedState, pkceVerifier := a.sessions.GetOAuthState(r)
	if expectedState == "" || r.URL.Query().Get("state") != expectedState {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Check for errors from the provider
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		desc := r.URL.Query().Get("error_description")
		http.Error(w, fmt.Sprintf("Authentication error: %s — %s", errMsg, desc), http.StatusUnauthorized)
		return
	}

	// Exchange authorization code for tokens
	code := r.URL.Query().Get("code")
	oauth2Token, err := a.oauth2Config.Exchange(ctx, code, oauth2.VerifierOption(pkceVerifier))
	if err != nil {
		http.Error(w, "Token exchange failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No id_token in response", http.StatusInternalServerError)
		return
	}

	idToken, err := a.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		http.Error(w, "ID token verification failed", http.StatusUnauthorized)
		return
	}

	// Extract claims
	var claims keycloakClaims
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, "Failed to parse claims", http.StatusInternalServerError)
		return
	}

	// Determine display name
	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}
	if name == "" {
		name = claims.Email
	}

	// Filter out Keycloak default roles
	var roles []string
	for _, role := range claims.RealmAccess.Roles {
		if !strings.HasPrefix(role, "default-roles-") && role != "offline_access" && role != "uma_authorization" {
			roles = append(roles, role)
		}
	}

	// Ensure default org exists for this user
	org, err := a.orgStore.EnsureDefaultOrg(ctx, claims.Email, name)
	if err != nil {
		http.Error(w, "Failed to initialize organization: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create session
	user := &UserSession{
		UserID:        claims.Sub,
		Email:         claims.Email,
		Name:          name,
		Roles:         roles,
		ActiveOrgID:   org.ID,
		Authenticated: true,
	}

	if err := a.sessions.SetUser(w, r, user); err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// LogoutHandler clears the local session and redirects to Keycloak's logout endpoint.
func (a *OIDCAuthenticator) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	_ = a.sessions.Clear(w, r)

	// Redirect to Keycloak end-session endpoint
	logoutURL := fmt.Sprintf("%s/protocol/openid-connect/logout?client_id=%s&post_logout_redirect_uri=%s",
		a.issuerURL,
		a.clientID,
		a.oauth2Config.RedirectURL[:strings.LastIndex(a.oauth2Config.RedirectURL, "/")]+"/login",
	)
	http.Redirect(w, r, logoutURL, http.StatusFound)
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
