package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// KeycloakConfigurator sets up the Keycloak realm, client, and roles via REST API.
type KeycloakConfigurator struct {
	baseURL    string
	httpClient *http.Client
}

// NewKeycloakConfigurator creates a configurator for the given Keycloak base URL.
func NewKeycloakConfigurator(baseURL string) *KeycloakConfigurator {
	return &KeycloakConfigurator{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// getAdminToken obtains an admin access token.
func (kc *KeycloakConfigurator) getAdminToken(adminUser, adminPass string) (string, error) {
	data := url.Values{
		"grant_type": {"password"},
		"client_id":  {"admin-cli"},
		"username":   {adminUser},
		"password":   {adminPass},
	}

	resp, err := kc.httpClient.PostForm(kc.baseURL+"/realms/master/protocol/openid-connect/token", data)
	if err != nil {
		return "", fmt.Errorf("requesting admin token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("admin login failed (%d): %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

// ConfigureRealm creates the microfoundry realm, client, and roles.
func (kc *KeycloakConfigurator) ConfigureRealm(adminUser, adminPass, clientSecret, redirectURI string) error {
	token, err := kc.getAdminToken(adminUser, adminPass)
	if err != nil {
		return err
	}

	// Create realm
	realm := map[string]any{
		"realm":             "microfoundry",
		"enabled":           true,
		"registrationAllowed": true,
		"loginWithEmailAllowed": true,
		"duplicateEmailsAllowed": false,
	}
	if err := kc.post(token, "/admin/realms", realm); err != nil {
		if !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("creating realm: %w", err)
		}
		// Realm already exists — continue
	}

	// Create client
	client := map[string]any{
		"clientId":                  "mf-admin",
		"name":                      "MicroFoundry Admin",
		"enabled":                   true,
		"protocol":                  "openid-connect",
		"publicClient":              false,
		"secret":                    clientSecret,
		"standardFlowEnabled":       true,
		"directAccessGrantsEnabled": false,
		"redirectUris":              []string{redirectURI},
		"webOrigins":                []string{"*"},
		"attributes": map[string]string{
			"pkce.code.challenge.method": "S256",
		},
	}
	if err := kc.post(token, "/admin/realms/microfoundry/clients", client); err != nil {
		if !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("creating client: %w", err)
		}
	}

	// Create realm roles
	roles := []string{"platform-admin", "org-admin", "org-member", "viewer"}
	for _, role := range roles {
		roleBody := map[string]any{
			"name": role,
		}
		if err := kc.post(token, "/admin/realms/microfoundry/roles", roleBody); err != nil {
			if !strings.Contains(err.Error(), "409") {
				return fmt.Errorf("creating role %q: %w", role, err)
			}
		}
	}

	return nil
}

// AddIdentityProvider configures a social identity provider in the microfoundry realm.
func (kc *KeycloakConfigurator) AddIdentityProvider(adminUser, adminPass, providerID, clientID, clientSecret string) error {
	token, err := kc.getAdminToken(adminUser, adminPass)
	if err != nil {
		return err
	}

	idp := map[string]any{
		"alias":                     providerID,
		"providerId":               providerID,
		"enabled":                  true,
		"trustEmail":               true,
		"storeToken":               false,
		"firstBrokerLoginFlowAlias": "first broker login",
		"config": map[string]string{
			"clientId":     clientID,
			"clientSecret": clientSecret,
		},
	}

	if err := kc.post(token, "/admin/realms/microfoundry/identity-providers/instances", idp); err != nil {
		if !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("adding identity provider %q: %w", providerID, err)
		}
	}

	return nil
}

func (kc *KeycloakConfigurator) post(token, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", kc.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("keycloak API error (%d): %s", resp.StatusCode, respBody)
	}
	return nil
}
