package outline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestBuildMatrix(t *testing.T) {
	root := t.TempDir()
	p := &project.Project{Root: root, Meta: project.Meta{Title: "测试"}}
	outlineDir := p.OutlineDir()
	if err := os.MkdirAll(outlineDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outline := `### 第1章 · 开篇
> 状态：已完成

### 第2章 · 发展

### 第3章 · 高潮
> 状态：偏离
`
	if err := os.WriteFile(p.VolumeOutlinePath(1), []byte(outline), 0o644); err != nil {
		t.Fatal(err)
	}
	chDir := filepath.Join(p.ChaptersDir(), "第001章-开篇")
	if err := os.MkdirAll(chDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chDir, project.ChapterBodyFile), []byte("# 第1章 开篇\n\n正文"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(p.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	idx := struct {
		rebuild func() error
	}{}
	_ = idx
	if err := st.UpsertChapter(store.Chapter{Number: 1, Title: "开篇", WordCount: 2, Status: "draft"}); err != nil {
		t.Fatal(err)
	}

	m, err := BuildMatrix(p, st, 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.Summary.Written != 1 || m.Summary.Unwritten != 1 || m.Summary.Deviated != 1 {
		t.Fatalf("summary = %+v", m.Summary)
	}
}

func TestBuildMatrixEmptyVol1WithVol2Outline(t *testing.T) {
	root := t.TempDir()
	p := &project.Project{Root: root, Meta: project.Meta{Title: "测试"}}
	outlineDir := p.OutlineDir()
	if err := os.MkdirAll(outlineDir, 0o755); err != nil {
		t.Fatal(err)
	}
	vol2 := `### 第9章 · 转卷
> 状态：已完成

### 第10章 · 后续
`
	if err := os.WriteFile(p.VolumeOutlinePath(2), []byte(vol2), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{1, 2, 8} {
		chDir := filepath.Join(p.ChaptersDir(), fmt.Sprintf("第%03d章-测试", n))
		if err := os.MkdirAll(chDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(chDir, project.ChapterBodyFile), []byte("正文"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	st, err := store.Open(p.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for _, n := range []int{1, 2, 8} {
		if err := st.UpsertChapter(store.Chapter{Number: n, Title: "测试", WordCount: 2, Status: "draft"}); err != nil {
			t.Fatal(err)
		}
	}

	m, err := BuildMatrix(p, st, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rows) != 8 {
		t.Fatalf("rows = %d, want 8 (ch 1-8)", len(m.Rows))
	}
	if m.Summary.Written != 3 {
		t.Fatalf("written = %d, want 3", m.Summary.Written)
	}
	if m.Rows[0].Chapter != 1 || !m.Rows[0].HasBody {
		t.Fatalf("first row = %+v", m.Rows[0])
	}
	if m.Rows[7].Chapter != 8 || !m.Rows[7].HasBody {
		t.Fatalf("last row = %+v", m.Rows[7])
	}
}
