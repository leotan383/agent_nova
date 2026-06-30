package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var planCmd = &cobra.Command{
	Use:   "plan [卷号]",
	Short: "生成卷纲与章纲",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPlan,
}

var planShowCmd = &cobra.Command{
	Use:   "show [卷号]",
	Short: "查看卷纲",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vols, err := project.ParseVolumeRange(args[0])
		if err != nil {
			return err
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		path := actx.Project.VolumeOutlinePath(vols[0])
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	},
}

func runPlan(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请指定卷号，如: nova plan 1")
	}
	vols, err := project.ParseVolumeRange(args[0])
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

	// 调用 LLM 规划卷
	wf := workflows.NewPlanWorkflow(actx.Config, actx.Project, actx.Store)
	for _, vol := range vols {
		rep, err := wf.PlanVolume(context.Background(), actx.Project, vol)
		if err != nil {
			return err
		}
		if err := rep.Print(outputFmt); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	planCmd.AddCommand(planShowCmd)
}
