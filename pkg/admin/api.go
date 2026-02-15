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
	items, err := s.k8sClient.ListAppItems(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) APIGetAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	detail, err := s.k8sClient.GetAppDetail(ctx, name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) APIGetConfigHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"domain":    s.k8sClient.Domain,
		"namespace": s.k8sClient.Namespace,
		"context":   s.config.Kubernetes.Context,
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

	if err := s.k8sClient.ScaleApp(ctx, name, body.Instances); err != nil {
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

	if err := s.k8sClient.DeleteApp(ctx, name); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
