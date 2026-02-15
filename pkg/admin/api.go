package admin

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) APIListAppsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := s.getClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	items, err := client.ListAppItems(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) APIGetAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	client, err := s.getClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	detail, err := client.GetAppDetail(ctx, name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) APIGetConfigHandler(w http.ResponseWriter, r *http.Request) {
	client, _ := s.getClient(r)

	domain := ""
	namespace := ""
	if client != nil {
		domain = client.Domain
		namespace = client.Namespace
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"domain":         domain,
		"namespace":      namespace,
		"active_cluster": s.clientManager.GetActive(),
		"github": map[string]string{
			"owner": s.config.GitHub.Owner,
			"repo":  s.config.GitHub.Repo,
		},
	})
}

func (s *Server) APIScaleAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	var body struct {
		Instances int `json:"instances"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if body.Instances < 0 || body.Instances > 20 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "instances must be between 0 and 20"})
		return
	}

	client, err := s.getClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	if err := client.ScaleApp(ctx, name, body.Instances); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":      name,
		"instances": body.Instances,
		"scaled":    true,
	})
}

func (s *Server) APIDeleteAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	client, err := s.getClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	if err := client.DeleteApp(ctx, name); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
