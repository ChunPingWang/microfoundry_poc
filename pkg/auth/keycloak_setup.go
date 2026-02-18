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

	// Derive wildcard redirect URI for logout flows (e.g. /auth/callback → /*)
	baseURI := redirectURI
	if idx := strings.LastIndex(redirectURI, "/auth/"); idx > 0 {
		baseURI = redirectURI[:idx] + "/*"
	} else if idx := strings.LastIndex(redirectURI, "/"); idx > 0 {
		baseURI = redirectURI[:idx] + "/*"
	}

	// Create client (with service account for admin API access)
	client := map[string]any{
		"clientId":                  "mf-admin",
		"name":                      "MicroFoundry Admin",
		"enabled":                   true,
		"protocol":                  "openid-connect",
		"publicClient":              false,
		"secret":                    clientSecret,
		"standardFlowEnabled":       true,
		"directAccessGrantsEnabled": true,
		"serviceAccountsEnabled":    true,
		"redirectUris":              []string{baseURI},
		"webOrigins":                []string{"*"},
		"attributes": map[string]string{
			"pkce.code.challenge.method":  "S256",
			"post.logout.redirect.uris":  baseURI,
		},
	}
	if err := kc.post(token, "/admin/realms/microfoundry/clients", client); err != nil {
		if !strings.Contains(err.Error(), "409") {
			return fmt.Errorf("creating client: %w", err)
		}
	}

	// Assign realm-management roles to the service account for admin API access
	kc.assignServiceAccountRoles(token, clientSecret)

	// Create realm roles
	roles := []string{"platform-admin", "workspace-admin", "org-admin", "org-member", "viewer"}
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

	// Enable realm_access.roles in ID token (required for OIDC role detection)
	kc.enableRealmRolesInIDToken(token)

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

// assignServiceAccountRoles grants the mf-admin service account the realm-management roles
// needed for user CRUD via the Keycloak Admin REST API.
func (kc *KeycloakConfigurator) assignServiceAccountRoles(token, clientSecret string) {
	// 1. Find the mf-admin client's internal ID
	clientInternalID := kc.getJSON(token, "/admin/realms/microfoundry/clients?clientId=mf-admin", "id")
	if clientInternalID == "" {
		return
	}

	// 2. Get the service account user
	saUserID := kc.getJSON(token, fmt.Sprintf("/admin/realms/microfoundry/clients/%s/service-account-user", clientInternalID), "id")
	if saUserID == "" {
		return
	}

	// 3. Find the realm-management client's internal ID
	realmMgmtID := kc.getJSON(token, "/admin/realms/microfoundry/clients?clientId=realm-management", "id")
	if realmMgmtID == "" {
		return
	}

	// 4. Get available realm-management client roles
	rolesURL := fmt.Sprintf("/admin/realms/microfoundry/clients/%s/roles", realmMgmtID)
	roles := kc.getJSONArray(token, rolesURL)

	// 5. Assign all realm-management roles to the service account
	if len(roles) > 0 {
		assignURL := fmt.Sprintf("/admin/realms/microfoundry/users/%s/role-mappings/clients/%s", saUserID, realmMgmtID)
		_ = kc.post(token, assignURL, roles)
	}
}

// getJSON fetches a JSON endpoint and returns the value of a field from the first array element or object.
func (kc *KeycloakConfigurator) getJSON(token, path, field string) string {
	req, err := http.NewRequest("GET", kc.baseURL+path, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kc.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Try as array first
	var arr []map[string]any
	if json.Unmarshal(body, &arr) == nil && len(arr) > 0 {
		if v, ok := arr[0][field].(string); ok {
			return v
		}
	}

	// Try as object
	var obj map[string]any
	if json.Unmarshal(body, &obj) == nil {
		if v, ok := obj[field].(string); ok {
			return v
		}
	}

	return ""
}

// getJSONArray fetches a JSON array endpoint.
func (kc *KeycloakConfigurator) getJSONArray(token, path string) []map[string]any {
	req, err := http.NewRequest("GET", kc.baseURL+path, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := kc.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()

	var arr []map[string]any
	json.NewDecoder(resp.Body).Decode(&arr)
	return arr
}

// enableRealmRolesInIDToken finds the "roles" client scope's "realm roles" mapper
// and sets id.token.claim=true so realm_access.roles appears in the ID token.
func (kc *KeycloakConfigurator) enableRealmRolesInIDToken(token string) {
	// Find the "roles" client scope
	scopes := kc.getJSONArray(token, "/admin/realms/microfoundry/client-scopes")
	var scopeID string
	for _, s := range scopes {
		if name, _ := s["name"].(string); name == "roles" {
			if id, _ := s["id"].(string); id != "" {
				scopeID = id
				break
			}
		}
	}
	if scopeID == "" {
		return
	}

	// Find the "realm roles" mapper within the scope
	mappers := kc.getJSONArray(token, fmt.Sprintf("/admin/realms/microfoundry/client-scopes/%s/protocol-mappers/models", scopeID))
	for _, m := range mappers {
		name, _ := m["name"].(string)
		if name != "realm roles" {
			continue
		}
		mapperID, _ := m["id"].(string)
		if mapperID == "" {
			continue
		}
		// Update config to include id.token.claim
		cfg, _ := m["config"].(map[string]any)
		if cfg == nil {
			cfg = map[string]any{}
		}
		cfg["id.token.claim"] = "true"
		m["config"] = cfg
		_ = kc.put(token, fmt.Sprintf("/admin/realms/microfoundry/client-scopes/%s/protocol-mappers/models/%s", scopeID, mapperID), m)
		return
	}
}

func (kc *KeycloakConfigurator) put(token, path string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", kc.baseURL+path, bytes.NewReader(data))
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
