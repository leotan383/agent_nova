package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/status"
)

var statusFocus string

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "创作健康报告",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		rep := status.Build(actx.Project, actx.Store, statusFocus)
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(rep)
		}
		fmt.Printf("《%s》 phase=%s 卷=%d 章=%d 已写=%d 记忆=%d open伏笔=%d\n",
			rep.Title, rep.Phase, rep.CurrentVolume, rep.CurrentChapter, rep.ChapterCount, rep.MemoryCount, rep.OpenForeshadows)
		fmt.Printf("进度: %s / %s (%.1f%%) · 预计还需 %d 章",
			formatWordsCLI(rep.WrittenWords), formatWordsCLI(rep.TargetWords), rep.ProgressPercent, rep.RemainingChapters)
		if rep.AvgWordsPerChapter > 0 {
			fmt.Printf(" · 章均 %d 字", rep.AvgWordsPerChapter)
		}
		fmt.Println()
		for _, u := range rep.Urgent {
			fmt.Printf("⚠ %s\n", u)
		}
		for _, n := range rep.NextSteps {
			fmt.Printf("→ %s\n", n)
		}
		return nil
	},
}

func init() {
	statusCmd.Flags().StringVar(&statusFocus, "focus", "all", "关注点: all|urgency")
}

func formatWordsCLI(n int) string {
	if n >= 10000 && n%10000 == 0 {
		return fmt.Sprintf("%d万字", n/10000)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.1f万字", float64(n)/10000)
	}
	return fmt.Sprintf("%d字", n)
}
