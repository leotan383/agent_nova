package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理全局配置",
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "设置配置项 (api_key|base_url|model)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		switch args[0] {
		case "api_key":
			cfg.OpenAIAPIKey = args[1]
		case "base_url":
			cfg.OpenAIBaseURL = args[1]
		case "model":
			cfg.Model = args[1]
		default:
			return fmt.Errorf("未知配置项: %s", args[0])
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Println("配置已保存")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		key := cfg.OpenAIAPIKey
		if len(key) > 8 {
			key = key[:4] + "..." + key[len(key)-4:]
		}
		fmt.Printf("model: %s\nbase_url: %s\napi_key: %s\n", cfg.Model, cfg.OpenAIBaseURL, key)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configShowCmd)
}
