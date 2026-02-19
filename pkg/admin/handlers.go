package admin

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"github.com/yuin/goldmark"
	"github.com/younjinjeong/microfoundry/docs"
	"github.com/younjinjeong/microfoundry/pkg/auth"
	"github.com/younjinjeong/microfoundry/pkg/models"
	"github.com/younjinjeong/microfoundry/pkg/monitoring"
	"github.com/younjinjeong/microfoundry/pkg/service"
)

// PageData is the base data passed to every full-page template.
type PageData struct {
	Title           string
	Version         string
	Active          string
	ActiveCluster   string
	Content         any
	User            *auth.UserSession // nil when auth is disabled or not logged in
	AuthEnabled     bool
	TooltipsEnabled bool
	AdminDomain     string
	TLSEnabled      bool
}

func (s *Server) pageData(title, active string) PageData {
	return PageData{
		Title:           title,
		Version:         s.version,
		Active:          active,
		ActiveCluster:   s.clientManager.GetActive(),
		AuthEnabled:     s.authEnabled(),
		TooltipsEnabled: s.config.UI.Tooltips,
		AdminDomain:     s.adminDomain,
		TLSEnabled:      s.tlsEnabled,
	}
}

func (s *Server) pageDataWithUser(r *http.Request, title, active string) PageData {
	pd := s.pageData(title, active)
	pd.User = auth.UserFromContext(r.Context())
	return pd
}

func (s *Server) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := auth.UserFromContext(ctx)

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Always fetched: apps and services (2 K8s calls)
	apps, _ := client.ListAppItems(ctx)
	svcMgr := service.NewManager(client)
	services, _ := svcMgr.List(ctx)

	// Compute app state breakdown
	var appsStarted, appsStopped, appsCrashed int
	for _, a := range apps {
		switch a.State {
		case "STARTED", "started", "running":
			appsStarted++
		case "STOPPED", "stopped":
			appsStopped++
		case "CRASHED", "crashed", "failed":
			appsCrashed++
		}
	}

	// Pre-slice for dashboard tables
	topApps := apps
	if len(topApps) > 8 {
		topApps = topApps[:8]
	}
	topServices := services
	if len(topServices) > 6 {
		topServices = topServices[:6]
	}

	content := map[string]any{
		"AppCount":         len(apps),
		"Apps":             apps,
		"TopApps":          topApps,
		"AppsStarted":      appsStarted,
		"AppsStopped":      appsStopped,
		"AppsCrashed":      appsCrashed,
		"ServiceCount":     len(services),
		"TopServices":      topServices,
		"CatalogTypeCount": len(service.Catalog()),
		"Domain":           client.Domain,
		"Namespace":        client.Namespace,
		"Context":          s.config.Kubernetes.Active,
		"GitHub":           s.config.GitHub.Owner + "/" + s.config.GitHub.Repo,
	}

	// Platform-admin only: clusters, alerts, monitoring health, grafana
	isPlatformAdmin := user != nil && hasRoleSlice(user.Roles, "platform-admin")
	if isPlatformAdmin {
		content["Clusters"] = s.clientManager.ListClusters(ctx)
		alerts, _ := s.alertmanager.ListAlerts(ctx)
		var firing []monitoring.Alert
		for _, a := range alerts {
			if a.Status == "firing" {
				firing = append(firing, a)
			}
		}
		content["Alerts"] = firing
		content["FiringCount"] = len(firing)
		content["MonitoringComponents"] = s.checkMonitoringHealth(ctx)
		content["GrafanaOverviewURL"] = s.grafana.DashboardURL(dashboardOverviewUID, nil)
	}

	// Workspace/org counts for admin roles
	isWSAdmin := user != nil && hasRoleSlice(user.Roles, "workspace-admin")
	isOrgAdmin := user != nil && hasRoleSlice(user.Roles, "org-admin")
	if isPlatformAdmin || isWSAdmin {
		if s.wsStore != nil {
			ws, _ := s.wsStore.List(ctx)
			content["WorkspaceCount"] = len(ws)
		}
	}
	if isPlatformAdmin || isWSAdmin || isOrgAdmin {
		if s.orgStore_ != nil {
			orgs, _ := s.orgStore_.List(ctx)
			content["OrgCount"] = len(orgs)
		}
	}

	data := s.pageDataWithUser(r, "Dashboard", "dashboard")
	data.Content = content
	s.templates.Render(w, "dashboard.html", data)
}

// hasRoleSlice checks if a slice of roles contains the target role.
func hasRoleSlice(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

func (s *Server) AppsListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	items, err := client.ListAppItems(ctx)
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
		data := s.pageDataWithUser(r, "Applications", "apps")
		data.Content = map[string]any{
			"Apps":   filtered,
			"Filter": filter,
		}
		s.templates.Render(w, "apps.html", data)
		return
	}

	data := s.pageDataWithUser(r, "Applications", "apps")
	data.Content = map[string]any{
		"Apps":   items,
		"Filter": "all",
	}
	s.templates.Render(w, "apps.html", data)
}

