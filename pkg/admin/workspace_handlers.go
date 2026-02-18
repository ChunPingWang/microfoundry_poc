package admin

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/younjinjeong/microfoundry/pkg/auth"
)

// WorkspacesPageHandler renders the workspace management page.
func (s *Server) WorkspacesPageHandler(w http.ResponseWriter, r *http.Request) {
	data := s.pageDataWithUser(r, "Workspaces", "workspaces")

	user := auth.UserFromContext(r.Context())
	if user == nil {
		s.templates.Render(w, "workspaces.html", data)
		return
	}

	store := s.wsStore
	if store == nil {
		data.Content = map[string]any{"Error": "Workspace store not available"}
		s.templates.Render(w, "workspaces.html", data)
		return
	}

	ctx := r.Context()
	workspaces, err := store.GetUserWorkspaces(ctx, user.Email)
	if err != nil {
		log.Printf("[Workspaces] failed to get user workspaces for %s: %v", user.Email, err)
	}

	// Platform admins see all workspaces
	for _, role := range user.Roles {
		if role == "platform-admin" {
			workspaces, err = store.List(ctx)
			if err != nil {
				log.Printf("[Workspaces] failed to list all workspaces: %v", err)
			}
			break
		}
	}

	selectedWS := r.URL.Query().Get("ws")
	if selectedWS == "" && len(workspaces) > 0 {
		selectedWS = workspaces[0].ID
	}

	content := map[string]any{
		"Workspaces":  workspaces,
		"SelectedWS":  selectedWS,
		"ActiveWS":    user.ActiveWorkspaceID,
		"User":        user,
	}

	if selectedWS != "" {
		wsDetail, err := store.Get(ctx, selectedWS)
		if err == nil {
			content["WSDetail"] = wsDetail
			members, err := store.ListMembers(ctx, selectedWS)
			if err != nil {
				log.Printf("[Workspaces] failed to list members for ws %s: %v", selectedWS, err)
			}
			content["Members"] = members

			// List orgs in this workspace
			if s.orgStore_ != nil {
				orgs, err := s.orgStore_.ListByWorkspace(ctx, selectedWS)
				if err != nil {
					log.Printf("[Workspaces] failed to list orgs for ws %s: %v", selectedWS, err)
				}
				content["Orgs"] = orgs
			}

			userRole := "viewer"
			for _, m := range members {
				if m.Email == user.Email {
					userRole = m.Role
					break
				}
			}
			content["UserRole"] = userRole
		}
	}

	data.Content = content
	s.templates.Render(w, "workspaces.html", data)
}

// CreateWorkspaceHandler creates a new workspace.
func (s *Server) CreateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Workspace name is required", http.StatusBadRequest)
		return
	}

	ws, err := s.wsStore.Create(r.Context(), name, user.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/workspaces?ws="+ws.ID, http.StatusSeeOther)
}

// DeleteWorkspaceHandler deletes a workspace.
func (s *Server) DeleteWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wsID := r.PathValue("id")
	if err := s.wsStore.Delete(r.Context(), wsID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/workspaces")
	w.WriteHeader(http.StatusOK)
}

// InviteWorkspaceMemberHandler adds a member to a workspace.
func (s *Server) InviteWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wsID := r.PathValue("id")
	email := r.FormValue("email")
	name := r.FormValue("name")
	role := r.FormValue("role")

	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if name == "" {
		name = email
	}
	if role == "" {
		role = "member"
	}

	if err := s.wsStore.AddMember(r.Context(), wsID, email, name, role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/workspaces?ws="+wsID, http.StatusSeeOther)
}

// RemoveWorkspaceMemberHandler removes a member from a workspace.
func (s *Server) RemoveWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wsID := r.PathValue("id")
	email := r.PathValue("email")

	if err := s.wsStore.RemoveMember(r.Context(), wsID, email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// SetWorkspaceMemberRoleHandler changes a workspace member's role.
func (s *Server) SetWorkspaceMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	wsID := r.PathValue("id")
	email := r.PathValue("email")
	role := r.FormValue("role")

	validRoles := map[string]bool{"admin": true, "member": true, "viewer": true}
	if !validRoles[role] {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	if err := s.wsStore.SetMemberRole(r.Context(), wsID, email, role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/workspaces?ws="+wsID, http.StatusSeeOther)
}

// SwitchWorkspaceHandler sets the active workspace for the user's session.
func (s *Server) SwitchWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	if s.sessions == nil {
		http.Error(w, "Auth not enabled", http.StatusBadRequest)
		return
	}

	wsID := r.PathValue("id")
	if err := s.sessions.SetActiveWorkspace(w, r, wsID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Also set the first org in this workspace as active org
	if s.orgStore_ != nil {
		orgs, err := s.orgStore_.ListByWorkspace(r.Context(), wsID)
		if err == nil && len(orgs) > 0 {
			_ = s.sessions.SetActiveOrg(w, r, orgs[0].ID)
		}
	}

	http.Redirect(w, r, "/workspaces?ws="+wsID, http.StatusSeeOther)
}

// --- JSON API handlers ---

func (s *Server) APIListWorkspacesHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	workspaces, err := s.wsStore.GetUserWorkspaces(r.Context(), user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, workspaces)
}

func (s *Server) APIGetWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	wsID := r.PathValue("id")
	ws, err := s.wsStore.Get(r.Context(), wsID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	members, err := s.wsStore.ListMembers(r.Context(), wsID)
	if err != nil {
		log.Printf("[Workspaces] failed to list members for ws %s: %v", wsID, err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"workspace": ws,
		"members":   members,
	})
}

func (s *Server) APICreateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	ws, err := s.wsStore.Create(r.Context(), body.Name, user.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, ws)
}

func (s *Server) APIDeleteWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	wsID := r.PathValue("id")
	if err := s.wsStore.Delete(r.Context(), wsID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) APIListWorkspaceMembersHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	wsID := r.PathValue("id")
	members, err := s.wsStore.ListMembers(r.Context(), wsID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, members)
}

func (s *Server) APIAddWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	wsID := r.PathValue("id")
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required"})
		return
	}
	if body.Name == "" {
		body.Name = body.Email
	}
	if body.Role == "" {
		body.Role = "member"
	}

	if err := s.wsStore.AddMember(r.Context(), wsID, body.Email, body.Name, body.Role); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

func (s *Server) APIRemoveWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	wsID := r.PathValue("id")
	email := r.PathValue("email")

	if err := s.wsStore.RemoveMember(r.Context(), wsID, email); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) APIListWorkspaceOrgsHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	wsID := r.PathValue("id")
	if s.orgStore_ == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "org store not available"})
		return
	}

	orgs, err := s.orgStore_.ListByWorkspace(r.Context(), wsID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, orgs)
}
