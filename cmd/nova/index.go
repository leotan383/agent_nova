package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/index"
)

var (
	indexChapter int
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "索引管理",
}

var indexStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "索引统计",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		idx := index.New(actx.Project, actx.Store)
		cFTS, sFTS, chs, err := idx.Stats()
		if err != nil {
			return err
		}
		out := map[string]int{"chapter_fts": cFTS, "setting_fts": sFTS, "chapters": chs}
		if outputFmt == "json" {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
		}
		fmt.Printf("章节 FTS: %d, 设定 FTS: %d, 章节记录: %d\n", cFTS, sFTS, chs)
		return nil
	},
}

var indexRebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "重建索引",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		idx := index.New(actx.Project, actx.Store)
		if indexChapter > 0 {
			if err := idx.RebuildChapters(indexChapter); err != nil {
				return err
			}
		} else {
			if err := idx.RebuildAll(); err != nil {
				return err
			}
		}
		fmt.Println("索引重建完成")
		return nil
	},
}

func init() {
	indexRebuildCmd.Flags().IntVar(&indexChapter, "chapter", 0, "仅重建指定章")
	indexCmd.AddCommand(indexStatsCmd)
	indexCmd.AddCommand(indexRebuildCmd)
	indexCmd.AddCommand(indexEmbedCmd)
}

var indexEmbedCmd = &cobra.Command{
	Use:   "embed",
	Short: "重建向量索引（需 OPENAI_API_KEY）",
	RunE: func(cmd *cobra.Command, args []string) error {
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		if err := app.RequireAPIKey(actx.Config); err != nil {
			return err
		}
		idx := index.New(actx.Project, actx.Store)
		n, err := idx.RebuildEmbeddings(context.Background(), actx.Config)
		if err != nil {
			return err
		}
		fmt.Printf("向量索引完成: %d 条\n", n)
		return nil
	},
}
