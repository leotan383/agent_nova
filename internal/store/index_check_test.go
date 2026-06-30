package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/index"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestCheckIndexStaleDuplicateChapterFiles(t *testing.T) {
	dir := t.TempDir()
	chDir := filepath.Join(dir, "正文")
	if err := os.MkdirAll(chDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// 同章两个文件（润色前后常见）
	_ = os.WriteFile(filepath.Join(chDir, "第001章-退婚.md"), []byte("chapter v1"), 0o644)
	_ = os.WriteFile(filepath.Join(chDir, "第001章.md"), []byte("chapter v2 longer"), 0o644)

	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	p := &project.Project{Root: dir}
	idx := index.New(p, st)
	if err := idx.RebuildChapters(0); err != nil {
		t.Fatal(err)
	}

	rep := st.CheckIndexStale(chDir)
	if rep.Stale {
		t.Fatalf("expected not stale with duplicate same-chapter files, issues: %v", rep.Issues)
	}
	if rep.FileCount != 1 {
		t.Fatalf("FileCount=%d want 1 unique chapter", rep.FileCount)
	}
}
