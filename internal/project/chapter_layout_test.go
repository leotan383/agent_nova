package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
)

func TestChapterLayoutPaths(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻"})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := project.Load(res.Root)

	bodyPath := p.ChapterPath(1, "开端")
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bodyPath, []byte("# 第一章\n\nhello"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, title, err := p.FindChapterFile(1)
	if err != nil {
		t.Fatal(err)
	}
	if got != bodyPath {
		t.Fatalf("body path: got %s want %s", got, bodyPath)
	}
	if title != "开端" {
		t.Fatalf("title=%q", title)
	}

	reviewPath := p.ReviewPath(1)
	if err := os.WriteFile(reviewPath, []byte("review"), 0o644); err != nil {
		t.Fatal(err)
	}
	summaryPath := p.SummaryPath(1)
	if err := os.WriteFile(summaryPath, []byte("summary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(reviewPath)) != "第001章-开端" {
		t.Fatalf("review dir: %s", reviewPath)
	}
}

func TestMigrateLegacyChapterLayout(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻"})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := project.Load(res.Root)

	legacyBody := filepath.Join(p.ChaptersDir(), "第001章-旧章.md")
	if err := os.WriteFile(legacyBody, []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(p.ReviewsDir(), 0o755)
	if err := os.WriteFile(filepath.Join(p.ReviewsDir(), "第001章.review.md"), []byte("rev"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(p.SummariesDir(), 0o755)
	if err := os.WriteFile(filepath.Join(p.SummariesDir(), "第001章.summary.md"), []byte("sum"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := p.MigrateChapterLayout(); err != nil {
		t.Fatal(err)
	}
	path, _, err := p.FindChapterFile(1)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != project.ChapterBodyFile {
		t.Fatalf("path=%s", path)
	}
	if _, err := os.Stat(p.ReviewPath(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p.SummariesDir()); !os.IsNotExist(err) {
		t.Fatal("expected legacy 摘要 dir removed when empty")
	}
}
