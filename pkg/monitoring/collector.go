package monitoring

import (
	"context"
	"log"
	"time"

	"github.com/younjinjeong/microfoundry/pkg/k8s"
)

// StartCollector runs a background loop that updates gauge metrics
// by querying the K8s API periodically.
func StartCollector(ctx context.Context, metrics *Metrics, clientManager *k8s.ClientManager, interval time.Duration) {
	// Initial collection
	collectMetrics(ctx, metrics, clientManager)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collectMetrics(ctx, metrics, clientManager)
		}
	}
}

func collectMetrics(ctx context.Context, m *Metrics, cm *k8s.ClientManager) {
	client, err := cm.GetActiveClient()
	if err != nil {
		return
	}

	clusterID := cm.GetActive()

	items, err := client.ListAppItems(ctx)
	if err != nil {
		log.Printf("[metrics] error listing apps: %v", err)
		return
	}

	m.AppsTotal.WithLabelValues(clusterID, client.Namespace).Set(float64(len(items)))

	for _, app := range items {
		m.AppsRunning.WithLabelValues(clusterID, client.Namespace, app.Name).Set(float64(app.RunningCount))
		m.AppsDesired.WithLabelValues(clusterID, client.Namespace, app.Name).Set(float64(app.TotalCount))
	}
}
