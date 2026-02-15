package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the mf CLI.
type Config struct {
	Anthropic  AnthropicConfig  `mapstructure:"anthropic"`
	GitHub     GitHubConfig     `mapstructure:"github"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	Pipeline   PipelineConfig   `mapstructure:"pipeline"`
	Agents     AgentsConfig     `mapstructure:"agents"`
}

type AnthropicConfig struct {
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	MaxTokens      int64  `mapstructure:"max_tokens"`
	ThinkingBudget int64  `mapstructure:"thinking_budget"`
}

type GitHubConfig struct {
	Owner string `mapstructure:"owner"`
	Repo  string `mapstructure:"repo"`
}

type KubernetesConfig struct {
	Context   string `mapstructure:"context"`
	Namespace string `mapstructure:"namespace"`
}

type PipelineConfig struct {
	StopOnFailure bool     `mapstructure:"stop_on_failure"`
	RefCodebases  []string `mapstructure:"ref_codebases"`
}

type AgentsConfig struct {
	Analyzer     AgentToggle     `mapstructure:"analyzer"`
	DataEngineer AgentToggle     `mapstructure:"data_engineer"`
	Designer     DesignerConfig  `mapstructure:"designer"`
	Developer    DeveloperConfig `mapstructure:"developer"`
	Reviewer     ReviewerConfig  `mapstructure:"reviewer"`
}

type AgentToggle struct {
	Enabled bool `mapstructure:"enabled"`
}

type DesignerConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	SkipIfNoUI bool `mapstructure:"skip_if_no_ui"`
}

type DeveloperConfig struct {
	Enabled        bool `mapstructure:"enabled"`
	BuildContainer bool `mapstructure:"build_container"`
	DeployToK8s    bool `mapstructure:"deploy_to_k8s"`
	RunTests       bool `mapstructure:"run_tests"`
}

type ReviewerConfig struct {
	Enabled bool         `mapstructure:"enabled"`
	Checks  ChecksConfig `mapstructure:"checks"`
}

type ChecksConfig struct {
	License     bool `mapstructure:"license"`
	Security    bool `mapstructure:"security"`
	Performance bool `mapstructure:"performance"`
	Cost        bool `mapstructure:"cost"`
	Sizing      bool `mapstructure:"sizing"`
}

// Load reads configuration from files, env vars, and returns a Config.
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
	v.SetDefault("anthropic.model", "claude-sonnet-4-5-20250929")
	v.SetDefault("anthropic.max_tokens", 8192)
	v.SetDefault("anthropic.thinking_budget", 10000)
	v.SetDefault("github.owner", "younjinjeong")
	v.SetDefault("github.repo", "microfoundry")
	v.SetDefault("kubernetes.context", "docker-desktop")
	v.SetDefault("kubernetes.namespace", "microfoundry")
	v.SetDefault("pipeline.stop_on_failure", true)
	v.SetDefault("pipeline.ref_codebases", []string{"./cli", "./cf-deployment"})
	v.SetDefault("agents.analyzer.enabled", true)
	v.SetDefault("agents.data_engineer.enabled", true)
	v.SetDefault("agents.designer.enabled", true)
	v.SetDefault("agents.designer.skip_if_no_ui", true)
	v.SetDefault("agents.developer.enabled", true)
	v.SetDefault("agents.developer.build_container", true)
	v.SetDefault("agents.developer.deploy_to_k8s", true)
	v.SetDefault("agents.developer.run_tests", true)
	v.SetDefault("agents.reviewer.enabled", true)
	v.SetDefault("agents.reviewer.checks.license", true)
	v.SetDefault("agents.reviewer.checks.security", true)
	v.SetDefault("agents.reviewer.checks.performance", true)
	v.SetDefault("agents.reviewer.checks.cost", true)
	v.SetDefault("agents.reviewer.checks.sizing", true)

	// Read config file (optional — not an error if missing)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	// Bind specific env vars
	v.BindEnv("anthropic.api_key", "ANTHROPIC_API_KEY", "MF_ANTHROPIC_API_KEY")

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

// SetValue writes a config key-value pair to the user config file.
func SetValue(key, value string) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)

	// Read existing config
	_ = v.ReadInConfig()

	v.Set(key, value)
	return v.WriteConfigAs(filepath.Join(dir, "config.yaml"))
}

// GetValue reads a config value by key.
func GetValue(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	v := viper.New()
	v.SetConfigName("mf")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".mf"))
	}
	_ = v.ReadInConfig()

	// Also check the loaded config for known keys
	_ = cfg

	return v.GetString(key), nil
}
