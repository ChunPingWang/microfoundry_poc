package config

import (
	"fmt"
	"os"
	"path/filepath"

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

type KubernetesConfig struct {
	Context   string `mapstructure:"context"`
	Namespace string `mapstructure:"namespace"`
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

	// Defaults
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
