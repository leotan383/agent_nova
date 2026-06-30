package main

import (
	"encoding/json"

	"github.com/tanlian/agent_nova/internal/app"
)

// EntityDTO 故事实体当前状态。
type EntityDTO struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	State       map[string]string `json:"state"`
	LastChapter int               `json:"last_chapter"`
}

// ListEntities 列出实体状态，type 可为 character/location/item，空为全部。
func (a *App) ListEntities(entityType string) (out []EntityDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
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
