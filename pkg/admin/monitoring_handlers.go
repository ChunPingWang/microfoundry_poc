package admin

import (
	"net/http"
	"time"
)

const (
	dashboardOverviewUID = "mf-overview"
	dashboardAppUID      = "mf-app-detail"
	dashboardClusterUID  = "mf-cluster"
)

// MonitoringHandler renders the Metrics & Alerts page with embedded Grafana dashboards.
func (s *Server) MonitoringHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Build embeddable Grafana iframe URLs
	overviewURL := s.grafana.DashboardURL(dashboardOverviewUID, nil)
	clusterURL := s.grafana.DashboardURL(dashboardClusterUID, nil)
	fullOverviewURL := s.grafana.FullDashboardURL(dashboardOverviewUID, nil)
	fullClusterURL := s.grafana.FullDashboardURL(dashboardClusterUID, nil)

	// Query Alertmanager for active alerts
	alerts, _ := s.alertmanager.ListAlerts(ctx)

	firingCount := 0
	for _, a := range alerts {
		if a.Status == "firing" {
			firingCount++
		}
	}

	data := s.pageData("Metrics & Alerts", "monitoring")
	data.Content = map[string]any{
		"OverviewIframeURL": overviewURL,
		"ClusterIframeURL":  clusterURL,
		"FullOverviewURL":   fullOverviewURL,
		"FullClusterURL":    fullClusterURL,
		"GrafanaBaseURL":    s.grafana.BaseURL,
		"Alerts":            alerts,
		"FiringCount":       firingCount,
	}
	s.templates.Render(w, "monitoring.html", data)
}

// AlertsListHandler returns the alerts partial for HTMX polling.
func (s *Server) AlertsListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alerts, _ := s.alertmanager.ListAlerts(ctx)

	s.templates.RenderPartial(w, "alerts_list.html", map[string]any{
		"Alerts": alerts,
	})
}

// LogHistoryHandler returns historical logs from Loki as an HTML partial.
func (s *Server) LogHistoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	client, err := s.getClient(r)
	if err != nil {
		http.Error(w, "No cluster available: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	duration := r.URL.Query().Get("duration")
	if duration == "" {
		duration = "1h"
	}
	d, _ := time.ParseDuration(duration)
	if d == 0 {
		d = time.Hour
	}
	end := time.Now()
	start := end.Add(-d)

	entries, err := s.loki.QueryLogs(ctx, client.Namespace, name, start, end, 500)
	if err != nil {
		http.Error(w, "Querying logs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	s.templates.RenderPartial(w, "log_history.html", map[string]any{
		"Entries":  entries,
		"AppName":  name,
		"Duration": duration,
	})
}

// APIAlertsHandler returns alerts as JSON.
func (s *Server) APIAlertsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	alerts, err := s.alertmanager.ListAlerts(ctx)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

// APILogHistoryHandler returns historical logs from Loki as JSON.
func (s *Server) APILogHistoryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	client, err := s.getClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	duration := r.URL.Query().Get("duration")
	if duration == "" {
		duration = "1h"
	}
	d, _ := time.ParseDuration(duration)
	if d == 0 {
		d = time.Hour
	}
	end := time.Now()
	start := end.Add(-d)

	entries, err := s.loki.QueryLogs(ctx, client.Namespace, name, start, end, 500)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, entries)
}
