package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/pipeline"
)

var (
	gateChapter int
	gateStage   string
)

var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "写章边界校验",
	RunE: func(cmd *cobra.Command, args []string) error {
		if gateChapter <= 0 {
			return fmt.Errorf("请指定 --chapter")
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		res := pipeline.RunGate(actx.Project, actx.Store, gateChapter, pipeline.GateStage(gateStage))
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(res)
		}
		if res.OK {
			fmt.Printf("gate %s 第%d章: 通过\n", gateStage, gateChapter)
		} else {
			fmt.Printf("gate %s 第%d章: 未通过\n", gateStage, gateChapter)
			for _, i := range res.Issues {
				fmt.Printf("  - %s\n", i)
			}
		}
		return nil
	},
}

func init() {
	gateCmd.Flags().IntVar(&gateChapter, "chapter", 0, "章号")
	gateCmd.Flags().StringVar(&gateStage, "stage", "prewrite", "阶段: prewrite|precommit|postcommit")
}
