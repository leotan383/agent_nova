package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("nova %s\n", Version)
	},
}
