package consistency

import (
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestForeshadowSeverity(t *testing.T) {
	if foreshadowSeverity(19) != "ok" {
		t.Fatal("19 should be ok")
	}
	if foreshadowSeverity(20) != "warn" {
		t.Fatal("20 should be warn")
	}
	if foreshadowSeverity(40) != "critical" {
		t.Fatal("40 should be critical")
	}
}

func TestAnalyzeForeshadows(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻"})
	if err != nil {
		t.Fatal(err)
	}
	p := &project.Project{Root: res.Root, Meta: res.Meta}
	p.Meta.CurrentChapter = 30
	st, err := store.Open(p.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	_ = st.UpsertForeshadow(store.Foreshadow{
		ID: "fs1", Description: "神秘来信", PlantedChapter: 5, Status: "open",
	})
	_ = st.UpsertForeshadow(store.Foreshadow{
		ID: "fs2", Description: "近期伏笔", PlantedChapter: 25, Status: "open",
	})

	r := Analyze(p, st)
	if r.Summary.OpenForeshadows != 2 {
		t.Fatalf("open=%d", r.Summary.OpenForeshadows)
	}
	if r.Summary.OverdueForeshadows+r.Summary.CriticalForeshadows == 0 {
		t.Fatal("expected overdue foreshadow")
	}
}

func TestAnalyzeMemoryConflicts(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_, _ = st.UpsertMemory(store.Memory{ID: "a", Category: "plot", Subject: "主角", Content: "A", Status: "active"})
	_, _ = st.UpsertMemory(store.Memory{ID: "b", Category: "style", Subject: "主角", Content: "B", Status: "active"})

	p := &project.Project{Meta: project.Meta{CurrentChapter: 10}}
	r := Analyze(p, st)
	if r.Summary.MemoryConflicts != 1 {
		t.Fatalf("conflicts=%d", r.Summary.MemoryConflicts)
	}
}

func TestAnalyzeEntityDuplicate(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_ = st.UpsertEntity(store.Entity{ID: "e1", Type: "character", Name: "萧炎", LastChapter: 5})
	_ = st.UpsertEntity(store.Entity{ID: "e2", Type: "character", Name: "萧炎", LastChapter: 8})

	p := &project.Project{Meta: project.Meta{CurrentChapter: 10}}
	r := Analyze(p, st)
	if r.Summary.EntityIssues == 0 {
		t.Fatal("expected duplicate entity issue")
	}
}

func TestAnalyzeCrossOrphanMemory(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_ = st.UpsertEntity(store.Entity{ID: "e1", Type: "character", Name: "萧炎", LastChapter: 5})
	_, _ = st.UpsertMemory(store.Memory{
		ID: "m1", Category: "character", Subject: "云岚宗长老", Content: "未登记的角色",
		SourceChapter: 3, Status: "active",
	})

	p := &project.Project{Meta: project.Meta{CurrentChapter: 10}}
	r := Analyze(p, st)
	if r.Summary.CrossIssues != 1 {
		t.Fatalf("cross=%d", r.Summary.CrossIssues)
	}
}
