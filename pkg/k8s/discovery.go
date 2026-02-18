package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/younjinjeong/microfoundry/pkg/models"
)

// DiscoverPlatformServices checks K8s for each predefined platform service,
// detects existing ingresses, and merges stored URL overrides.
func (c *Client) DiscoverPlatformServices(ctx context.Context, domain string, overrides map[string]string) []models.ServiceEndpoint {
	results := make([]models.ServiceEndpoint, len(models.PlatformServices))

	for i, svc := range models.PlatformServices {
		ep := svc // copy template

		// Resolve namespace: empty means use the client's app namespace
		ns := svc.Namespace
		if ns == "" {
			ns = c.Namespace
		}

		ep.InternalURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d",
			svc.ServiceName, ns, svc.ServicePort)

		// Check if an Ingress exists for this service
		ingress, err := c.Clientset.NetworkingV1().Ingresses(ns).Get(
			ctx, svc.Name, metav1.GetOptions{})
		if err == nil && len(ingress.Spec.Rules) > 0 {
			ep.HasIngress = true
			ep.IngressHost = ingress.Spec.Rules[0].Host
		}

		// Apply user override if present
		if url, ok := overrides[svc.Name]; ok && url != "" {
			ep.OverrideURL = url
		}

		results[i] = ep
	}
	return results
}
