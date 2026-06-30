package pipeline

import (
	"fmt"
	"os"
	"strings"

	contextbuilder "github.com/tanlian/agent_nova/internal/context"
	"github.com/tanlian/agent_nova/internal/index"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// GateCheckItem 单项写前/写后检查结果。
type GateCheckItem struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`
	Blocking bool   `json:"blocking"`
}

// GateReport 结构化 Gate 报告，供 CLI / 桌面端展示。
type GateReport struct {
	OK      bool            `json:"ok"`
	Chapter int             `json:"chapter"`
	Volume  int             `json:"volume"`
	Checks  []GateCheckItem `json:"checks"`
}

// BuildGateReport 组装写前检查清单（blocking 项不通过则 OK=false）。
func BuildGateReport(p *project.Project, st *store.Store, chapter, volume int) GateReport {
	if volume <= 0 {
		volume = 1
	}
	if chapter <= 0 {
		chapter = 1
	}
	report := GateReport{Chapter: chapter, Volume: volume, OK: true}

	phaseOK := p.Meta.Phase == project.PhaseWriting || p.Meta.Phase == project.PhasePlanning
	phaseDetail := p.Meta.Phase
	if !phaseOK {
		phaseDetail += "（需为 planning 或 writing）"
	}
	report.Checks = append(report.Checks, GateCheckItem{
		Key: "phase", Label: "项目阶段", OK: phaseOK, Detail: phaseDetail, Blocking: true,
	})

	volPath := p.VolumeOutlinePath(volume)
	if _, err := os.Stat(volPath); err != nil {
		report.Checks = append(report.Checks, GateCheckItem{
			Key: "volume_outline", Label: "卷纲", OK: false,
			Detail: fmt.Sprintf("缺少 大纲/第%02d卷.md，请先 nova plan %d", volume, volume),
			Blocking: true,
		})
	} else {
		report.Checks = append(report.Checks, GateCheckItem{
			Key: "volume_outline", Label: "卷纲", OK: true,
			Detail: fmt.Sprintf("第 %d 卷卷纲已就绪", volume), Blocking: true,
		})
	}

	if chapter > 1 {
		prev := chapter - 1
		if _, err := os.Stat(p.SummaryPath(prev)); err != nil {
			report.Checks = append(report.Checks, GateCheckItem{
				Key: "prev_summary", Label: "上章摘要", OK: false,
				Detail: fmt.Sprintf("第 %d 章摘要缺失，长篇连贯性风险", prev), Blocking: true,
			})
		} else {
			report.Checks = append(report.Checks, GateCheckItem{
				Key: "prev_summary", Label: "上章摘要", OK: true,
				Detail: fmt.Sprintf("第 %d 章摘要已存在", prev), Blocking: true,
			})
		}
	}

	if st != nil {
		stale := st.CheckIndexStale(p.ChaptersDir())
		if stale.Stale {
			idx := index.New(p, st)
			if err := idx.RebuildChapters(0); err == nil {
				stale = st.CheckIndexStale(p.ChaptersDir())
			}
		}
		detail := "索引与正文同步"
		if stale.Stale {
			detail = strings.Join(stale.Issues, "；")
		}
		report.Checks = append(report.Checks, GateCheckItem{
			Key: "index", Label: "章节索引", OK: !stale.Stale, Detail: detail, Blocking: true,
		})
	}

	cb := contextbuilder.Builder{Proj: p, Store: st}
	snap, _ := cb.Build(chapter, volume)
	chOutline := strings.TrimSpace(snap.ChapterOutline)
	chOK := chOutline != ""
	chDetail := "已从卷纲提取本章任务"
	if !chOK {
		chDetail = "卷纲中未找到本章段落，写章易跑题"
	}
	report.Checks = append(report.Checks, GateCheckItem{
		Key: "chapter_outline", Label: "本章章纲", OK: chOK, Detail: chDetail, Blocking: false,
	})

	if st != nil {
		fs, _ := st.ListForeshadows("open")
		report.Checks = append(report.Checks, GateCheckItem{
			Key: "foreshadows", Label: "Open 伏笔", OK: true,
			Detail: fmt.Sprintf("当前 %d 条未回收（写章时可择机处理）", len(fs)), Blocking: false,
		})
	}

	for _, c := range report.Checks {
		if !c.OK && c.Blocking {
			report.OK = false
		}
	}
	return report
}
