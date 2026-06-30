package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/paths"
	"gopkg.in/yaml.v3"
)

const (
	DefaultModel = "gpt-4o"
	EnvAPIKey    = "OPENAI_API_KEY"
	EnvBaseURL   = "OPENAI_BASE_URL"
	EnvModel     = "NOVA_MODEL"
)

type Config struct {
	OpenAIAPIKey  string `yaml:"openai_api_key"`
	OpenAIBaseURL string `yaml:"openai_base_url"`
	Model         string `yaml:"model"`
	Debug         bool   `yaml:"debug"`
}

func Load() (*Config, error) {
	cfg := &Config{Model: DefaultModel}
	loadDotEnv()
	if v := os.Getenv(EnvAPIKey); v != "" {
		cfg.OpenAIAPIKey = v
	}
	if v := os.Getenv(EnvBaseURL); v != "" {
		cfg.OpenAIBaseURL = v
	}
	if v := os.Getenv(EnvModel); v != "" {
		cfg.Model = v
	}
	layout := paths.Global()
	data, err := os.ReadFile(layout.ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.OpenAIAPIKey == "" {
		if v := os.Getenv(EnvAPIKey); v != "" {
			cfg.OpenAIAPIKey = v
		}
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	layout := paths.Global()
	if err := os.MkdirAll(layout.ConfigDir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(layout.ConfigFile(), data, 0o600)
}

func loadDotEnv() {
	for _, name := range []string{".env", filepath.Join(paths.Global().ConfigDir, ".env")} {
		loadEnvFile(name)
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}
