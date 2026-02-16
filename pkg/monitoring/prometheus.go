package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PrometheusClient queries Prometheus for pre-computed recording rule values.
type PrometheusClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// REDMetrics holds Rate, Error rate, and Duration percentiles for an app.
type REDMetrics struct {
	RequestRate float64 `json:"request_rate"`
	ErrorRate   float64 `json:"error_rate"`
	LatencyP50  float64 `json:"latency_p50"`
	LatencyP95  float64 `json:"latency_p95"`
	LatencyP99  float64 `json:"latency_p99"`
	HasData     bool    `json:"has_data"`
}

// prometheusResponse is the JSON envelope from the Prometheus instant query API.
type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]any            `json:"value"` // [timestamp, "value"]
		} `json:"result"`
	} `json:"data"`
}

// GetAppREDMetrics queries Prometheus for pre-computed RED metrics for an app.
func (p *PrometheusClient) GetAppREDMetrics(ctx context.Context, appName string) (*REDMetrics, error) {
	metrics := &REDMetrics{}

	queries := map[string]*float64{
		fmt.Sprintf(`microfoundry:http_request_rate:5m{k8s_deployment_name="%s"}`, appName): &metrics.RequestRate,
		fmt.Sprintf(`microfoundry:http_error_rate:5m{k8s_deployment_name="%s"}`, appName):   &metrics.ErrorRate,
		fmt.Sprintf(`microfoundry:http_latency_p50:5m{k8s_deployment_name="%s"}`, appName):  &metrics.LatencyP50,
		fmt.Sprintf(`microfoundry:http_latency_p95:5m{k8s_deployment_name="%s"}`, appName):  &metrics.LatencyP95,
		fmt.Sprintf(`microfoundry:http_latency_p99:5m{k8s_deployment_name="%s"}`, appName):  &metrics.LatencyP99,
	}

	for query, dest := range queries {
		val, err := p.instantQuery(ctx, query)
		if err != nil {
			continue
		}
		if !math.IsNaN(val) {
			*dest = val
			metrics.HasData = true
		}
	}

	return metrics, nil
}

// instantQuery executes a Prometheus instant query and returns the scalar value.
func (p *PrometheusClient) instantQuery(ctx context.Context, query string) (float64, error) {
	u, err := url.Parse(p.BaseURL)
	if err != nil {
		return 0, err
	}
	u.Path = "/api/v1/query"
	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := p.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("prometheus query failed: %s", resp.Status)
	}

	var pr prometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return 0, err
	}

	if pr.Status != "success" || len(pr.Data.Result) == 0 {
		return math.NaN(), nil
	}

	valStr, ok := pr.Data.Result[0].Value[1].(string)
	if !ok {
		return math.NaN(), nil
	}
	return strconv.ParseFloat(valStr, 64)
}

// NewPrometheusClient creates a Prometheus query client.
func NewPrometheusClient(baseURL string) *PrometheusClient {
	return &PrometheusClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}
