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

var reviewCmd = &cobra.Command{
	Use:   "review [范围]",
	Short: "审查章节质量",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chapters, err := project.ParseChapterRange(args[0])
		if err != nil {
			return err
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		if err := app.RequireAPIKey(actx.Config); err != nil {
			return err
		}
		wf := workflows.NewReviewWorkflow(actx.Config, actx.Project, actx.Store)
		for _, ch := range chapters {
			rep, err := wf.ReviewChapter(context.Background(), actx.Project, actx.Store, ch)
			if err != nil {
				return err
			}
			if err := rep.Print(outputFmt); err != nil {
				return err
			}
		}
		return nil
	},
}

var reviewShowCmd = &cobra.Command{
	Use:   "show [章号]",
	Short: "查看审查报告",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		chs, err := project.ParseChapterRange(args[0])
		if err != nil {
			return err
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		data, err := os.ReadFile(actx.Project.ReviewPath(chs[0]))
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	reviewCmd.AddCommand(reviewShowCmd)
}
