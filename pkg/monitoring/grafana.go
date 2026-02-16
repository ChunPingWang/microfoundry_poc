package monitoring

import (
	"fmt"
	"net/url"
)

// GrafanaConfig holds the connection info for embedded Grafana.
type GrafanaConfig struct {
	BaseURL string
}

// DashboardURL returns an embeddable Grafana solo dashboard URL.
func (g *GrafanaConfig) DashboardURL(uid string, params map[string]string) string {
	u, _ := url.Parse(g.BaseURL)
	u.Path = fmt.Sprintf("/d-solo/%s", uid)
	q := u.Query()
	q.Set("orgId", "1")
	q.Set("theme", "light")
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// PanelURL returns an embeddable URL for a specific panel within a dashboard.
func (g *GrafanaConfig) PanelURL(uid string, panelID int, params map[string]string) string {
	u, _ := url.Parse(g.BaseURL)
	u.Path = fmt.Sprintf("/d-solo/%s", uid)
	q := u.Query()
	q.Set("orgId", "1")
	q.Set("theme", "light")
	q.Set("panelId", fmt.Sprintf("%d", panelID))
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// FullDashboardURL returns a non-solo Grafana dashboard URL for "Open in Grafana" links.
func (g *GrafanaConfig) FullDashboardURL(uid string, params map[string]string) string {
	u, _ := url.Parse(g.BaseURL)
	u.Path = fmt.Sprintf("/d/%s", uid)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
