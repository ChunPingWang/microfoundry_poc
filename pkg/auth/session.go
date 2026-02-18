package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/gorilla/sessions"
)

const sessionName = "mf-session"

// UserSession represents the authenticated user's session data.
type UserSession struct {
	UserID            string   `json:"user_id"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	Roles             []string `json:"roles"`
	ActiveWorkspaceID string   `json:"active_workspace"`
	ActiveOrgID       string   `json:"active_org"`
	Authenticated     bool     `json:"authenticated"`
}

// SessionManager wraps gorilla/sessions for secure cookie management.
type SessionManager struct {
	store *sessions.CookieStore
}

// NewSessionManager creates a session manager with the given hex-encoded key.
// If the key is empty or invalid, a random key is generated (sessions won't survive restarts).
func NewSessionManager(hexKey string) *SessionManager {
	authKey := make([]byte, 32)
	encKey := make([]byte, 32)

	if decoded, err := hex.DecodeString(hexKey); err == nil && len(decoded) >= 32 {
		copy(authKey, decoded[:32])
		if len(decoded) >= 64 {
			copy(encKey, decoded[32:64])
		} else {
			rand.Read(encKey)
		}
	} else {
		rand.Read(authKey)
		rand.Read(encKey)
	}

	store := sessions.NewCookieStore(authKey, encKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	return &SessionManager{store: store}
}

// GetUser extracts the authenticated user from the session cookie.
func (sm *SessionManager) GetUser(r *http.Request) (*UserSession, error) {
	session, err := sm.store.Get(r, sessionName)
	if err != nil {
		return nil, err
	}

	data, ok := session.Values["user"].([]byte)
	if !ok {
		return nil, nil
	}

	var user UserSession
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	if !user.Authenticated {
		return nil, nil
	}
	return &user, nil
}

// SetUser stores the user session in the cookie.
func (sm *SessionManager) SetUser(w http.ResponseWriter, r *http.Request, user *UserSession) error {
	session, err := sm.store.Get(r, sessionName)
	if err != nil {
		return err
	}

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	session.Values["user"] = data
	return session.Save(r, w)
}

// SetOAuthState stores the OIDC state and PKCE verifier in the session.
func (sm *SessionManager) SetOAuthState(w http.ResponseWriter, r *http.Request, state, verifier string) error {
	session, err := sm.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Values["oauth_state"] = state
	session.Values["pkce_verifier"] = verifier
	return session.Save(r, w)
}

// GetOAuthState retrieves and clears the OIDC state and PKCE verifier.
func (sm *SessionManager) GetOAuthState(r *http.Request) (state, verifier string) {
	session, err := sm.store.Get(r, sessionName)
	if err != nil {
		return "", ""
	}
	s, _ := session.Values["oauth_state"].(string)
	v, _ := session.Values["pkce_verifier"].(string)
	return s, v
}

// SetActiveWorkspace updates the active workspace in the session.
func (sm *SessionManager) SetActiveWorkspace(w http.ResponseWriter, r *http.Request, wsID string) error {
	user, err := sm.GetUser(r)
	if err != nil || user == nil {
		return err
	}
	user.ActiveWorkspaceID = wsID
	return sm.SetUser(w, r, user)
}

// SetActiveOrg updates the active org in the session.
func (sm *SessionManager) SetActiveOrg(w http.ResponseWriter, r *http.Request, orgID string) error {
	user, err := sm.GetUser(r)
	if err != nil || user == nil {
		return err
	}
	user.ActiveOrgID = orgID
	return sm.SetUser(w, r, user)
}

// Clear destroys the session.
func (sm *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	session, err := sm.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	session.Options.MaxAge = -1
	return session.Save(r, w)
}
