package search

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/wiki"
)

// Hit 统一检索结果。
type Hit struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Chapter int    `json:"chapter"`
	WikiID  string `json:"wiki_id,omitempty"`
}

// Search 混合检索章节 FTS、设定 FTS、记忆、伏笔、实体。
func Search(p *project.Project, st *store.Store, query string, limit int) ([]Hit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	seen := map[string]struct{}{}
	var out []Hit

	add := func(h Hit) {
		key := h.Kind + ":" + h.ID
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, h)
	}

	if st != nil {
		hits, _ := st.SearchFTS(query, limit)
		for _, h := range hits {
			kind := h["kind"]
			id := h["id"]
			title := h["title"]
			snippet := h["snippet"]
			chapter := 0
			wikiID := ""
			if kind == "chapter" {
				chapter, _ = strconv.Atoi(id)
				if title == "" {
					title = fmt.Sprintf("第%d章", chapter)
				}
			} else if kind == "setting" {
				base := filepath.Base(id)
				title = strings.TrimSuffix(base, ".md")
				if base != "" {
					wikiID = wiki.KindSetting + ":" + base
				}
			}
			add(Hit{Kind: kind, ID: id, Title: title, Snippet: snippet, Chapter: chapter, WikiID: wikiID})
		}

		memories, _ := st.QueryMemories("", query, limit)
		for _, m := range memories {
			if m.Status != "" && m.Status != "active" {
				continue
			}
			title := m.Subject
			if title == "" {
				title = m.Category
			}
			snippet := m.Content
			if len([]rune(snippet)) > 80 {
				snippet = string([]rune(snippet)[:80]) + "…"
			}
			add(Hit{
				Kind: "memory", ID: m.ID, Title: title, Snippet: snippet,
				Chapter: m.SourceChapter, WikiID: wiki.KindMemory + ":" + m.ID,
			})
		}

		foreshadows, _ := st.ListForeshadows("")
		for _, f := range foreshadows {
			if !strings.Contains(f.Description, query) && !strings.Contains(f.ID, query) {
				continue
			}
			add(Hit{
				Kind: "foreshadow", ID: f.ID, Title: "伏笔",
				Snippet: f.Description, Chapter: f.PlantedChapter,
			})
		}

		entities, _ := st.SearchEntities(query, limit)
		for _, e := range entities {
			snippet := e.StateJSON
			if len(snippet) > 80 {
				snippet = snippet[:80] + "…"
			}
			add(Hit{
				Kind: "entity", ID: e.ID, Title: e.Name, Snippet: snippet,
				Chapter: e.LastChapter, WikiID: wiki.KindEntity + ":" + e.ID,
			})
		}
	}

	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
