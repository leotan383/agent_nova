package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
)

func setupTestProject(t *testing.T) *project.Project {
	t.Helper()
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{
		Dir: dir, Title: "测试", Genre: "玄幻",
	})
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.Load(res.Root)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLineDiff(t *testing.T) {
	lines := lineDiff("a\nb\nc", "a\nx\nc")
	var adds, dels int
	for _, l := range lines {
		if l.Type == "add" {
			adds++
		}
		if l.Type == "del" {
			dels++
		}
	}
	if adds != 1 || dels != 1 {
		t.Fatalf("adds=%d dels=%d lines=%v", adds, dels, lines)
	}
}

func TestBeforeSaveSkipsSame(t *testing.T) {
	p := setupTestProject(t)
	body := "# 第一章\n\nhello"
	path := p.ChapterPath(1, "第一章")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := BeforeSave(p, 1, body, SourceCoachApply, "test"); err != nil {
		t.Fatal(err)
	}
	versions, _ := List(p, 1)
	if len(versions) != 0 {
		t.Fatalf("expected no snapshot for same content, got %d", len(versions))
	}
}

func TestBeforeSaveSnapshotsOnChange(t *testing.T) {
	p := setupTestProject(t)
	path := p.ChapterPath(1, "第一章")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := BeforeSave(p, 1, "new content", SourceCoachApply, "改稿"); err != nil {
		t.Fatal(err)
	}
	versions, err := List(p, 1)
	if err != nil || len(versions) != 1 {
		t.Fatalf("versions=%v err=%v", versions, err)
	}
	if !strings.Contains(versions[0].File, ".md") {
		t.Fatalf("file=%s", versions[0].File)
	}
	content, err := GetContent(p, 1, versions[0].ID)
	if err != nil || content != "old content" {
		t.Fatalf("content=%q err=%v", content, err)
	}
}

func TestDiffWithNew(t *testing.T) {
	p := setupTestProject(t)
	path := p.ChapterPath(1, "第一章")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("line1\nline2"), 0o644); err != nil {
		t.Fatal(err)
	}
	diff, err := DiffWithNew(p, 1, "line1\nline3")
	if err != nil {
		t.Fatal(err)
	}
	if diff.FromID != "current" || diff.ToID != "pending" {
		t.Fatalf("ids: from=%s to=%s", diff.FromID, diff.ToID)
	}
}
