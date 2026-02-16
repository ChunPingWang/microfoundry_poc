package admin

import (
	"fmt"
	"net/http"
	"time"

	"github.com/younjinjeong/microfoundry/pkg/monitoring"
)

const dashboardPerfUID = "mf-app-detail"

// AppPerformanceTabHandler renders the Performance tab with RED metrics, Grafana panels, and recent logs.
func (s *Server) AppPerformanceTabHandler(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	red, _ := s.prometheus.GetAppREDMetrics(ctx, name)

	// Fetch recent logs from Loki for the integrated view
	client, _ := s.getClient(r)
	namespace := "microfoundry"
	if client != nil {
		namespace = client.Namespace
	}
	end := time.Now()
	start := end.Add(-15 * time.Minute)
	recentLogs, _ := s.loki.QueryLogs(ctx, namespace, name, start, end, 50)

	appParams := map[string]string{"var-app": name}
	data := map[string]any{
		"Name":         name,
		"RED":          red,
		"BeylaEnabled": s.config.Monitoring.BeylaEnabled,
		"RecentLogs":   recentLogs,
		// Grafana panel URLs for RED metric panels (IDs 11-19 from enhanced dashboard)
		"GrafanaAppURL":          s.grafana.FullDashboardURL(dashboardPerfUID, appParams),
		"GrafanaReqRatePanelURL": s.grafana.PanelURL(dashboardPerfUID, 16, appParams),
		"GrafanaLatencyPanelURL": s.grafana.PanelURL(dashboardPerfUID, 17, appParams),
		"GrafanaStatusPanelURL":  s.grafana.PanelURL(dashboardPerfUID, 18, appParams),
		"GrafanaHeatmapPanelURL": s.grafana.PanelURL(dashboardPerfUID, 19, appParams),
	}

	s.templates.RenderPartial(w, "tab_performance.html", data)
}

// APIAppREDMetricsHandler returns RED metrics for an app as JSON.
func (s *Server) APIAppREDMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	red, err := s.prometheus.GetAppREDMetrics(ctx, name)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, red)
}

// APIAppHealthHandler returns a health summary combining RED metrics and alert status.
func (s *Server) APIAppHealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	red, _ := s.prometheus.GetAppREDMetrics(ctx, name)

	// Determine health status from RED metrics
	status := "healthy"
	var issues []string

	if red.HasData {
		if red.ErrorRate > 0.05 {
			status = "degraded"
			issues = append(issues, fmt.Sprintf("high error rate: %.1f%%", red.ErrorRate*100))
		}
		if red.LatencyP99 > 2.0 {
			status = "degraded"
			issues = append(issues, fmt.Sprintf("high p99 latency: %.0fms", red.LatencyP99*1000))
		}
		if red.RequestRate == 0 {
			issues = append(issues, "no traffic detected")
		}
	} else {
		status = "unknown"
		issues = append(issues, "no Beyla metrics available")
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"app":     name,
		"status":  status,
		"issues":  issues,
		"metrics": red,
	})
}

// APIAppObservabilityHandler returns combined RED metrics + recent logs for an app.
func (s *Server) APIAppObservabilityHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := r.PathValue("name")

	red, _ := s.prometheus.GetAppREDMetrics(ctx, name)

	client, err := s.getClient(r)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	duration := r.URL.Query().Get("duration")
	if duration == "" {
		duration = "15m"
	}
	d, _ := time.ParseDuration(duration)
	if d == 0 {
		d = 15 * time.Minute
	}
	end := time.Now()
	start := end.Add(-d)

	entries, _ := s.loki.QueryLogs(ctx, client.Namespace, name, start, end, 200)

	writeJSON(w, http.StatusOK, map[string]any{
		"app":     name,
		"metrics": red,
		"logs": map[string]any{
			"entries":  entries,
			"duration": duration,
			"count":    len(entries),
		},
	})
}

// AppLogsTabHandler renders the Logs tab with RED metrics summary banner.
func (s *Server) AppLogsTabHandler(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	var red *monitoring.REDMetrics
	if s.config.Monitoring.BeylaEnabled {
		red, _ = s.prometheus.GetAppREDMetrics(ctx, name)
	}

	data := map[string]any{
		"Name":         name,
		"RED":          red,
		"BeylaEnabled": s.config.Monitoring.BeylaEnabled,
	}

	s.templates.RenderPartial(w, "tab_logs.html", data)
}
