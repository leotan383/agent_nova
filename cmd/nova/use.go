package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/project"
)

var useCmd = &cobra.Command{
	Use:   "use [path]",
	Short: "绑定当前工作项目",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := project.FindRoot(args[0])
		if err != nil {
			return err
		}
		if err := project.SetCurrentProject(root); err != nil {
			return err
		}
		fmt.Printf("当前项目: %s\n", root)
		return nil
	},
}
