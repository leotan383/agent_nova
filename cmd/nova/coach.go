package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var coachStream bool

var coachCmd = &cobra.Command{
	Use:   "coach [章号]",
	Short: "与 AI 讨论已写章节的改稿（分析建议、生成修改稿、写回正文）",
	Long: `进入交互式改稿讨论，类似 nova init --discover。

示例：
  nova coach 3
  nova coach 3 --stream    # 生成修改稿时流式输出到终端

讨论中可用命令：
  /revise  生成修改稿
  /apply   写回正文（自动备份）
  /save    保存修改稿到 .nova/coach/
  /quit    退出`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chapter, err := strconv.Atoi(args[0])
		if err != nil || chapter <= 0 {
			return fmt.Errorf("请指定有效章号")
		}

		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()

		if err := app.RequireAPIKey(actx.Config); err != nil {
			return err
		}

		fmt.Println("=== 章节改稿讨论 ===")
		return workflows.RunCoachSession(context.Background(), actx.Config, actx.Project, actx.Store, chapter, workflows.CoachSessionOptions{
			StreamRevise: coachStream,
		})
	},
}

func init() {
	coachCmd.Flags().BoolVar(&coachStream, "stream", false, "生成修改稿时流式输出到终端")
}
