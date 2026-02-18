package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	userContextKey contextKey = "mf-user"
	orgContextKey  contextKey = "mf-org"
	wsContextKey   contextKey = "mf-workspace"
)

// UserFromContext returns the authenticated user from the request context, or nil.
func UserFromContext(ctx context.Context) *UserSession {
	u, _ := ctx.Value(userContextKey).(*UserSession)
	return u
}

// ActiveOrgFromContext returns the active org ID from the request context.
func ActiveOrgFromContext(ctx context.Context) string {
	s, _ := ctx.Value(orgContextKey).(string)
	return s
}

// ActiveWorkspaceFromContext returns the active workspace ID from the request context.
func ActiveWorkspaceFromContext(ctx context.Context) string {
	s, _ := ctx.Value(wsContextKey).(string)
	return s
}

// InjectUser reads the session and adds the user to the request context.
// Does not block unauthenticated requests.
func InjectUser(sessions *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _ := sessions.GetUser(r)
			if user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				ctx = context.WithValue(ctx, orgContextKey, user.ActiveOrgID)
				ctx = context.WithValue(ctx, wsContextKey, user.ActiveWorkspaceID)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth redirects to /login if no valid session exists.
func RequireAuth(sessions *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole checks that the session user has the specified realm role.
func RequireRole(role string, sessions *SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			for _, ur := range user.Roles {
				if ur == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		})
	}
}

// RequireOrgRole checks that the user has at least the given role in the active org.
// Role hierarchy: admin > member > viewer.
func RequireOrgRole(minRole string, sessions *SessionManager, orgStore *OrgStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}

			// Platform admins and workspace admins bypass org role checks
			for _, role := range user.Roles {
				if role == "platform-admin" || role == "workspace-admin" {
					next.ServeHTTP(w, r)
					return
				}
			}

			orgID := ActiveOrgFromContext(r.Context())
			if orgID == "" {
				http.Error(w, "No active organization", http.StatusForbidden)
				return
			}

			members, err := orgStore.ListMembers(r.Context(), orgID)
			if err != nil {
				http.Error(w, "Failed to check permissions", http.StatusInternalServerError)
				return
			}

			userRole := ""
			for _, m := range members {
				if m.Email == user.Email {
					userRole = m.Role
					break
				}
			}

			if !roleAtLeast(userRole, minRole) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// roleAtLeast returns true if actual >= required in the hierarchy admin > member > viewer.
func roleAtLeast(actual, required string) bool {
	levels := map[string]int{"viewer": 1, "member": 2, "admin": 3}
	return levels[actual] >= levels[required]
}
