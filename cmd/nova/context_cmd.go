package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	contextbuilder "github.com/tanlian/agent_nova/internal/context"
)

var (
	ctxChapter int
	ctxVolume  int
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "上下文调试",
}

var contextExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "提取写章上下文",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ctxChapter <= 0 {
			return fmt.Errorf("请指定 --chapter")
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		cb := contextbuilder.Builder{Proj: actx.Project, Store: actx.Store, Config: actx.Config}
		snap, err := cb.Build(ctxChapter, ctxVolume)
		if err != nil {
			return err
		}
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(snap)
		}
		fmt.Print(snap.ToPrompt())
		return nil
	},
}

func init() {
	contextExtractCmd.Flags().IntVar(&ctxChapter, "chapter", 0, "章号")
	contextExtractCmd.Flags().IntVar(&ctxVolume, "volume", 1, "卷号")
	contextCmd.AddCommand(contextExtractCmd)
}
