package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/doctor"
)

var preflightCmd = &cobra.Command{
	Use:   "preflight",
	Short: "写章前预检",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		result := map[string]any{"ok": true, "checks": []string{}}
		var issues []string
		if cfg.OpenAIAPIKey == "" {
			issues = append(issues, "OPENAI_API_KEY 未配置")
			result["ok"] = false
		}
		ctx, err := app.LoadContext(projectRoot)
		if err != nil {
			issues = append(issues, err.Error())
			result["ok"] = false
		} else {
			defer ctx.Close()
			rep := doctor.Check(ctx.Project, ctx.Store, false)
			if !rep.OK {
				result["ok"] = false
				for _, f := range rep.Findings {
					if f.Level == "error" {
						issues = append(issues, f.Message)
					}
				}
			}
			result["phase"] = ctx.Project.Meta.Phase
			result["title"] = ctx.Project.Meta.Title
		}
		result["issues"] = issues
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}
		if result["ok"] == true {
			fmt.Println("预检通过")
		} else {
			fmt.Println("预检未通过:")
			for _, i := range issues {
				fmt.Printf("  - %s\n", i)
			}
		}
		return nil
	},
}
