package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var (
	queryForeshadowStatus string
)

var queryCmd = &cobra.Command{
	Use:   "query [关键词]",
	Short: "查询角色、伏笔、章节片段",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		keyword := ""
		if len(args) > 0 {
			keyword = args[0]
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		res, err := workflows.Query(actx.Store, keyword, queryForeshadowStatus)
		if err != nil {
			return err
		}
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
		}
		for _, e := range res.Entities {
			fmt.Printf("[实体/%s] %s: %s\n", e.Type, e.Name, e.StateJSON)
		}
		for _, f := range res.Foreshadows {
			fmt.Printf("[伏笔/%s] 第%d章: %s\n", f.Status, f.PlantedChapter, f.Description)
		}
		for _, h := range res.FTS {
			fmt.Printf("[检索/%s] %s %s\n", h["kind"], h["title"], h["snippet"])
		}
		for _, m := range res.Memories {
			fmt.Printf("[记忆/%s] %s: %s\n", m.Category, m.Subject, m.Content)
		}
		for _, cp := range res.CoolPoints {
			delivered := "未兑现"
			if cp.Delivered {
				delivered = "已兑现"
			}
			fmt.Printf("[爽点/%s] 第%d章 %s: %s\n", cp.Type, cp.Chapter, delivered, cp.Description)
		}
		if len(res.Entities) == 0 && len(res.Foreshadows) == 0 && len(res.FTS) == 0 && len(res.Memories) == 0 && len(res.CoolPoints) == 0 {
			fmt.Println("未找到匹配结果")
		}
		return nil
	},
}

func init() {
	queryCmd.Flags().StringVar(&queryForeshadowStatus, "status", "", "伏笔状态: open|resolved")
}
