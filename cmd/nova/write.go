package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var (
	writeResume        bool // 断点续写，读取 .nova/run_ledger.json（--resume）
	writeStream        bool // 流式输出正文到终端（--stream）
	writeVolume        int  // 指定卷号，默认取 nova.yaml 当前卷（--volume）
	writeContinueOnErr bool // 批量写章时单章失败不中断（--continue-on-error）
)

var writeCmd = &cobra.Command{
	Use:   "write [章号]",
	Short: "写章（上下文→起草→审查→润色→摘要→索引）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 解析章号范围
		chapters, err := project.ParseChapterRange(args[0])
		if err != nil {
			return err
		}

		// 加载项目上下文
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()

		// 检查 API Key 是否配置
		if err := app.RequireAPIKey(actx.Config); err != nil {
			return err
		}

		// 创建写章工作流
		wf := workflows.NewWriteWorkflow(actx.Config, actx.Project, actx.Store)

		// 获取卷号
		vol := writeVolume
		if vol <= 0 {
			vol = actx.Project.Meta.CurrentVolume
			if vol <= 0 {
				vol = 1
			}
		}
		var lastErr error
		for _, ch := range chapters {
			fmt.Fprintf(os.Stderr, "正在写第 %d 章...\n", ch)
			rep, err := wf.WriteChapter(context.Background(), actx.Project, actx.Store, workflows.WriteOptions{
				Chapter: ch, Volume: vol, Resume: writeResume, Stream: writeStream,
				OnDelta: func(s string) error {
					if writeStream {
						_, err := os.Stdout.WriteString(s)
						return err
					}
					return nil
				},
			})
			if err != nil {
				lastErr = err
				fmt.Fprintf(os.Stderr, "第 %d 章失败: %v\n", ch, err)
				if !writeContinueOnErr {
					return err
				}
				continue
			}
			if rep.Status == report.StatusNeedsAction || rep.Status == report.StatusFailed {
				fmt.Fprintf(os.Stderr, "第 %d 章未完成: %s\n", ch, rep.Summary)
				if !writeContinueOnErr {
					return rep.Print(outputFmt)
				}
			}
			if err := rep.Print(outputFmt); err != nil {
				return err
			}
		}
		return lastErr
	},
}

func init() {
	writeCmd.Flags().BoolVar(&writeResume, "resume", false, "断点续写")
	writeCmd.Flags().BoolVar(&writeStream, "stream", false, "流式输出正文")
	writeCmd.Flags().IntVar(&writeVolume, "volume", 0, "卷号")
	writeCmd.Flags().BoolVar(&writeContinueOnErr, "continue-on-error", false, "批量写章时遇错继续")
}
