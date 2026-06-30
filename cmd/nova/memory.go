package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	memorypkg "github.com/tanlian/agent_nova/internal/memory"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "长期记忆管理",
}

var memoryStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "记忆统计",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		stats, total, err := actx.Store.MemoryStats()
		if err != nil {
			return err
		}
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"total": total, "by_category": stats})
		}
		fmt.Printf("总计: %d\n", total)
		for k, v := range stats {
			fmt.Printf("  %s: %d\n", k, v)
		}
		return nil
	},
}

var (
	memQueryCategory string
	memQuerySubject  string
)

var memoryQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "查询记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		items, err := actx.Store.QueryMemories(memQueryCategory, memQuerySubject, 50)
		if err != nil {
			return err
		}
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
		}
		for _, m := range items {
			fmt.Printf("[%s/%s] ch=%d %s\n", m.Category, m.Subject, m.SourceChapter, m.Content)
		}
		return nil
	},
}

var memoryDumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "导出全部记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		items, err := actx.Store.DumpMemories()
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
	},
}

func init() {
	memoryQueryCmd.Flags().StringVar(&memQueryCategory, "category", "", "分类")
	memoryQueryCmd.Flags().StringVar(&memQuerySubject, "subject", "", "主题")
	memoryCmd.AddCommand(memoryStatsCmd)
	memoryCmd.AddCommand(memoryQueryCmd)
	memoryCmd.AddCommand(memoryDumpCmd)
	memoryCmd.AddCommand(memoryBootstrapCmd)
	memoryCmd.AddCommand(memoryConflictsCmd)
}

var memoryBootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "从设定集回填初始长期记忆",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		n, err := memorypkg.BootstrapFromSettings(actx.Project, actx.Store)
		if err != nil {
			return err
		}
		fmt.Printf("已回填 %d 条记忆\n", n)
		return nil
	},
}

var memoryConflictsCmd = &cobra.Command{
	Use:   "conflicts",
	Short: "查看同 subject 的 active 记忆冲突",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		conflicts, err := actx.Store.FindMemoryConflicts()
		if err != nil {
			return err
		}
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(conflicts)
		}
		if len(conflicts) == 0 {
			fmt.Println("无冲突")
			return nil
		}
		for _, c := range conflicts {
			fmt.Printf("subject=%s count=%d\n", c.Subject, c.Count)
			for _, m := range c.Memories {
				fmt.Printf("  [%s] ch=%d %s\n", m.Category, m.SourceChapter, m.Content)
			}
		}
		return nil
	},
}
