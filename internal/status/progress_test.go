package status

import (
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestComputeProgress(t *testing.T) {
	meta := project.Meta{
		TargetWords:  300000,
		ChapterWords: 4000,
		Style:        "热血",
	}
	chs := []store.Chapter{
		{Number: 1, WordCount: 3800},
		{Number: 2, WordCount: 4200},
	}
	p := ComputeProgress(meta, chs)
	if p.WrittenWords != 8000 {
		t.Fatalf("written=%d", p.WrittenWords)
	}
	if p.RemainingWords != 292000 {
		t.Fatalf("remaining=%d", p.RemainingWords)
	}
	if p.EstimatedTotalChapters != 75 {
		t.Fatalf("est total=%d", p.EstimatedTotalChapters)
	}
	if p.RemainingChapters != 73 {
		t.Fatalf("remaining chapters=%d", p.RemainingChapters)
	}
	if p.ProgressPercent < 2.6 || p.ProgressPercent > 2.7 {
		t.Fatalf("percent=%f", p.ProgressPercent)
	}
}

func TestComputeProgressDefaults(t *testing.T) {
	p := ComputeProgress(project.Meta{}, nil)
	if p.TargetWords != project.DefaultTargetWords {
		t.Fatalf("target=%d", p.TargetWords)
	}
	if p.ChapterWordsGoal != project.DefaultChapterWords {
		t.Fatalf("chapter goal=%d", p.ChapterWordsGoal)
	}
}
