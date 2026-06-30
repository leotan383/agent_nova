package pipeline_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestRunGatePrewrite(t *testing.T) {
	dir := t.TempDir()
	p := &project.Project{
		Root: dir,
		Meta: project.Meta{Phase: project.PhaseInitDone},
	}
	_ = os.MkdirAll(filepath.Join(dir, "大纲"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "正文"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "大纲", "第01卷.md"), []byte("# vol1"), 0o644)
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	res := pipeline.RunGate(p, st, 1, pipeline.GatePrewrite)
	if res.OK {
		t.Fatal("expected gate to fail without writing phase")
	}
	p.Meta.Phase = project.PhaseWriting
	res = pipeline.RunGate(p, st, 1, pipeline.GatePrewrite)
	if !res.OK {
		t.Fatalf("expected ok: %v", res.Issues)
	}
}
