package outline

import (
	"os"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// VolumeStructureHealth 单卷结构对照摘要。
type VolumeStructureHealth struct {
	Volume         int     `json:"volume"`
	HasOutlineFile bool    `json:"has_outline_file"`
	Summary        Summary `json:"summary"`
}

// StructureHealth 全书卷纲 ↔ 正文结构健康。
type StructureHealth struct {
	Volumes []VolumeStructureHealth `json:"volumes"`
	Total   Summary               `json:"total"`
}

// BuildStructureHealth 聚合各卷对照矩阵摘要。
func BuildStructureHealth(p *project.Project, st *store.Store) (StructureHealth, error) {
	vols, err := ListVolumeNumbers(p)
	if err != nil {
		return StructureHealth{}, err
	}
	if len(vols) == 0 {
		vols = []int{1}
	}
	out := StructureHealth{Volumes: make([]VolumeStructureHealth, 0, len(vols))}
	for _, v := range vols {
		m, err := BuildMatrix(p, st, v)
		if err != nil {
			return StructureHealth{}, err
		}
		_, statErr := os.Stat(p.VolumeOutlinePath(v))
		out.Volumes = append(out.Volumes, VolumeStructureHealth{
			Volume:         v,
			HasOutlineFile: statErr == nil,
			Summary:        m.Summary,
		})
		out.Total.TotalInOutline += m.Summary.TotalInOutline
		out.Total.Written += m.Summary.Written
		out.Total.Unwritten += m.Summary.Unwritten
		out.Total.Deviated += m.Summary.Deviated
		out.Total.Abandoned += m.Summary.Abandoned
		out.Total.Orphan += m.Summary.Orphan
	}
	return out, nil
}

// HasStructureIssues 是否存在需关注的结构偏差。
func (h StructureHealth) HasStructureIssues() bool {
	t := h.Total
	return t.Unwritten > 0 || t.Deviated > 0 || t.Orphan > 0
}

// FirstVolumeWithIssues 返回首个有偏差的卷号，无则 0。
func (h StructureHealth) FirstVolumeWithIssues() int {
	for _, v := range h.Volumes {
		s := v.Summary
		if s.Unwritten > 0 || s.Deviated > 0 || s.Orphan > 0 {
			return v.Volume
		}
	}
	return 0
}
