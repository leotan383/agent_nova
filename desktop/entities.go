package main

import (
	"encoding/json"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/store"
)

// EntityDTO 故事实体当前状态。
type EntityDTO struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	State       map[string]string `json:"state"`
	LastChapter int               `json:"last_chapter"`
}

// MergeEntityDuplicates 合并括号变体重复实体，返回删除的重复条数。
func (a *App) MergeEntityDuplicates() (merged int, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return 0, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if actx.Store == nil {
			return nil
		}
		n, mergeErr := actx.Store.MergeDuplicateEntities()
		if mergeErr != nil {
			return mergeErr
		}
		merged = n
		return nil
	})
	return merged, err
}

// ListEntities 列出实体状态，type 可为 character/location/item，空为全部。
func (a *App) ListEntities(entityType string) (out []EntityDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if actx.Store != nil {
			if _, mergeErr := actx.Store.MergeDuplicateEntities(); mergeErr != nil {
				return mergeErr
			}
		}
		list, err := actx.Store.ListEntities(entityType, 200)
		if err != nil {
			return err
		}
		out = make([]EntityDTO, len(list))
		for i, e := range list {
			out[i] = EntityDTO{
				ID: e.ID, Type: e.Type, Name: e.Name, LastChapter: e.LastChapter,
				State: parseEntityState(e.StateJSON),
			}
		}
		return nil
	})
	return out, err
}

// EntityStateSnapshotDTO 实体历史状态快照。
type EntityStateSnapshotDTO struct {
	Chapter      int               `json:"chapter"`
	ChapterTitle string            `json:"chapter_title"`
	State        map[string]string `json:"state"`
	RecordedAt   string            `json:"recorded_at"`
	IsCurrent    bool              `json:"is_current"`
}

// GetEntityHistory 返回实体按章排列的状态历史（无历史时用当前状态兜底）。
func (a *App) GetEntityHistory(entityID string) (out []EntityStateSnapshotDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if actx.Store == nil {
			return nil
		}
		ent, findErr := actx.Store.FindEntity(entityID)
		if findErr != nil {
			return findErr
		}
		entityID = ent.ID
		snaps, listErr := actx.Store.ListEntityStateHistory(entityID)
		if listErr != nil {
			return listErr
		}
		if len(snaps) == 0 && ent.StateJSON != "" && ent.LastChapter > 0 {
			title := chapterTitle(actx.Store, ent.LastChapter)
			out = []EntityStateSnapshotDTO{{
				Chapter: ent.LastChapter, ChapterTitle: title,
				State: parseEntityState(ent.StateJSON), IsCurrent: true,
			}}
			return nil
		}
		out = make([]EntityStateSnapshotDTO, len(snaps))
		for i, snap := range snaps {
			out[i] = EntityStateSnapshotDTO{
				Chapter:      snap.Chapter,
				ChapterTitle: chapterTitle(actx.Store, snap.Chapter),
				State:        parseEntityState(snap.StateJSON),
				RecordedAt:   snap.RecordedAt,
				IsCurrent:    snap.Chapter == ent.LastChapter && i == len(snaps)-1,
			}
		}
		return nil
	})
	return out, err
}

func chapterTitle(st *store.Store, chapter int) string {
	if st == nil || chapter <= 0 {
		return ""
	}
	ch, err := st.GetChapter(chapter)
	if err != nil || ch == nil {
		return ""
	}
	return ch.Title
}

func parseEntityState(raw string) map[string]string {
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
