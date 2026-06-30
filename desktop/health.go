package main

import (
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/status"
)

// TodoItemDTO 概览待办项。
type TodoItemDTO struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Detail      string `json:"detail"`
	Severity    string `json:"severity"`
	Action      string `json:"action"`
	ActionParam string `json:"action_param,omitempty"`
}

// ProjectHealthDTO 项目健康报告。
type ProjectHealthDTO struct {
	OK                bool          `json:"ok"`
	SuggestedVolume   int           `json:"suggested_volume"`
	NextChapter       int           `json:"next_chapter"`
	HasVolumeOutline  bool          `json:"has_volume_outline"`
	VolumeOutlinePath string        `json:"volume_outline_path,omitempty"`
	Todos             []TodoItemDTO `json:"todos"`
}

// GetProjectHealth 返回当前小说的结构化待办与健康状态。
func (a *App) GetProjectHealth() (ProjectHealthDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return ProjectHealthDTO{}, err
	}
	var out ProjectHealthDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := a.syncChaptersFromDisk(actx); err != nil {
			return err
		}
		rep := status.BuildHealth(actx.Project, actx.Store)
		out = ProjectHealthDTO{
			OK:                rep.OK,
			SuggestedVolume:   rep.SuggestedVolume,
			NextChapter:       rep.NextChapter,
			HasVolumeOutline:  rep.HasVolumeOutline,
			VolumeOutlinePath: rep.VolumeOutlinePath,
			Todos:             make([]TodoItemDTO, len(rep.Todos)),
		}
		for i, t := range rep.Todos {
			out.Todos[i] = TodoItemDTO{
				ID: t.ID, Label: t.Label, Detail: t.Detail, Severity: t.Severity,
				Action: t.Action, ActionParam: t.ActionParam,
			}
		}
		return nil
	})
	return out, err
}
