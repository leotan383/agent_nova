package workflows

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/store"
)

type storyFacts struct {
	Entities    []extractEntity    `json:"entities"`
	Foreshadows []extractForeshadow `json:"foreshadows"`
	CoolPoints  []extractCoolPoint  `json:"cool_points"`
	Memories    []extractMemory     `json:"memories"`
}

type extractEntity struct {
	Type  string         `json:"type"`
	Name  string         `json:"name"`
	State map[string]any `json:"state"`
}

type extractForeshadow struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Status      string `json:"status"`
}

type extractCoolPoint struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Delivered   bool   `json:"delivered"`
}

type extractMemory struct {
	Category string `json:"category"`
	Subject  string `json:"subject"`
	Content  string `json:"content"`
}

func ExtractAndPersistFacts(ctx context.Context, ag *agent.Agent, st *store.Store, chapter int, body, summary string) error {
	facts, err := extractStoryFacts(ctx, ag, chapter, body, summary)
	if err != nil {
		return err
	}
	persistExtractedEntities(st, chapter, facts.Entities)
	for _, f := range facts.Foreshadows {
		if f.Description == "" {
			continue
		}
		id := f.ID
		if id == "" {
			id = foreshadowID(f.Description)
		}
		status := f.Status
		if status == "" {
			status = "open"
		}
		resolved := 0
		if f.Action == "resolve" || status == "resolved" {
			status = "resolved"
			resolved = chapter
		}
		_ = st.UpsertForeshadow(store.Foreshadow{
			ID: id, Description: f.Description,
			PlantedChapter: chapter, ResolvedChapter: resolved, Status: status,
		})
	}
	for _, cp := range facts.CoolPoints {
		if cp.Description == "" {
			continue
		}
		typ := cp.Type
		if typ == "" {
			typ = "micro"
		}
		_ = st.UpsertCoolPoint(store.CoolPoint{
			ID: project.NewMemoryID(), Chapter: chapter, Type: typ,
			Description: cp.Description, Delivered: cp.Delivered,
		})
	}
	for _, m := range facts.Memories {
		if m.Content == "" {
			continue
		}
		_, _ = st.UpsertMemory(store.Memory{
			Category: m.Category, Subject: m.Subject, Content: m.Content,
			SourceChapter: chapter, Status: "active", CreatedAt: project.Timestamp(),
		})
	}
	return nil
}

// ExtractAndRecordEntityHistoryOnly 仅更新实体当前状态并写入历史快照，不写记忆/伏笔/爽点。
func ExtractAndRecordEntityHistoryOnly(ctx context.Context, ag *agent.Agent, st *store.Store, chapter int, body, summary string) error {
	facts, err := extractStoryFacts(ctx, ag, chapter, body, summary)
	if err != nil {
		return err
	}
	persistExtractedEntities(st, chapter, facts.Entities)
	return nil
}

// BackfillEntityStateHistoryResult 历史回溯结果。
type BackfillEntityStateHistoryResult struct {
	Processed int
	Skipped   []string
}

// BackfillEntityStateHistory 对已写章节重新提取实体状态，补齐历史时间线（幂等）。
func BackfillEntityStateHistory(ctx context.Context, ag *agent.Agent, st *store.Store, p *project.Project, throughChapter int, onProgress func(chapter int, message string)) (BackfillEntityStateHistoryResult, error) {
	if throughChapter <= 0 {
		throughChapter = p.Meta.CurrentChapter
	}
	var result BackfillEntityStateHistoryResult
	for ch := 1; ch <= throughChapter; ch++ {
		if onProgress != nil {
			onProgress(ch, fmt.Sprintf("正在处理第 %d 章…", ch))
		}
		_, body := loadChapterFile(p, ch)
		if strings.TrimSpace(body) == "" {
			continue
		}
		summaryBytes, err := os.ReadFile(p.SummaryPath(ch))
		if err != nil || strings.TrimSpace(string(summaryBytes)) == "" {
			continue
		}
		if err := ExtractAndRecordEntityHistoryOnly(ctx, ag, st, ch, body, string(summaryBytes)); err != nil {
			msg := fmt.Sprintf("第 %d 章: %s", ch, err)
			result.Skipped = append(result.Skipped, msg)
			if onProgress != nil {
				onProgress(ch, fmt.Sprintf("第 %d 章跳过（%s），继续后续章节…", ch, err))
			}
			continue
		}
		result.Processed++
	}
	return result, nil
}

func extractStoryFacts(ctx context.Context, ag *agent.Agent, chapter int, body, summary string) (*storyFacts, error) {
	userPrompt := fmt.Sprintf("章号：%d\n\n摘要：\n%s\n\n正文（节选前8000字）：\n%s",
		chapter, summary, truncateRunes(body, 8000))
	raw, err := ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.ExtractSystem(),
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return nil, err
	}
	facts, parseErr := parseStoryFactsJSON(raw)
	if parseErr == nil {
		return facts, nil
	}
	retryPrompt := userPrompt + "\n\n上次输出不是合法 JSON。请只输出单个 JSON 对象，不要 markdown 说明，不要尾随逗号，不要省略字段值。"
	raw, err = ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.ExtractSystem(),
		UserPrompt:   retryPrompt,
	})
	if err != nil {
		return nil, parseErr
	}
	facts, retryErr := parseStoryFactsJSON(raw)
	if retryErr != nil {
		return nil, fmt.Errorf("parse facts: %w", retryErr)
	}
	return facts, nil
}

func parseStoryFactsJSON(raw string) (*storyFacts, error) {
	jsonRaw, err := agent.ExtractJSONBlock(raw)
	if err != nil {
		return nil, fmt.Errorf("extract facts: %w", err)
	}
	var facts storyFacts
	if err := agent.UnmarshalJSONObject(jsonRaw, &facts); err != nil {
		return nil, fmt.Errorf("parse facts: %w", err)
	}
	return &facts, nil
}

func persistExtractedEntities(st *store.Store, chapter int, entities []extractEntity) {
	for _, e := range entities {
		if e.Name == "" {
			continue
		}
		typ := e.Type
		if typ == "" {
			typ = "character"
		}
		canonical := store.CanonicalEntityName(e.Name)
		state := store.AppendEntityAlias(e.State, e.Name)
		id := store.EntityID(typ, canonical)
		stateJSON := store.EntityStateJSON(state)
		_ = st.UpsertEntity(store.Entity{
			ID: id, Type: typ, Name: canonical,
			StateJSON: stateJSON, LastChapter: chapter,
		})
		_ = st.RecordEntityStateHistory(id, chapter, stateJSON)
	}
}

func foreshadowID(desc string) string {
	d := strings.TrimSpace(desc)
	if len([]rune(d)) > 32 {
		d = string([]rune(d)[:32])
	}
	return "fs:" + strings.ReplaceAll(d, " ", "_")
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
