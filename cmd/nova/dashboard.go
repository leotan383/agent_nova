package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/daemon"
)

var dashboardPort int

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "启动只读可视化面板",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		return daemon.RunDashboard(ctx, daemon.Options{
			ProjectRoot: projectRoot,
			Port:        dashboardPort,
		})
	},
}

func init() {
	dashboardCmd.Flags().IntVar(&dashboardPort, "port", 8765, "HTTP 端口")
}
