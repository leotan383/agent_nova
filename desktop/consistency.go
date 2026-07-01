package main

import (
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/consistency"
)

// ConsistencySummaryDTO 一致性概览指标。
type ConsistencySummaryDTO struct {
	OpenForeshadows     int `json:"open_foreshadows"`
	OverdueForeshadows  int `json:"overdue_foreshadows"`
	CriticalForeshadows int `json:"critical_foreshadows"`
	MemoryConflicts     int `json:"memory_conflicts"`
	EntityIssues        int `json:"entity_issues"`
	CrossIssues         int `json:"cross_issues"`
	TotalIssues         int `json:"total_issues"`
}

// ForeshadowHealthDTO 伏笔健康条目。
type ForeshadowHealthDTO struct {
	ID             string `json:"id"`
	Description    string `json:"description"`
	PlantedChapter int    `json:"planted_chapter"`
	Gap            int    `json:"gap"`
	Severity       string `json:"severity"`
}

// EntityIssueDTO 实体一致性问题。
type EntityIssueDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	IssueType   string `json:"issue_type"`
	Detail      string `json:"detail"`
	LastChapter int    `json:"last_chapter"`
	Gap         int    `json:"gap"`
}

// CrossIssueDTO 跨模块轻量矛盾。
type CrossIssueDTO struct {
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	Detail   string `json:"detail"`
	MemoryID string `json:"memory_id,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
}

// ConsistencyReportDTO 一致性仪表盘完整报告。
type ConsistencyReportDTO struct {
	CurrentChapter  int                   `json:"current_chapter"`
	Summary         ConsistencySummaryDTO `json:"summary"`
	Foreshadows     []ForeshadowHealthDTO `json:"foreshadows"`
	MemoryConflicts []MemoryConflictDTO   `json:"memory_conflicts"`
	EntityIssues    []EntityIssueDTO      `json:"entity_issues"`
	CrossIssues     []CrossIssueDTO       `json:"cross_issues"`
}

// GetConsistencyReport 返回当前小说的一致性分析。
func (a *App) GetConsistencyReport() (ConsistencyReportDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return ConsistencyReportDTO{}, err
	}
	var out ConsistencyReportDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		rep := consistency.Analyze(actx.Project, actx.Store)
		out.CurrentChapter = rep.CurrentChapter
		out.Summary = ConsistencySummaryDTO(rep.Summary)
		out.Foreshadows = make([]ForeshadowHealthDTO, len(rep.Foreshadows))
		for i, f := range rep.Foreshadows {
			out.Foreshadows[i] = ForeshadowHealthDTO(f)
		}
		out.MemoryConflicts = make([]MemoryConflictDTO, len(rep.MemoryConflicts))
		for i, c := range rep.MemoryConflicts {
			out.MemoryConflicts[i] = MemoryConflictDTO{Subject: c.Subject, Count: c.Count}
			out.MemoryConflicts[i].Memories = make([]MemoryDTO, len(c.Memories))
			for j, m := range c.Memories {
				out.MemoryConflicts[i].Memories[j] = MemoryDTO{
					ID: m.ID, Category: m.Category, Subject: m.Subject, Content: m.Content,
					SourceChapter: m.SourceChapter, Status: m.Status,
				}
			}
		}
		out.EntityIssues = make([]EntityIssueDTO, len(rep.EntityIssues))
		for i, e := range rep.EntityIssues {
			out.EntityIssues[i] = EntityIssueDTO(e)
		}
		out.CrossIssues = make([]CrossIssueDTO, len(rep.CrossIssues))
		for i, c := range rep.CrossIssues {
			out.CrossIssues[i] = CrossIssueDTO(c)
		}
		return nil
	})
	return out, err
}