func (s *Server) AppDetailHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	detail, err := client.GetAppDetail(ctx, name)
	if err != nil {
		http.Error(w, "App not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Determine active tab from query parameter
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "overview"
	}

	content := map[string]any{
		"Detail": detail,
		"Tab":    tab,
	}

	// Enrich context for tabs that need additional data on initial page load
	if tab == "logs" {
		var red *monitoring.REDMetrics
		if s.config.Monitoring.BeylaEnabled {
			red, _ = s.prometheus.GetAppREDMetrics(ctx, name)
		}
		content["LogsData"] = map[string]any{
			"Name":         name,
			"RED":          red,
			"BeylaEnabled": s.config.Monitoring.BeylaEnabled,
		}
	}
	if tab == "performance" {
		red, _ := s.prometheus.GetAppREDMetrics(ctx, name)
		namespace := "microfoundry"
		if client != nil {
			namespace = client.Namespace
		}
		end := time.Now()
		start := end.Add(-15 * time.Minute)
		recentLogs, _ := s.loki.QueryLogs(ctx, namespace, name, start, end, 50)

		appParams := map[string]string{"var-app": name}
		content["PerfData"] = map[string]any{
			"Name":                   name,
			"RED":                    red,
			"BeylaEnabled":           s.config.Monitoring.BeylaEnabled,
			"RecentLogs":             recentLogs,
			"GrafanaAppURL":          s.grafana.FullDashboardURL(dashboardPerfUID, appParams),
			"GrafanaReqRatePanelURL": s.grafana.PanelURL(dashboardPerfUID, 16, appParams),
			"GrafanaLatencyPanelURL": s.grafana.PanelURL(dashboardPerfUID, 17, appParams),
			"GrafanaStatusPanelURL":  s.grafana.PanelURL(dashboardPerfUID, 18, appParams),
			"GrafanaHeatmapPanelURL": s.grafana.PanelURL(dashboardPerfUID, 19, appParams),
		}
	}
	if tab == "metrics" {
		appParams := map[string]string{"var-app": name}
		content["MetricsData"] = map[string]any{
			"Name":                   detail.Name,
			"GrafanaAppURL":          s.grafana.FullDashboardURL(dashboardAppUID, appParams),
			"GrafanaCPUPanelURL":     s.grafana.PanelURL(dashboardAppUID, 1, appParams),
			"GrafanaMemPanelURL":     s.grafana.PanelURL(dashboardAppUID, 2, appParams),
			"GrafanaNetPanelURL":     s.grafana.PanelURL(dashboardAppUID, 3, appParams),
			"GrafanaRestartPanelURL": s.grafana.PanelURL(dashboardAppUID, 4, appParams),
			"GrafanaLogPanelURL":     s.grafana.PanelURL(dashboardAppUID, 5, appParams),
		}
	}
	if tab == "services" {
		mgr := service.NewManager(client)
		allServices, _ := mgr.List(ctx)
		boundNames := make(map[string]bool)
		for _, svc := range detail.Services {
			boundNames[svc.Name] = true
		}
		var available []models.ServiceListItem
		for _, svc := range allServices {
			if svc.Status == models.ServiceStatusAvailable && !boundNames[svc.Name] {
				available = append(available, svc)
			}
		}
		content["ServicesData"] = map[string]any{
			"AppName":           name,
			"Services":          detail.Services,
			"Secrets":           detail.Secrets,
			"AvailableServices": available,
		}
	}

	data := s.pageDataWithUser(r, name, "apps")
	data.Content = content
	s.templates.Render(w, "app_detail.html", data)
}

