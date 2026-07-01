package workflows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
)

func TestCollectWrittenSummaries(t *testing.T) {
	dir := t.TempDir()
	p := &project.Project{Root: dir}
	for i := 1; i <= 3; i++ {
		path := p.SummaryPath(i)
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, []byte("summary"), 0o644)
	}
	out := collectWrittenSummaries(p, 3, 10000)
	if out == "" {
		t.Fatal("expected summaries")
	}
	for _, want := range []string{"第1章摘要", "第2章摘要", "第3章摘要"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestIfEmpty(t *testing.T) {
	if ifEmpty("(无)", "") != "(无)" {
		t.Fatal("empty should use fallback")
	}
	if ifEmpty("(无)", "x") != "x" {
		t.Fatal("non-empty should pass through")
	}
}
