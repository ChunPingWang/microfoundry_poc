package developer

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Builder handles Docker image building.
type Builder struct {
	registry string // e.g., "localhost:5000" or "ghcr.io/younjinjeong"
}

// NewBuilder creates a new Builder.
func NewBuilder(registry string) *Builder {
	if registry == "" {
		registry = "localhost:5000"
	}
	return &Builder{registry: registry}
}

// BuildImage builds a Docker image from the given directory.
func (b *Builder) BuildImage(ctx context.Context, workDir, appName, tag string) (string, error) {
	if tag == "" {
		tag = "latest"
	}
	image := fmt.Sprintf("%s/%s:%s", b.registry, appName, tag)

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", image, ".")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker build failed: %w\noutput: %s", err, string(out))
	}

	return image, nil
}

// PushImage pushes a Docker image to the registry.
func (b *Builder) PushImage(ctx context.Context, image string) error {
	// For local Docker Desktop k8s, push is not needed (images are shared).
	// Only push to remote registries.
	if strings.HasPrefix(b.registry, "localhost") {
		return nil
	}

	cmd := exec.CommandContext(ctx, "docker", "push", image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker push failed: %w\noutput: %s", err, string(out))
	}
	return nil
}
