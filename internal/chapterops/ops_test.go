package chapterops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestInsertAndDeleteChapter(t *testing.T) {
	root := t.TempDir()
	p := &project.Project{Root: root, Meta: project.Meta{Title: "测试"}}
	if err := os.MkdirAll(p.ChaptersDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	for i, title := range []string{"开篇", "发展", "高潮"} {
		dir := filepath.Join(p.ChaptersDir(), project.ChapterDirName(i+1, title))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		body := filepath.Join(dir, project.ChapterBodyFile)
		if err := os.WriteFile(body, []byte("# 标题\n\n正文"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st, err := store.Open(p.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.UpsertChapter(store.Chapter{Number: 1, Title: "开篇", WordCount: 2}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertChapter(store.Chapter{Number: 2, Title: "发展", WordCount: 2}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertChapter(store.Chapter{Number: 3, Title: "高潮", WordCount: 2}); err != nil {
		t.Fatal(err)
	}

	newNum, err := InsertAfter(p, st, 1, "插曲")
	if err != nil {
		t.Fatal(err)
	}
	if newNum != 2 {
		t.Fatalf("new chapter = %d", newNum)
	}
	nums, err := p.ListChapterNumbers()
	if err != nil {
		t.Fatal(err)
	}
	if len(nums) != 4 {
		t.Fatalf("want 4 chapters, got %v", nums)
	}

	if err := DeleteChapter(p, st, 2); err != nil {
		t.Fatal(err)
	}
	nums, err = p.ListChapterNumbers()
	if err != nil {
		t.Fatal(err)
	}
	if len(nums) != 3 {
		t.Fatalf("after delete want 3, got %v", nums)
	}
}
