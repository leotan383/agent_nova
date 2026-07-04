package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
)

func TestWritePDF(t *testing.T) {
	fontPath, err := resolvePDFFont()
	if err != nil {
		t.Skip("no CJK font:", err)
	}
	t.Setenv("NOVA_PDF_FONT", fontPath)

	dir := t.TempDir()
	p := &project.Project{
		Root: dir,
		Meta: project.Meta{Title: "测试小说", Genre: "玄幻", Style: "热血"},
	}
	chDir := filepath.Join(p.ChaptersDir(), "第001章-开篇")
	if err := os.MkdirAll(chDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# 第一章 开篇\n\n林枫踏入宗门，心中隐忍不发。\n\n## 小节\n\n测试段落。"
	if err := os.WriteFile(filepath.Join(chDir, project.ChapterBodyFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "out.pdf")
	if err := WritePDF(p, out, Options{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() < 100 {
		t.Fatalf("pdf missing or too small: %v", err)
	}
}
