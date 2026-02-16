package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all custom MicroFoundry Prometheus metrics.
type Metrics struct {
	AppsTotal        *prometheus.GaugeVec
	AppsRunning      *prometheus.GaugeVec
	AppsDesired      *prometheus.GaugeVec
	ServicesTotal    *prometheus.GaugeVec
	DeployTotal      *prometheus.CounterVec
	DeployErrors     *prometheus.CounterVec
	ScaleEvents      *prometheus.CounterVec
	ServiceProvision *prometheus.CounterVec
	ServiceBind      *prometheus.CounterVec
	HTTPRequests     *prometheus.CounterVec
	HTTPDuration     *prometheus.HistogramVec
}

// NewMetrics creates and registers all MicroFoundry metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		AppsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "microfoundry_apps_total",
			Help: "Total number of managed applications",
		}, []string{"cluster", "namespace"}),

		AppsRunning: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "microfoundry_apps_running",
			Help: "Number of running instances per application",
		}, []string{"cluster", "namespace", "app"}),

		AppsDesired: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "microfoundry_apps_desired",
			Help: "Desired replica count per application",
		}, []string{"cluster", "namespace", "app"}),

		ServicesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "microfoundry_services_total",
			Help: "Total provisioned service instances",
		}, []string{"cluster", "type", "status"}),

		DeployTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "microfoundry_deploy_total",
			Help: "Total deployment operations",
		}, []string{"cluster", "app", "result"}),

		DeployErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "microfoundry_deploy_errors_total",
			Help: "Total deployment failures",
		}, []string{"cluster", "app"}),

		ScaleEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "microfoundry_scale_events_total",
			Help: "Total scale operations",
		}, []string{"cluster", "app"}),

		ServiceProvision: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "microfoundry_service_provision_total",
			Help: "Total service provisioning operations",
		}, []string{"type", "plan", "result"}),

		ServiceBind: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "microfoundry_service_bind_total",
			Help: "Total service bind/unbind operations",
		}, []string{"service", "app", "action"}),

		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "microfoundry_http_requests_total",
			Help: "Total HTTP requests to admin UI",
		}, []string{"method", "path", "status"}),

		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "microfoundry_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
	}

	reg.MustRegister(
		m.AppsTotal, m.AppsRunning, m.AppsDesired, m.ServicesTotal,
		m.DeployTotal, m.DeployErrors, m.ScaleEvents,
		m.ServiceProvision, m.ServiceBind,
		m.HTTPRequests, m.HTTPDuration,
	)

	return m
}

// Handler returns the Prometheus metrics HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}
