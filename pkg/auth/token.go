package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const tokenFileName = "token.json"

// TokenData holds the persisted CLI authentication token.
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Roles        []string  `json:"roles"`
}

// IsExpired returns true if the access token has expired (with 30s safety margin).
func (t *TokenData) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}

// tokenFilePath returns ~/.mf/token.json.
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mf")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFileName), nil
}

// SaveToken writes the token data to ~/.mf/token.json.
func SaveToken(data *TokenData) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// LoadToken reads the token data from ~/.mf/token.json.
func LoadToken() (*TokenData, error) {
	path, err := tokenFilePath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in — run `mf login` first")
		}
		return nil, err
	}
	var data TokenData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("corrupted token file: %w", err)
	}
	return &data, nil
}

// ClearToken removes the stored token file.
func ClearToken() error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// LoginWithPassword authenticates via Resource Owner Password Credentials grant.
func LoginWithPassword(issuerURL, clientID, clientSecret, username, password string) (*TokenData, error) {
	tokenURL := strings.TrimSuffix(issuerURL, "/") + "/protocol/openid-connect/token"

	data := url.Values{
		"grant_type":    {"password"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"username":      {username},
		"password":      {password},
		"scope":         {"openid profile email"},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("connecting to Keycloak: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errMsg, _ := result["error_description"].(string)
		if errMsg == "" {
			errMsg, _ = result["error"].(string)
		}
		return nil, fmt.Errorf("authentication failed: %s", errMsg)
	}

	accessToken, _ := result["access_token"].(string)
	refreshToken, _ := result["refresh_token"].(string)
	expiresIn, _ := result["expires_in"].(float64)

	claims, err := parseJWTClaims(accessToken)
	if err != nil {
		return nil, fmt.Errorf("parsing token claims: %w", err)
	}

	return &TokenData{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
		Username:     claims.Username,
		Email:        claims.Email,
		Roles:        claims.Roles,
	}, nil
}

// RefreshToken obtains a new access token using the refresh token.
func RefreshToken(issuerURL, clientID, clientSecret string, token *TokenData) error {
	tokenURL := strings.TrimSuffix(issuerURL, "/") + "/protocol/openid-connect/token"

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {token.RefreshToken},
	}

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return fmt.Errorf("connecting to Keycloak: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed — run `mf login` again")
	}

	accessToken, _ := result["access_token"].(string)
	refreshToken, _ := result["refresh_token"].(string)
	expiresIn, _ := result["expires_in"].(float64)

	claims, err := parseJWTClaims(accessToken)
	if err != nil {
		return fmt.Errorf("parsing token claims: %w", err)
	}

	token.AccessToken = accessToken
	token.RefreshToken = refreshToken
	token.ExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	token.Username = claims.Username
	token.Email = claims.Email
	token.Roles = claims.Roles
	return nil
}

// jwtClaims holds the relevant fields from a Keycloak JWT.
type jwtClaims struct {
	Username string
	Email    string
	Roles    []string
}

// parseJWTClaims extracts claims from a JWT access token without signature verification
// (the token was just received from Keycloak over HTTPS).
func parseJWTClaims(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding JWT payload: %w", err)
	}

	var raw struct {
		PreferredUsername string `json:"preferred_username"`
		Email            string `json:"email"`
		RealmAccess      struct {
			Roles []string `json:"roles"`
		} `json:"realm_access"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}

	// Filter out Keycloak internal roles
	var roles []string
	for _, r := range raw.RealmAccess.Roles {
		if !strings.HasPrefix(r, "default-roles-") && r != "offline_access" && r != "uma_authorization" {
			roles = append(roles, r)
		}
	}

	return &jwtClaims{
		Username: raw.PreferredUsername,
		Email:    raw.Email,
		Roles:    roles,
	}, nil
}
