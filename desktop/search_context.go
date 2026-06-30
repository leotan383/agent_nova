package main

import (
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/search"
	contextbuilder "github.com/tanlian/agent_nova/internal/context"
)

// SearchHitDTO 检索结果。
type SearchHitDTO struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Chapter int    `json:"chapter"`
	WikiID  string `json:"wiki_id,omitempty"`
}

// WriteContextDTO 写章上下文快照。
type WriteContextDTO struct {
	Chapter          int    `json:"chapter"`
	Volume           int    `json:"volume"`
	Outline          string `json:"outline"`
	RecentSummary    string `json:"recent_summary"`
	Settings         string `json:"settings"`
	Memories         string `json:"memories"`
	FTSHits          string `json:"fts_hits"`
	OpenForeshadows  string `json:"open_foreshadows"`
}

func toSearchHits(hits []search.Hit) []SearchHitDTO {
	out := make([]SearchHitDTO, len(hits))
	for i, h := range hits {
		out[i] = SearchHitDTO{
			Kind: h.Kind, ID: h.ID, Title: h.Title, Snippet: h.Snippet,
			Chapter: h.Chapter, WikiID: h.WikiID,
		}
	}
	return out
}

// SearchProject 全局混合检索。
func (a *App) SearchProject(query string, limit int) ([]SearchHitDTO, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	var out []SearchHitDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		hits, err := search.Search(actx.Project, actx.Store, query, limit)
		if err != nil {
			return err
		}
		out = toSearchHits(hits)
		return nil
	})
	return out, err
}

// GetWriteContext 组装指定章节的写作上下文。
func (a *App) GetWriteContext(chapter, volume int) (WriteContextDTO, error) {
	if chapter <= 0 {
		return WriteContextDTO{}, fmt.Errorf("无效章号")
	}
	if volume <= 0 {
		volume = 1
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return WriteContextDTO{}, err
	}
	var result WriteContextDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		cb := contextbuilder.Builder{Proj: actx.Project, Store: actx.Store}
		snap, err := cb.Build(chapter, volume)
		if err != nil {
			return err
		}
		result = WriteContextDTO{
			Chapter: snap.Chapter, Volume: snap.Volume,
			Outline:    firstNonEmpty(snap.ChapterOutline, snap.VolumeOutline),
			RecentSummary: snap.RecentSummary, Settings: snap.Settings,
			Memories: snap.Memories, FTSHits: snap.FTSHits,
			OpenForeshadows: snap.OpenForeshadows,
		}
		return nil
	})
	return result, err
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}
