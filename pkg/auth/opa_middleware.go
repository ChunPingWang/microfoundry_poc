package auth

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// OPAMiddleware creates an HTTP middleware that evaluates OPA authorization policies.
// When opa is nil (auth disabled), all requests are allowed.
func OPAMiddleware(opa *OPAEngine, sessions *SessionManager, orgStore *OrgStore, wsStore *WorkspaceStore, auditLog *AuditLog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip public paths
			if isPublicPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// If OPA engine is nil, allow everything (auth disabled)
			if opa == nil {
				next.ServeHTTP(w, r)
				return
			}

			input := buildAuthzInput(r, orgStore, wsStore)
			result, err := opa.Evaluate(r.Context(), input)
			if err != nil {
				http.Error(w, "authorization error", http.StatusInternalServerError)
				return
			}

			// Record audit entry
			if auditLog != nil {
				email := ""
				userID := ""
				if input.User != nil {
					email = input.User.Email
					userID = input.User.ID
				}
				auditLog.Record(AuditEntry{
					Timestamp: time.Now().UTC(),
					UserEmail: email,
					UserID:    userID,
					Action:    input.Action,
					Resource:  input.Resource,
					Path:      r.URL.Path,
					Method:    r.Method,
					OrgID:     input.OrgID,
					Allowed:   result.Allow,
					Reason:    result.Reason,
					IP:        r.RemoteAddr,
				})
			}

			if !result.Allow {
				if input.User == nil {
					http.Redirect(w, r, "/login", http.StatusFound)
					return
				}
				// API routes get JSON 403; web routes redirect to denied page
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"error":"forbidden"}`))
					return
				}
				http.Redirect(w, r, "/denied", http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicPath returns true for paths that bypass OPA authorization.
func isPublicPath(path string) bool {
	publicPrefixes := []string{
		"/static/",
		"/metrics",
		"/login",
		"/auth/",
		"/health",
	}
	for _, p := range publicPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return path == "/" || path == "/favicon.ico" || path == "/denied"
}

// buildAuthzInput constructs the OPA input from the HTTP request context.
func buildAuthzInput(r *http.Request, orgStore *OrgStore, wsStore *WorkspaceStore) AuthzInput {
	user := UserFromContext(r.Context())
	action, resource := classifyRoute(r.Method, r.URL.Path)

	input := AuthzInput{
		Action:      action,
		Resource:    resource,
		OrgID:       ActiveOrgFromContext(r.Context()),
		WorkspaceID: ActiveWorkspaceFromContext(r.Context()),
		Path:        r.URL.Path,
		Method:      r.Method,
	}

	if user != nil {
		orgRole := ""
		if orgStore != nil && input.OrgID != "" {
			members, err := orgStore.ListMembers(r.Context(), input.OrgID)
			if err != nil {
				log.Printf("[OPA] failed to list org members for org %s: %v", input.OrgID, err)
			}
			for _, m := range members {
				if m.Email == user.Email {
					orgRole = m.Role
					break
				}
			}
		}

		wsRole := ""
		if wsStore != nil && input.WorkspaceID != "" {
			members, err := wsStore.ListMembers(r.Context(), input.WorkspaceID)
			if err != nil {
				log.Printf("[OPA] failed to list workspace members for ws %s: %v", input.WorkspaceID, err)
			}
			for _, m := range members {
				if m.Email == user.Email {
					wsRole = m.Role
					break
				}
			}
		}

		input.User = &AuthzUser{
			ID:            user.UserID,
			Email:         user.Email,
			Roles:         user.Roles,
			OrgRole:       orgRole,
			WorkspaceRole: wsRole,
		}
	}

	return input
}

// classifyRoute determines the action and resource type from HTTP method and path.
func classifyRoute(method, path string) (action, resource string) {
	// Determine action from HTTP method
	switch method {
	case "GET", "HEAD", "OPTIONS":
		action = "read"
	case "POST", "PUT", "PATCH":
		action = "write"
	case "DELETE":
		action = "delete"
	default:
		action = "read"
	}

	// Determine resource from path
	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) == 0 {
		return action, "dashboard"
	}

	switch parts[0] {
	case "apps":
		resource = "apps"
	case "services":
		resource = "services"
	case "secrets":
		resource = "secrets"
	case "clusters":
		resource = "clusters"
	case "catalog", "topologies":
		resource = "catalog"
	case "monitoring":
		resource = "monitoring"
	case "config":
		resource = "settings"
	case "settings":
		resource = "settings"
	case "users":
		resource = "users"
	case "workspaces":
		resource = "workspaces"
	case "scim":
		resource = "scim"
	case "api":
		// For API routes, classify by the second path segment
		if len(parts) >= 2 {
			switch parts[1] {
			case "apps":
				resource = "apps"
			case "services":
				resource = "services"
			case "secrets":
				resource = "secrets"
			case "clusters":
				resource = "clusters"
			case "catalog", "topologies":
				resource = "catalog"
			case "orgs":
				resource = "users"
			case "settings":
				resource = "settings"
			case "monitoring":
				resource = "monitoring"
			case "config":
				resource = "settings"
			case "audit":
				resource = "audit"
			case "workspaces":
				resource = "workspaces"
			default:
				resource = parts[1]
			}
		} else {
			resource = "api"
		}
	default:
		resource = "unknown"
	}

	return action, resource
}
