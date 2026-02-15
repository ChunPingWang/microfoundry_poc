package admin

import (
	"context"
	"net/http"
	"strconv"
)

// PageData is the base data passed to every full-page template.
type PageData struct {
	Title   string
	Version string
	Active  string
	Content any
}

// AppRow is used for the app list table.
type AppRow struct {
	Name         string
	State        string
	RunningCount int
	TotalCount   int
	Routes       []string
}

func (s *Server) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	apps, _ := s.k8sClient.ListApps(ctx)

	data := PageData{
		Title:   "Dashboard",
		Version: s.version,
		Active:  "dashboard",
		Content: map[string]any{
			"AppCount":  len(apps),
			"Domain":    s.k8sClient.Domain,
			"Namespace": s.k8sClient.Namespace,
			"Context":   s.config.Kubernetes.Context,
			"GitHub":    s.config.GitHub.Owner + "/" + s.config.GitHub.Repo,
		},
	}
	s.templates.Render(w, "dashboard.html", data)
}

func (s *Server) AppsListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	names, err := s.k8sClient.ListApps(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ingressMap, _ := s.k8sClient.ListIngresses(ctx)

	var apps []AppRow
	for _, name := range names {
		row := AppRow{Name: name, State: "unknown"}
		if status, err := s.k8sClient.GetAppStatus(ctx, name); err == nil {
			row.RunningCount = status.RunningCount
			row.TotalCount = status.TotalCount
			if status.RunningCount > 0 {
				row.State = "started"
			} else {
				row.State = "stopped"
			}
		}
		if hosts, ok := ingressMap[name]; ok {
			row.Routes = hosts
		}
		apps = append(apps, row)
	}

	data := PageData{
		Title:   "Applications",
		Version: s.version,
		Active:  "apps",
		Content: apps,
	}
	s.templates.Render(w, "apps.html", data)
}

func (s *Server) AppDetailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	name := r.PathValue("name")

	status, err := s.k8sClient.GetAppStatus(ctx, name)
	if err != nil {
		http.Error(w, "App not found: "+err.Error(), http.StatusNotFound)
		return
	}

	ingressMap, _ := s.k8sClient.ListIngresses(ctx)
	routes := ingressMap[name]

	state := "stopped"
	if status.RunningCount > 0 {
		state = "started"
	}

	data := PageData{
		Title:   name,
		Version: s.version,
		Active:  "apps",
		Content: map[string]any{
			"Name":         name,
			"State":        state,
			"RunningCount": status.RunningCount,
			"TotalCount":   status.TotalCount,
			"Instances":    status.Instances,
			"Routes":       routes,
		},
	}
	s.templates.Render(w, "app_detail.html", data)
}

func (s *Server) AppInstancesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	name := r.PathValue("name")

	status, err := s.k8sClient.GetAppStatus(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.templates.RenderPartial(w, "app_instances.html", map[string]any{
		"Instances":    status.Instances,
		"RunningCount": status.RunningCount,
		"TotalCount":   status.TotalCount,
	})
}

func (s *Server) ScaleAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	name := r.PathValue("name")

	instancesStr := r.FormValue("instances")
	instances, err := strconv.Atoi(instancesStr)
	if err != nil || instances < 0 {
		http.Error(w, "Invalid instance count", http.StatusBadRequest)
		return
	}

	if err := s.k8sClient.ScaleApp(ctx, name, instances); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated app row
	ingressMap, _ := s.k8sClient.ListIngresses(ctx)
	row := AppRow{Name: name, State: "started", RunningCount: instances, TotalCount: instances}
	if hosts, ok := ingressMap[name]; ok {
		row.Routes = hosts
	}
	s.templates.RenderPartial(w, "app_row.html", row)
}

func (s *Server) DeleteAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	name := r.PathValue("name")

	if err := s.k8sClient.DeleteApp(ctx, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Configuration",
		Version: s.version,
		Active:  "config",
		Content: map[string]any{
			"Domain":    s.k8sClient.Domain,
			"Namespace": s.k8sClient.Namespace,
			"Context":   s.config.Kubernetes.Context,
			"GitHub":    s.config.GitHub.Owner + "/" + s.config.GitHub.Repo,
			"Owner":     s.config.GitHub.Owner,
			"Repo":      s.config.GitHub.Repo,
		},
	}
	s.templates.Render(w, "config.html", data)
}

func (s *Server) ServicesHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Backing Services",
		Version: s.version,
		Active:  "services",
	}
	s.templates.Render(w, "services.html", data)
}

func (s *Server) SecretsHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Secrets",
		Version: s.version,
		Active:  "secrets",
	}
	s.templates.Render(w, "secrets.html", data)
}

func (s *Server) UsersHandler(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Title:   "Users & Organizations",
		Version: s.version,
		Active:  "users",
	}
	s.templates.Render(w, "users.html", data)
}
