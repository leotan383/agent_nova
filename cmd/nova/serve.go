package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/daemon"
)

var (
	servePort   int
	serveSocket string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "启动后台 Daemon（API + 可选 Dashboard）",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		return daemon.RunDaemon(ctx, daemon.Options{
			ProjectRoot: projectRoot,
			Port:        servePort,
			SocketPath:  serveSocket,
		})
	},
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8787, "HTTP 端口")
	serveCmd.Flags().StringVar(&serveSocket, "socket", "", "Unix socket 路径")
}
