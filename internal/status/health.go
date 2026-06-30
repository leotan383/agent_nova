package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// TodoItem 可操作的待办项，供桌面端概览展示。
type TodoItem struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Detail      string `json:"detail"`
	Severity    string `json:"severity"`
	Action      string `json:"action"`
	ActionParam string `json:"action_param,omitempty"`
}

// HealthReport 项目健康与待办汇总。
type HealthReport struct {
	OK                bool       `json:"ok"`
	SuggestedVolume   int        `json:"suggested_volume"`
	NextChapter       int        `json:"next_chapter"`
	HasVolumeOutline  bool       `json:"has_volume_outline"`
	VolumeOutlinePath string     `json:"volume_outline_path,omitempty"`
	Todos             []TodoItem `json:"todos"`
}

// BuildHealth 基于项目状态与索引生成结构化待办。
func BuildHealth(p *project.Project, st *store.Store) HealthReport {
	rep := Build(p, st, "all")
	vol := max(1, rep.CurrentVolume)
	nextCh := rep.CurrentChapter + 1
	if nextCh <= 0 {
		nextCh = 1
	}

	h := HealthReport{
		SuggestedVolume:  vol,
		NextChapter:      nextCh,
		VolumeOutlinePath: p.VolumeOutlinePath(vol),
	}

	volPath := p.VolumeOutlinePath(vol)
	if _, err := os.Stat(volPath); err != nil {
		h.Todos = append(h.Todos, TodoItem{
			ID:          "missing_volume_outline",
			Label:       fmt.Sprintf("生成第 %d 卷卷纲", vol),
			Detail:      "写章前需要卷纲作为章节任务来源",
			Severity:    "urgent",
			Action:      "plan_volume",
			ActionParam: fmt.Sprint(vol),
		})
	} else {
		h.HasVolumeOutline = true
	}

	if rep.CurrentChapter > 0 {
		if _, err := os.Stat(p.ReviewPath(rep.CurrentChapter)); err != nil {
			h.Todos = append(h.Todos, TodoItem{
				ID:          fmt.Sprintf("missing_review_%d", rep.CurrentChapter),
				Label:       fmt.Sprintf("查看第 %d 章审查", rep.CurrentChapter),
				Detail:      "上一章尚未完成审查流程",
				Severity:    "warn",
				Action:      "open_chapter_review",
				ActionParam: fmt.Sprint(rep.CurrentChapter),
			})
		}
	}

	if st != nil {
		stale := st.CheckIndexStale(p.ChaptersDir())
		if stale.Stale {
			detail := strings.Join(stale.Issues, "；")
			if detail == "" {
				detail = "章节索引与正文目录不同步"
			}
			h.Todos = append(h.Todos, TodoItem{
				ID:       "rebuild_index",
				Label:    "重建章节索引",
				Detail:   detail,
				Severity: "warn",
				Action:   "rebuild_index",
			})
		}
	}

	switch rep.Phase {
	case project.PhaseInitDone:
		if h.HasVolumeOutline {
			h.Todos = append(h.Todos, TodoItem{
				ID:       "phase_planning",
				Label:    "进入规划阶段",
				Detail:   "卷纲已就绪，可开始写第一章",
				Severity: "info",
				Action:   "open_write",
			})
		}
	case project.PhasePlanning, project.PhaseWriting:
		if h.HasVolumeOutline {
			h.Todos = append(h.Todos, TodoItem{
				ID:          "write_next",
				Label:       fmt.Sprintf("写第 %d 章", nextCh),
				Detail:      "继续创作下一章",
				Severity:    "info",
				Action:      "open_write",
				ActionParam: fmt.Sprint(nextCh),
			})
		}
	}

	h.OK = true
	for _, t := range h.Todos {
		if t.Severity == "urgent" {
			h.OK = false
			break
		}
	}
	return h
}
