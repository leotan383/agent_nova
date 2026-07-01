package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/rag"
)

func (idx *Indexer) RebuildEmbeddings(ctx context.Context, cfg *config.Config) (int, error) {
	if cfg.OpenAIAPIKey == "" {
		return 0, nil
	}
	ragIdx := rag.NewIndexer(cfg)
	count := 0
	entries, err := os.ReadDir(idx.proj.ChaptersDir())
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(idx.proj.ChaptersDir(), e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return count, err
		}
		num := parseChapterNum(e.Name())
		ref := fmt.Sprintf("%d", num)
		if err := ragIdx.IndexText(ctx, idx.store, "chapter", ref, string(data)); err != nil {
			return count, err
		}
		count++
	}
	err = filepath.Walk(idx.proj.SettingsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(idx.proj.Root, path)
		if err := ragIdx.IndexText(ctx, idx.store, "setting", rel, string(data)); err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		return count, err
	}
	memories, err := idx.store.ListActiveMemories(5000)
	if err != nil {
		return count, err
	}
	for _, m := range memories {
		text := m.Subject + "\n" + m.Content
		if err := ragIdx.IndexText(ctx, idx.store, "memory", m.ID, text); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
