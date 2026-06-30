package index

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

type Indexer struct {
	proj  *project.Project
	store *store.Store
}

func New(proj *project.Project, st *store.Store) *Indexer {
	return &Indexer{proj: proj, store: st}
}

func (idx *Indexer) RebuildAll() error {
	if err := idx.rebuildSettings(); err != nil {
		return err
	}
	return idx.RebuildChapters(0)
}

func (idx *Indexer) RebuildChapters(chapterNum int) error {
	entries, err := os.ReadDir(idx.proj.ChaptersDir())
	if err != nil {
		return err
	}
	seen := map[int]struct{}{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		num := parseChapterNum(e.Name())
		if num <= 0 {
			continue
		}
		if chapterNum > 0 && num != chapterNum {
			continue
		}
		seen[num] = struct{}{}
	}
	for num := range seen {
		path, title, err := idx.proj.FindChapterFile(num)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if title == "" {
			title = extractTitle(filepath.Base(path))
		}
		wordCount := utf8.RuneCountInString(string(data))
		status := "draft"
		if existing, err := idx.store.GetChapter(num); err == nil && existing.Status != "" {
			status = existing.Status
		}
		summaryPath := idx.proj.SummaryPath(num)
		if err := idx.store.IndexChapterFTS(num, title, string(data)); err != nil {
			return err
		}
		info, err := os.Stat(path)
		updatedAt := project.Timestamp()
		if err == nil {
			updatedAt = info.ModTime().UTC().Truncate(time.Second).Format(time.RFC3339)
		}
		_ = idx.store.UpsertChapter(store.Chapter{
			Number: num, Title: title, WordCount: wordCount, Path: path,
			SummaryPath: summaryPath, Status: status, UpdatedAt: updatedAt,
		})
	}
	return nil
}

func (idx *Indexer) rebuildSettings() error {
	return filepath.Walk(idx.proj.SettingsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(idx.proj.Root, path)
		return idx.store.IndexSettingFTS(rel, string(data))
	})
}

func parseChapterNum(name string) int {
	var n int
	for _, r := range name {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		} else if n > 0 {
			break
		}
	}
	return n
}

func extractTitle(name string) string {
	base := strings.TrimSuffix(name, ".md")
	if i := strings.Index(base, "-"); i > 0 {
		return base[i+1:]
	}
	return base
}

func (idx *Indexer) Stats() (chapterFTS, settingFTS int, chapters int, err error) {
	chapterFTS, settingFTS, err = idx.store.FTSStats()
	if err != nil {
		return
	}
	list, err := idx.store.ListChapters()
	if err != nil {
		return
	}
	chapters = len(list)
	return
}
