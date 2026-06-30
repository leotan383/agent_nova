package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var learnCmd = &cobra.Command{
	Use:   "learn [内容]",
	Short: "沉淀写作模式到长期记忆",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		content := args[0]
		if len(args) > 1 {
			for _, a := range args[1:] {
				content += " " + a
			}
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		if err := app.RequireAPIKey(actx.Config); err != nil {
			return err
		}
		wf := workflows.NewLearnWorkflow(actx.Config, actx.Project, actx.Store)
		ch := actx.Project.Meta.CurrentChapter
		rep, err := wf.Learn(context.Background(), actx.Store, content, ch)
		if err != nil {
			return err
		}
		return rep.Print(outputFmt)
	},
}
