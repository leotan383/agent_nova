package consistency

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

const (
	ForeshadowOverdueGap  = 20
	ForeshadowCriticalGap = 40
	EntityStaleGap        = 30
)

type Summary struct {
	OpenForeshadows     int `json:"open_foreshadows"`
	OverdueForeshadows  int `json:"overdue_foreshadows"`
	CriticalForeshadows int `json:"critical_foreshadows"`
	MemoryConflicts     int `json:"memory_conflicts"`
	EntityIssues        int `json:"entity_issues"`
	CrossIssues         int `json:"cross_issues"`
	TotalIssues         int `json:"total_issues"`
}

type ForeshadowItem struct {
	ID             string `json:"id"`
	Description    string `json:"description"`
	PlantedChapter int    `json:"planted_chapter"`
	Gap            int    `json:"gap"`
	Severity       string `json:"severity"` // ok | warn | critical
}

type EntityIssue struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	IssueType   string `json:"issue_type"` // duplicate_name | stale
	Detail      string `json:"detail"`
	LastChapter int    `json:"last_chapter"`
	Gap         int    `json:"gap"`
}

type CrossIssue struct {
	Kind     string `json:"kind"` // orphan_character_memory
	Subject  string `json:"subject"`
	Detail   string `json:"detail"`
	MemoryID string `json:"memory_id,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
}

type Report struct {
	CurrentChapter int              `json:"current_chapter"`
	Summary        Summary          `json:"summary"`
	Foreshadows    []ForeshadowItem `json:"foreshadows"`
	MemoryConflicts []store.MemoryConflict `json:"-"`
	EntityIssues   []EntityIssue    `json:"entity_issues"`
	CrossIssues    []CrossIssue     `json:"cross_issues"`
}

// Analyze 生成一致性仪表盘报告。
func Analyze(p *project.Project, st *store.Store) Report {
	r := Report{CurrentChapter: referenceChapter(p, st)}
	if st == nil {
		return r
	}

	open, _ := st.ListForeshadows("open")
	r.Summary.OpenForeshadows = len(open)
	for _, f := range open {
		gap := r.CurrentChapter - f.PlantedChapter
		if gap < 0 {
			gap = 0
		}
		sev := foreshadowSeverity(gap)
		item := ForeshadowItem{
			ID: f.ID, Description: f.Description, PlantedChapter: f.PlantedChapter,
			Gap: gap, Severity: sev,
		}
		r.Foreshadows = append(r.Foreshadows, item)
		switch sev {
		case "critical":
			r.Summary.CriticalForeshadows++
		case "warn":
			r.Summary.OverdueForeshadows++
		}
	}

	conflicts, _ := st.FindMemoryConflicts()
	r.MemoryConflicts = conflicts
	r.Summary.MemoryConflicts = len(conflicts)

	entities, _ := st.ListEntities("", 500)
	r.EntityIssues = analyzeEntities(entities, r.CurrentChapter)
	r.Summary.EntityIssues = len(r.EntityIssues)

	memories, _ := st.ListActiveMemories(500)
	r.CrossIssues = analyzeCrossIssues(memories, entities)
	r.Summary.CrossIssues = len(r.CrossIssues)

	r.Summary.TotalIssues = r.Summary.MemoryConflicts + r.Summary.EntityIssues + r.Summary.CrossIssues
	for _, f := range r.Foreshadows {
		if f.Severity != "ok" {
			r.Summary.TotalIssues++
		}
	}
	return r
}

func referenceChapter(p *project.Project, st *store.Store) int {
	ref := p.Meta.CurrentChapter
	if st == nil {
		return ref
	}
	chs, _ := st.ListChapters()
	for _, c := range chs {
		if c.Number > ref {
			ref = c.Number
		}
	}
	return ref
}

func foreshadowSeverity(gap int) string {
	switch {
	case gap >= ForeshadowCriticalGap:
		return "critical"
	case gap >= ForeshadowOverdueGap:
		return "warn"
	default:
		return "ok"
	}
}

func analyzeEntities(entities []store.Entity, currentChapter int) []EntityIssue {
	byName := map[string][]store.Entity{}
	for _, e := range entities {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			continue
		}
		byName[name] = append(byName[name], e)
	}

	var out []EntityIssue
	seenDup := map[string]struct{}{}
	for name, list := range byName {
		if len(list) > 1 {
			ids := make([]string, len(list))
			for i, e := range list {
				ids[i] = e.ID
			}
			key := name
			if _, ok := seenDup[key]; ok {
				continue
			}
			seenDup[key] = struct{}{}
			out = append(out, EntityIssue{
				Name: name, Type: list[0].Type, IssueType: "duplicate_name",
				Detail: fmt.Sprintf("存在 %d 条同名实体（%s）", len(list), strings.Join(ids, ", ")),
				LastChapter: maxLastChapter(list),
			})
		}
	}

	for _, e := range entities {
		if currentChapter <= 0 || e.LastChapter <= 0 {
			continue
		}
		gap := currentChapter - e.LastChapter
		if gap < EntityStaleGap {
			continue
		}
		out = append(out, EntityIssue{
			ID: e.ID, Name: e.Name, Type: e.Type, IssueType: "stale",
			Detail: fmt.Sprintf("实体状态已 %d 章未更新（最近第 %d 章）", gap, e.LastChapter),
			LastChapter: e.LastChapter, Gap: gap,
		})
	}
	return out
}

func maxLastChapter(list []store.Entity) int {
	max := 0
	for _, e := range list {
		if e.LastChapter > max {
			max = e.LastChapter
		}
	}
	return max
}

func analyzeCrossIssues(memories []store.Memory, entities []store.Entity) []CrossIssue {
	if len(entities) == 0 {
		return nil
	}
	names := map[string]struct{}{}
	for _, e := range entities {
		if e.Name != "" {
			names[e.Name] = struct{}{}
		}
	}

	var out []CrossIssue
	for _, m := range memories {
		if m.Category != "character" {
			continue
		}
		subject := strings.TrimSpace(m.Subject)
		if subject == "" || m.SourceChapter <= 0 {
			continue
		}
		if _, ok := names[subject]; ok {
			continue
		}
		// 内容里提到了已知实体则不算 orphan
		linked := false
		for name := range names {
			if strings.Contains(m.Content, name) {
				linked = true
				break
			}
		}
		if linked {
			continue
		}
		out = append(out, CrossIssue{
			Kind: "orphan_character_memory", Subject: subject, MemoryID: m.ID,
			Detail: fmt.Sprintf("角色类记忆「%s」在 entities 中无对应实体", subject),
		})
	}
	return out
}

// ParseEntityState 解析 state_json 为 map（供展示）。
func ParseEntityState(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return map[string]string{"raw": raw}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		switch t := v.(type) {
		case string:
			out[k] = t
		default:
			b, _ := json.Marshal(t)
			out[k] = string(b)
		}
	}
	return out
}
