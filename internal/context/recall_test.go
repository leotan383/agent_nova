package contextbuilder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestExtractKeywords(t *testing.T) {
	outline := `### 第5章 · 家族大会
- 核心冲突：萧炎 vs 纳兰嫣然
- 爽点：「三年之约」当众打脸`
	got := extractKeywords(outline, "", "萧炎", "药老", "纳兰嫣然")
	if len(got) == 0 {
		t.Fatal("expected keywords")
	}
	if !containsAny(got, "萧炎", "药老", "纳兰嫣然", "家族大会", "三年之约") {
		t.Fatalf("missing expected keywords: %v", got)
	}
}

func TestScoreMemoryRule(t *testing.T) {
	m := store.Memory{
		Subject: "主角性格", Content: "萧炎不轻易示弱，遇强则强",
		SourceChapter: 3,
	}
	score, reasons := scoreMemoryRule(m, []string{"萧炎", "家族"}, []string{"纳兰嫣然"}, "萧炎", 5)
	if score < 5 {
		t.Fatalf("expected high score, got %v reasons=%v", score, reasons)
	}
}

func TestRecallMemoriesRuleOnly(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "nova.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	_, _ = st.UpsertMemory(store.Memory{ID: "m1", Category: "plot", Subject: "三年之约", Content: "萧炎与纳兰嫣然的对赌", Status: "active"})
	_, _ = st.UpsertMemory(store.Memory{ID: "m2", Category: "style", Subject: "无关记忆", Content: "随便写写", Status: "active"})

	hits := RecallMemories(context.Background(), st, nil, RecallInput{
		Chapter: 5, Outline: "萧炎在家族大会面对纳兰嫣然",
		Keywords: []string{"萧炎", "纳兰嫣然", "家族大会"}, Protagonist: "萧炎",
	})
	if len(hits) == 0 {
		t.Fatal("expected recall hits")
	}
	if hits[0].Subject != "三年之约" {
		t.Fatalf("expected plot memory first, got %q", hits[0].Subject)
	}
	if hits[0].Reason == "" {
		t.Fatal("expected recall reason")
	}
}

func TestRRFMerge(t *testing.T) {
	rule := []recallRanked{
		{rank: 1, hit: MemoryRecallHit{ID: "a", Subject: "A", Source: "rule", Reason: "关键词"}},
		{rank: 2, hit: MemoryRecallHit{ID: "b", Subject: "B", Source: "rule", Reason: "关键词"}},
	}
	semantic := []recallRanked{
		{rank: 1, hit: MemoryRecallHit{ID: "b", Subject: "B", Source: "semantic", Reason: "语义"}},
	}
	merged := rrfMerge(rule, semantic)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged, got %d", len(merged))
	}
	if merged[0].ID != "b" {
		t.Fatalf("expected b on top after rrf, got %s", merged[0].ID)
	}
	if merged[0].Source != "rrf" {
		t.Fatalf("expected rrf source, got %s", merged[0].Source)
	}
}

func TestApplyMemoryBudget(t *testing.T) {
	hits := []MemoryRecallHit{
		{Category: "plot", Subject: "A", Content: string(make([]rune, 1500))},
		{Category: "plot", Subject: "B", Content: string(make([]rune, 1500))},
	}
	out := applyMemoryBudget(hits, 2000)
	if len(out) != 1 {
		t.Fatalf("expected budget to trim to 1 hit, got %d", len(out))
	}
}

func containsAny(slice []string, subs ...string) bool {
	for _, s := range slice {
		for _, sub := range subs {
			if s == sub || contains(s, sub) {
				return true
			}
		}
	}
	return false
}

func TestBuildFTSQuery(t *testing.T) {
	q := buildFTSQuery([]string{"萧炎", "家族"}, 5)
	if !contains(q, "第5") || !contains(q, "萧炎") {
		t.Fatalf("unexpected fts query: %q", q)
	}
}

func TestBuilderRecallIntegration(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻", Protagonist: "萧炎"})
	if err != nil {
		t.Fatal(err)
	}
	p := &project.Project{Root: res.Root, Meta: res.Meta}
	st, err := store.Open(p.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_, _ = st.UpsertMemory(store.Memory{ID: "m1", Category: "character", Subject: "萧炎", Content: "主角不轻易杀人", Status: "active"})

	volPath := p.VolumeOutlinePath(1)
	outline := `### 第1章 · 开局
- 萧炎被退婚
`
	if err := os.WriteFile(volPath, []byte(outline), 0o644); err != nil {
		t.Fatal(err)
	}

	cb := Builder{Proj: p, Store: st}
	snap, err := cb.Build(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Memories == "" {
		t.Fatal("expected memories in snapshot")
	}
	if len(snap.MemoryRecalls) == 0 {
		t.Fatal("expected memory recalls metadata")
	}
}
