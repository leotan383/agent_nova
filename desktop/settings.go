package main

import (
	"strings"

	"github.com/tanlian/agent_nova/internal/config"
)

// AppConfigDTO 全局应用配置（API / 模型）。
type AppConfigDTO struct {
	Model      string `json:"model"`
	BaseURL    string `json:"base_url"`
	HasAPIKey  bool   `json:"has_api_key"`
	APIKeyMask string `json:"api_key_mask"`
}

// GetAppConfig 读取当前全局配置。
func (a *App) GetAppConfig() (AppConfigDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return AppConfigDTO{}, err
	}
	return AppConfigDTO{
		Model:      cfg.Model,
		BaseURL:    cfg.OpenAIBaseURL,
		HasAPIKey:  strings.TrimSpace(cfg.OpenAIAPIKey) != "",
		APIKeyMask: maskAPIKey(cfg.OpenAIAPIKey),
	}, nil
}

// SaveAppConfigInput 保存配置；api_key 留空表示不修改已有密钥。
type SaveAppConfigInput struct {
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

// SaveAppConfig 持久化全局配置。
func (a *App) SaveAppConfig(in SaveAppConfigInput) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.Model) != "" {
		cfg.Model = strings.TrimSpace(in.Model)
	}
	cfg.OpenAIBaseURL = strings.TrimSpace(in.BaseURL)
	if strings.TrimSpace(in.APIKey) != "" {
		cfg.OpenAIAPIKey = strings.TrimSpace(in.APIKey)
	}
	return config.Save(cfg)
}

func maskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
