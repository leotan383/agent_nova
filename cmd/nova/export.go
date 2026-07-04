package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/export"
)

var (
	exportOut    string
	exportFormat string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "导出合集（markdown|epub|txt|pdf）",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		format := strings.ToLower(exportFormat)
		out := exportOut
		switch format {
		case "epub":
			if out == "" {
				out = filepath.Join(actx.Project.Root, actx.Project.Meta.Title+".epub")
			}
			if err := export.WriteEPUB(actx.Project, out, export.Options{}); err != nil {
				return err
			}
		case "txt":
			if out == "" {
				out = filepath.Join(actx.Project.Root, actx.Project.Meta.Title+".txt")
			}
			if err := export.WriteTXT(actx.Project, out, export.Options{}); err != nil {
				return err
			}
		case "pdf":
			if out == "" {
				out = filepath.Join(actx.Project.Root, actx.Project.Meta.Title+".pdf")
			}
			if err := export.WritePDF(actx.Project, out, export.Options{}); err != nil {
				return err
			}
		default:
			if out == "" {
				out = filepath.Join(actx.Project.Root, "export.md")
			}
			if err := export.WriteMarkdown(actx.Project, out, export.Options{}); err != nil {
				return err
			}
		}
		fmt.Printf("已导出: %s\n", out)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportOut, "out", "", "输出路径")
	exportCmd.Flags().StringVar(&exportFormat, "format", "markdown", "格式: markdown|epub|txt|pdf")
}
