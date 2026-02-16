package admin

import (
	"fmt"
	"net/http"
)

const dashboardPerfUID = "mf-app-detail"

// AppPerformanceTabHandler renders the Performance tab with RED metrics and Grafana panels.
func (s *Server) AppPerformanceTabHandler(w http.ResponseWriter, r *http.Request, name string) {
	ctx := r.Context()

	red, _ := s.prometheus.GetAppREDMetrics(ctx, name)

	appParams := map[string]string{"var-app": name}
	data := map[string]any{
		"Name":        name,
		"RED":         red,
		"BeylaEnabled": s.config.Monitoring.BeylaEnabled,
		// Grafana panel URLs for RED metric panels (IDs 11-19 from enhanced dashboard)
		"GrafanaAppURL":            s.grafana.FullDashboardURL(dashboardPerfUID, appParams),
		"GrafanaReqRatePanelURL":   s.grafana.PanelURL(dashboardPerfUID, 16, appParams),
		"GrafanaLatencyPanelURL":   s.grafana.PanelURL(dashboardPerfUID, 17, appParams),
		"GrafanaStatusPanelURL":    s.grafana.PanelURL(dashboardPerfUID, 18, appParams),
		"GrafanaHeatmapPanelURL":   s.grafana.PanelURL(dashboardPerfUID, 19, appParams),
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
