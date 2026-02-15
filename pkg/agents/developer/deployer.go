package developer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Deployer handles deploying to Kubernetes via kubectl.
type Deployer struct {
	kubeContext string
	namespace   string
}

// NewDeployer creates a new Deployer.
func NewDeployer(kubeContext, namespace string) *Deployer {
	if kubeContext == "" {
		kubeContext = "docker-desktop"
	}
	if namespace == "" {
		namespace = "microfoundry"
	}
	return &Deployer{kubeContext: kubeContext, namespace: namespace}
}

// EnsureNamespace creates the namespace if it doesn't exist.
func (d *Deployer) EnsureNamespace(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "kubectl",
		"--context", d.kubeContext,
		"create", "namespace", d.namespace,
		"--dry-run=client", "-o", "yaml",
	)
	yamlOut, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("generating namespace yaml: %w", err)
	}

	apply := exec.CommandContext(ctx, "kubectl",
		"--context", d.kubeContext,
		"apply", "-f", "-",
	)
	apply.Stdin = strings.NewReader(string(yamlOut))
	out, err := apply.CombinedOutput()
	if err != nil {
		return fmt.Errorf("applying namespace: %w\noutput: %s", err, string(out))
	}
	return nil
}

// Deploy applies a k8s manifest to the cluster.
func (d *Deployer) Deploy(ctx context.Context, manifestYAML string) error {
	cmd := exec.CommandContext(ctx, "kubectl",
		"--context", d.kubeContext,
		"--namespace", d.namespace,
		"apply", "-f", "-",
	)
	cmd.Stdin = strings.NewReader(manifestYAML)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w\noutput: %s", err, string(out))
	}
	return nil
}

// WaitForRollout waits for a deployment to complete.
func (d *Deployer) WaitForRollout(ctx context.Context, deploymentName string) error {
	cmd := exec.CommandContext(ctx, "kubectl",
		"--context", d.kubeContext,
		"--namespace", d.namespace,
		"rollout", "status", "deployment/"+deploymentName,
		"--timeout=120s",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rollout wait failed: %w\noutput: %s", err, string(out))
	}
	return nil
}
