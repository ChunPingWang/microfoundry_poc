package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for MicroFoundry.
type Config struct {
	GitHub     GitHubConfig     `mapstructure:"github"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
}

type GitHubConfig struct {
	Owner string `mapstructure:"owner"`
	Repo  string `mapstructure:"repo"`
}

// ClusterConfig represents a registered Kubernetes cluster.
type ClusterConfig struct {
	Name      string `mapstructure:"name"      json:"name"`
	Context   string `mapstructure:"context"   json:"context"`
	Namespace string `mapstructure:"namespace" json:"namespace"`
	Domain    string `mapstructure:"domain"    json:"domain"`
	Provider  string `mapstructure:"provider"  json:"provider"`
	Region    string `mapstructure:"region"    json:"region,omitempty"`
	Enabled   bool   `mapstructure:"enabled"   json:"enabled"`
}

type KubernetesConfig struct {
	// Multi-cluster support
	Clusters map[string]ClusterConfig `mapstructure:"clusters"`
	Active   string                   `mapstructure:"active"`

	// Legacy single-cluster fields (for migration)
	Context   string `mapstructure:"context"`
	Namespace string `mapstructure:"namespace"`
}

// Migrate converts legacy single-cluster config to multi-cluster format.
func (k *KubernetesConfig) Migrate() {
	if len(k.Clusters) > 0 {
		return // Already new format
	}

	ctx := k.Context
	if ctx == "" {
		ctx = "docker-desktop"
	}
	ns := k.Namespace
	if ns == "" {
		ns = "microfoundry"
	}

	k.Clusters = map[string]ClusterConfig{
		ctx: {
			Name:      ctx,
			Context:   ctx,
			Namespace: ns,
			Domain:    "cf-local.dev",
			Provider:  DetectProvider(ctx),
			Enabled:   true,
		},
	}
	k.Active = ctx
}

// GetActiveCluster returns the active cluster config.
func (k *KubernetesConfig) GetActiveCluster() (string, ClusterConfig, bool) {
	if k.Active == "" {
		return "", ClusterConfig{}, false
	}
	cfg, ok := k.Clusters[k.Active]
	return k.Active, cfg, ok
}

// DetectProvider guesses the provider from a kubeconfig context name.
func DetectProvider(context string) string {
	lower := strings.ToLower(context)
	switch {
	case strings.Contains(lower, "docker-desktop"):
		return "docker-desktop"
	case strings.Contains(lower, "eks") || strings.Contains(lower, "aws"):
		return "eks"
	case strings.Contains(lower, "gke") || strings.Contains(lower, "gke_"):
		return "gke"
	case strings.Contains(lower, "aks") || strings.Contains(lower, "azure"):
		return "aks"
	default:
		return "native"
	}
}

// Load reads configuration from files and env vars and returns a Config.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("mf")
	v.SetConfigType("yaml")

	// Search paths: project dir, then user home
	v.AddConfigPath("./configs")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".mf"))
	}

	// Environment variable bindings
	v.SetEnvPrefix("MF")
	v.AutomaticEnv()

	// Defaults (legacy format)
	v.SetDefault("github.owner", "younjinjeong")
	v.SetDefault("github.repo", "microfoundry")
	v.SetDefault("kubernetes.context", "docker-desktop")
	v.SetDefault("kubernetes.namespace", "microfoundry")

	// Read config file (optional — not an error if missing)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	// Auto-migrate legacy config to multi-cluster format
	cfg.Kubernetes.Migrate()

	return &cfg, nil
}

// ConfigDir returns the path to the user's mf config directory.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".mf")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