// validTabs is the allowlist of valid tab names for the app detail view.
var validTabs = map[string]bool{
	"overview": true, "instances": true, "config": true,
	"services": true, "routes": true, "logs": true,
	"metrics": true, "performance": true,
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

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	detail, err := client.GetAppDetail(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Performance tab uses dedicated handler with RED metrics + logs
	if tab == "performance" {
		s.AppPerformanceTabHandler(w, r, name)
		return
	}

	// Logs tab uses dedicated handler with RED metrics summary
	if tab == "logs" {
		s.AppLogsTabHandler(w, r, name)
		return
	}


	// Metrics tab needs Grafana panel URLs
	if tab == "metrics" {
		appParams := map[string]string{"var-app": name}
		metricsData := map[string]any{
			"Name":                   detail.Name,
			"GrafanaAppURL":          s.grafana.FullDashboardURL(dashboardAppUID, appParams),
			"GrafanaCPUPanelURL":     s.grafana.PanelURL(dashboardAppUID, 1, appParams),
			"GrafanaMemPanelURL":     s.grafana.PanelURL(dashboardAppUID, 2, appParams),
			"GrafanaNetPanelURL":     s.grafana.PanelURL(dashboardAppUID, 3, appParams),
			"GrafanaRestartPanelURL": s.grafana.PanelURL(dashboardAppUID, 4, appParams),
			"GrafanaLogPanelURL":     s.grafana.PanelURL(dashboardAppUID, 5, appParams),
		}
		s.templates.RenderPartial(w, "tab_metrics.html", metricsData)
		return
	}

	// Services tab needs available service instances for bind UI
	if tab == "services" {
		mgr := service.NewManager(client)
		allServices, _ := mgr.List(ctx)

		// Filter to only available (not already bound to this app)
		boundNames := make(map[string]bool)
		for _, svc := range detail.Services {
			boundNames[svc.Name] = true
		}
		var available []models.ServiceListItem
		for _, svc := range allServices {
			if svc.Status == models.ServiceStatusAvailable && !boundNames[svc.Name] {
				available = append(available, svc)
			}
		}

		svcData := map[string]any{
			"AppName":           name,
			"Services":          detail.Services,
			"Secrets":           detail.Secrets,
			"AvailableServices": available,
		}
		s.templates.RenderPartial(w, "tab_services.html", svcData)
		return
	}

	templateName := "tab_" + tab + ".html"
	s.templates.RenderPartial(w, templateName, detail)
}

func (s *Server) AppInstancesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	status, err := client.GetAppStatus(ctx, name)
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

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Handle instance scaling
	if instancesStr := r.FormValue("instances"); instancesStr != "" {
		instances, err := strconv.Atoi(instancesStr)
		if err != nil || instances < 0 || instances > 20 {
			http.Error(w, "instances must be between 0 and 20", http.StatusBadRequest)
			return
		}
		if err := client.ScaleApp(ctx, name, instances); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.metrics.ScaleEvents.WithLabelValues(s.clientManager.GetActive(), name).Inc()
	}

	// Handle memory/disk resource update
	memStr := r.FormValue("memory_mb")
	diskStr := r.FormValue("disk_mb")
	if memStr != "" || diskStr != "" {
		var memMB, diskMB int
		if memStr != "" {
			memMB, err = strconv.Atoi(memStr)
			if err != nil || memMB < 64 || memMB > 8192 {
				http.Error(w, "memory must be between 64 and 8192 MB", http.StatusBadRequest)
				return
			}
		}
		if diskStr != "" {
			diskMB, err = strconv.Atoi(diskStr)
			if err != nil || diskMB < 256 || diskMB > 8192 {
				http.Error(w, "disk must be between 256 and 8192 MB", http.StatusBadRequest)
				return
			}
		}
		if err := client.UpdateAppResources(ctx, name, memMB, diskMB); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Return updated app row via ListAppItems
	items, _ := client.ListAppItems(ctx)
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

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	if err := client.DeleteApp(ctx, name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	client, _ := s.getClient(r)

	domain := ""
	namespace := ""
	if client != nil {
		domain = client.Domain
		namespace = client.Namespace
	}

	data := s.pageDataWithUser(r, "Configuration", "config")
	data.Content = map[string]any{
		"Domain":    domain,
		"Namespace": namespace,
		"Context":   s.config.Kubernetes.Active,
		"GitHub":    s.config.GitHub.Owner + "/" + s.config.GitHub.Repo,
		"Owner":     s.config.GitHub.Owner,
		"Repo":      s.config.GitHub.Repo,
	}
	s.templates.Render(w, "config.html", data)
}

// DocsHandler renders the in-app documentation page.
func (s *Server) DocsHandler(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "manual"
	}

	fileMap := map[string]string{
		"manual":       "user-manual.md",
		"admin":        "admin-guide.md",
		"architecture": "architecture.md",
	}

	filename, ok := fileMap[tab]
	if !ok {
		tab = "manual"
		filename = "user-manual.md"
	}

	md, err := docs.Files.ReadFile(filename)
	if err != nil {
		http.Error(w, "doc not found: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := goldmark.Convert(md, &buf); err != nil {
		http.Error(w, "markdown render error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := s.pageDataWithUser(r, "Documentation", "docs")
	data.Content = map[string]any{
		"Tab":  tab,
		"HTML": buf.String(),
	}
	s.templates.Render(w, "docs.html", data)
}

// DeniedHandler renders the access denied page.
func (s *Server) DeniedHandler(w http.ResponseWriter, r *http.Request) {
	data := s.pageDataWithUser(r, "Access Denied", "")
	s.templates.Render(w, "denied.html", data)
}

// UsersHandler is now replaced by OrgsPageHandler in org_handlers.go
