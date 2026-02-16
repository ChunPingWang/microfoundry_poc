package admin

import (
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

func (s *Server) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	ctx := r.Context()
	items, err := s.k8sClient.ListAppItems(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply state filter if provided
	filter := r.URL.Query().Get("state")
	if filter != "" && filter != "all" {
		var filtered []any
		for _, item := range items {
			if item.State == filter {
				filtered = append(filtered, item)
			}
		}
		data := PageData{
			Title:   "Applications",
			Version: s.version,
			Active:  "apps",
			Content: map[string]any{
				"Apps":   filtered,
				"Filter": filter,
			},
		}
		s.templates.Render(w, "apps.html", data)
		return
	}

	data := PageData{
		Title:   "Applications",
		Version: s.version,
		Active:  "apps",
		Content: map[string]any{
			"Apps":   items,
			"Filter": "all",
		},
	}
	s.templates.Render(w, "apps.html", data)
}

func (s *Server) AppDetailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	detail, err := s.k8sClient.GetAppDetail(ctx, name)
	if err != nil {
		http.Error(w, "App not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Determine active tab from query parameter
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "overview"
	}

	data := PageData{
		Title:   name,
		Version: s.version,
		Active:  "apps",
		Content: map[string]any{
			"Detail": detail,
			"Tab":    tab,
		},
	}
	s.templates.Render(w, "app_detail.html", data)
}

// validTabs is the allowlist of valid tab names for the app detail view.
var validTabs = map[string]bool{
	"overview": true, "instances": true, "config": true,
	"services": true, "routes": true, "logs": true,
}

// AppTabHandler serves individual tab content as HTMX partials.
func (s *Server) AppTabHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")
	tab := r.PathValue("tab")

	if !validTabs[tab] {
		http.Error(w, "invalid tab", http.StatusBadRequest)
		return
	}

	detail, err := s.k8sClient.GetAppDetail(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	templateName := "tab_" + tab + ".html"
	s.templates.RenderPartial(w, templateName, detail)
}

func (s *Server) AppInstancesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
	ctx := r.Context()
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

	// Return updated app row via ListAppItems
	items, _ := s.k8sClient.ListAppItems(ctx)
	for _, item := range items {
		if item.Name == name {
			s.templates.RenderPartial(w, "app_row.html", item)
			return
		}
	}

	// Fallback
	http.Redirect(w, r, "/apps", http.StatusSeeOther)
}

func (s *Server) DeleteAppHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
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
