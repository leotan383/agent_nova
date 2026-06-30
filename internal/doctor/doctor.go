package doctor

import (
	"os"
	"path/filepath"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

type Finding struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

type Report struct {
	Phase    string    `json:"phase"`
	Findings []Finding `json:"findings"`
	OK       bool      `json:"ok"`
}

func Check(p *project.Project, st *store.Store, deep bool) Report {
	r := Report{Phase: p.Meta.Phase, OK: true}
	checkDir := func(path, fix string) {
		if _, err := os.Stat(path); err != nil {
			r.Findings = append(r.Findings, Finding{Level: "error", Message: "缺失目录/文件: " + path, Fix: fix})
			r.OK = false
		}
	}
	checkDir(p.SettingsDir(), "运行 nova init")
	checkDir(p.OutlineDir(), "运行 nova init")
	checkDir(filepath.Join(p.Root, project.MetaFile), "运行 nova init")
	if _, err := os.Stat(p.DBPath()); err != nil {
		r.Findings = append(r.Findings, Finding{Level: "warn", Message: "数据库未初始化", Fix: "运行 nova init 或 nova index rebuild"})
	}
	if p.Meta.Phase == project.PhaseWriting || deep {
		if _, err := os.Stat(p.VolumeOutlinePath(1)); err != nil {
			r.Findings = append(r.Findings, Finding{Level: "warn", Message: "第1卷纲缺失", Fix: "运行 nova plan 1"})
		}
	}
	if deep && st != nil {
		chs, _ := st.ListChapters()
		if len(chs) == 0 && p.Meta.CurrentChapter > 0 {
			r.Findings = append(r.Findings, Finding{Level: "warn", Message: "索引中无章节记录", Fix: "运行 nova index rebuild"})
		}
	}
	if len(r.Findings) == 0 {
		r.Findings = append(r.Findings, Finding{Level: "info", Message: "项目结构正常"})
	}
	return r
}

