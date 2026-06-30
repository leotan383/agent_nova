package main

import (
	"github.com/spf13/cobra"
)

var (
	projectRoot string
	outputFmt   string
	debugMode   bool
)

var rootCmd = &cobra.Command{
	Use:   "nova",
	Short: "命令行网文创作 Agent",
	Long:  "nova — 长篇网文创作一条龙：初始化、规划、写章、审查、记忆、查询与可视化。",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debugMode {
			// logger wired in commands as needed
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectRoot, "project", "p", "", "小说项目根目录（含 nova.yaml）")
	rootCmd.PersistentFlags().StringVar(&outputFmt, "format", "text", "输出格式: text|json")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "调试日志")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(preflightCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(writeCmd)
	rootCmd.AddCommand(coachCmd)
	rootCmd.AddCommand(reviewCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(memoryCmd)
	rootCmd.AddCommand(learnCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(gateCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(dashboardCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(exportCmd)
}
