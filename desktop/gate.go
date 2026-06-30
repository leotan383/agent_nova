package main

import (
	"fmt"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/pipeline"
)

// GateCheckDTO 写前检查项。
type GateCheckDTO struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`
	Blocking bool   `json:"blocking"`
}

// WriteGateDTO 写章前 Gate 报告。
type WriteGateDTO struct {
	OK      bool           `json:"ok"`
	Chapter int            `json:"chapter"`
	Volume  int            `json:"volume"`
	Checks  []GateCheckDTO `json:"checks"`
}

// GetWriteGate 返回指定章节的写前检查清单。
func (a *App) GetWriteGate(chapter, volume int) (WriteGateDTO, error) {
	if chapter <= 0 {
		return WriteGateDTO{}, fmt.Errorf("无效章号")
	}
	if volume <= 0 {
		volume = 1
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return WriteGateDTO{}, err
	}
	var out WriteGateDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		rep := pipeline.BuildGateReport(actx.Project, actx.Store, chapter, volume)
		out = WriteGateDTO{
			OK: rep.OK, Chapter: rep.Chapter, Volume: rep.Volume,
			Checks: make([]GateCheckDTO, len(rep.Checks)),
		}
		for i, c := range rep.Checks {
			out.Checks[i] = GateCheckDTO{
				Key: c.Key, Label: c.Label, OK: c.OK, Detail: c.Detail, Blocking: c.Blocking,
			}
		}
		return nil
	})
	return out, err
}
