package outline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestBuildStructureHealth(t *testing.T) {
	root := t.TempDir()
	p := &project.Project{Root: root, Meta: project.Meta{Title: "测试", CurrentChapter: 2}}
	outlineDir := p.OutlineDir()
	if err := os.MkdirAll(outlineDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outline := `### 第1章 · 开篇
> 状态：已完成

### 第2章 · 发展
`
	if err := os.WriteFile(p.VolumeOutlinePath(1), []byte(outline), 0o644); err != nil {
		t.Fatal(err)
	}
	chDir := filepath.Join(p.ChaptersDir(), "第001章-开篇")
	if err := os.MkdirAll(chDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chDir, project.ChapterBodyFile), []byte("正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(p.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.UpsertChapter(store.Chapter{Number: 1, Title: "开篇", WordCount: 2, Status: "draft"}); err != nil {
		t.Fatal(err)
	}

	h, err := BuildStructureHealth(p, st)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Volumes) == 0 {
		t.Fatal("expected volumes")
	}
	if h.Total.Unwritten != 1 {
		t.Fatalf("unwritten = %d, want 1", h.Total.Unwritten)
	}
}
