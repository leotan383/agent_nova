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
	nums, err := idx.proj.ListChapterNumbers()
	if err != nil {
		return 0, err
	}
	for _, num := range nums {
		path, _, err := idx.proj.FindChapterFile(num)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return count, err
		}
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
