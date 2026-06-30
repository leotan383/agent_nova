package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/doctor"
)

var doctorDeep bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "项目体检",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer ctx.Close()
		rep := doctor.Check(ctx.Project, ctx.Store, doctorDeep)
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(rep)
		}
		fmt.Printf("阶段: %s\n", rep.Phase)
		for _, f := range rep.Findings {
			fmt.Printf("[%s] %s", f.Level, f.Message)
			if f.Fix != "" {
				fmt.Printf(" → %s", f.Fix)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorDeep, "deep", false, "深度检查")
}
