package status

import (
	"fmt"
	"os"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

type Report struct {
	Phase           string   `json:"phase"`
	Title           string   `json:"title"`
	CurrentVolume   int      `json:"current_volume"`
	CurrentChapter  int      `json:"current_chapter"`
	ChapterCount    int      `json:"chapter_count"`
	OpenForeshadows int      `json:"open_foreshadows"`
	MemoryCount     int      `json:"memory_count"`
	Urgent          []string `json:"urgent,omitempty"`
	NextSteps       []string `json:"next_steps,omitempty"`
	Progress
}

func Build(p *project.Project, st *store.Store, focus string) Report {
	r := Report{
		Phase:          p.Meta.Phase,
		Title:          p.Meta.Title,
		CurrentVolume:  p.Meta.CurrentVolume,
		CurrentChapter: p.Meta.CurrentChapter,
	}
	chs, _ := st.ListChapters()
	r.ChapterCount = len(chs)
	r.Progress = ComputeProgress(p.Meta, chs)
	open, _ := st.ListForeshadows("open")
	r.OpenForeshadows = len(open)
	_, total, _ := st.MemoryStats()
	r.MemoryCount = total
	switch p.Meta.Phase {
	case project.PhaseInitDone:
		r.NextSteps = []string{"nova plan 1"}
	case project.PhasePlanning:
		r.NextSteps = []string{fmt.Sprintf("nova write %d", p.Meta.CurrentChapter+1)}
	case project.PhaseWriting:
		r.NextSteps = []string{fmt.Sprintf("nova write %d", p.Meta.CurrentChapter+1), "nova review " + fmt.Sprint(max(1, p.Meta.CurrentChapter))}
	}
	if focus == "urgency" || focus == "all" {
		if _, err := os.Stat(p.VolumeOutlinePath(max(1, p.Meta.CurrentVolume))); err != nil {
			r.Urgent = append(r.Urgent, "缺少卷纲，运行 nova plan 1")
		}
		if p.Meta.CurrentChapter > 0 {
			if _, err := os.Stat(p.ReviewPath(p.Meta.CurrentChapter)); err != nil {
				r.Urgent = append(r.Urgent, fmt.Sprintf("第 %d 章未审查", p.Meta.CurrentChapter))
			}
		}
	}
	return r
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
