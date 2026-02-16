package monitoring

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// InstrumentHandler wraps an http.Handler to collect HTTP metrics.
func InstrumentHandler(metrics *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusRecorder{ResponseWriter: w, status: 200}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := classifyPath(r.URL.Path)

		metrics.HTTPRequests.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.status)).Inc()
		metrics.HTTPDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE streaming support.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// classifyPath normalizes URL paths to reduce metric label cardinality.
func classifyPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return "/"
	}

	switch parts[0] {
	case "apps":
		if len(parts) == 1 {
			return "/apps"
		}
		if len(parts) == 2 {
			return "/apps/{name}"
		}
		if len(parts) >= 3 {
			return "/apps/{name}/" + strings.Join(parts[2:], "/")
		}
	case "services":
		if len(parts) == 1 {
			return "/services"
		}
		if len(parts) == 2 {
			return "/services/{name}"
		}
		return "/services/{name}/" + strings.Join(parts[2:], "/")
	case "secrets":
		if len(parts) == 1 {
			return "/secrets"
		}
		if len(parts) == 2 {
			if parts[1] == "new" {
				return "/secrets/new"
			}
			return "/secrets/{name}"
		}
		return "/secrets/{name}/" + strings.Join(parts[2:], "/")
	case "clusters":
		if len(parts) == 1 {
			return "/clusters"
		}
		if len(parts) == 2 {
			return "/clusters/{id}"
		}
		return "/clusters/{id}/" + strings.Join(parts[2:], "/")
	case "topologies":
		if len(parts) <= 2 {
			return "/" + strings.Join(parts, "/")
		}
		return "/topologies/{type}/{plan}"
	case "catalog":
		if len(parts) == 1 {
			return "/catalog"
		}
		return "/catalog/{type}/{plan}"
	case "api":
		if len(parts) >= 3 && parts[1] == "apps" {
			if len(parts) == 3 {
				return "/api/apps/{name}"
			}
			return "/api/apps/{name}/" + strings.Join(parts[3:], "/")
		}
		if len(parts) >= 3 && parts[1] == "services" {
			return "/api/services/{name}"
		}
		if len(parts) >= 3 && parts[1] == "clusters" {
			return "/api/clusters/{id}"
		}
		if len(parts) >= 3 && parts[1] == "secrets" {
			return "/api/secrets/{name}"
		}
		if len(parts) >= 3 && parts[1] == "topologies" {
			return "/api/topologies/{type}/{plan}"
		}
		return "/" + strings.Join(parts, "/")
	case "static":
		return "/static"
	case "metrics":
		return "/metrics"
	case "monitoring":
		return "/" + strings.Join(parts, "/")
	}

	return "/" + parts[0]
}
