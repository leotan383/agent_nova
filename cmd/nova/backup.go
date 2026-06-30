package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/backup"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "备份管理",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create [label]",
	Short: "创建备份",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		label := "manual"
		if len(args) > 0 {
			label = args[0]
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		if err := backup.Create(actx.Project, label); err != nil {
			return err
		}
		fmt.Println("备份已创建")
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出备份",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		items, err := backup.List(actx.Project)
		if err != nil {
			return err
		}
		for _, i := range items {
			fmt.Println(i)
		}
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore [name]",
	Short: "恢复备份",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		if err := backup.Restore(actx.Project, args[0]); err != nil {
			return err
		}
		fmt.Println("备份已恢复")
		return nil
	},
}

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
}
