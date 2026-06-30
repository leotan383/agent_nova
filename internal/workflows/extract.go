package workflows

import (
	"context"
	"encoding/json"
	"fmt"
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
	raw, err := ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.ExtractSystem(),
		UserPrompt: fmt.Sprintf("章号：%d\n\n摘要：\n%s\n\n正文（节选前8000字）：\n%s",
			chapter, summary, truncateRunes(body, 8000)),
	})
	if err != nil {
		return err
	}
	jsonRaw, err := agent.ExtractJSONBlock(raw)
	if err != nil {
		return fmt.Errorf("extract facts: %w", err)
	}
	var facts storyFacts
	if err := json.Unmarshal([]byte(jsonRaw), &facts); err != nil {
		return fmt.Errorf("parse facts: %w", err)
	}
	for _, e := range facts.Entities {
		if e.Name == "" {
			continue
		}
		typ := e.Type
		if typ == "" {
			typ = "character"
		}
		id := entityID(typ, e.Name)
		_ = st.UpsertEntity(store.Entity{
			ID: id, Type: typ, Name: e.Name,
			StateJSON: store.EntityStateJSON(e.State), LastChapter: chapter,
		})
	}
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

func entityID(typ, name string) string {
	return fmt.Sprintf("%s:%s", typ, strings.TrimSpace(name))
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
