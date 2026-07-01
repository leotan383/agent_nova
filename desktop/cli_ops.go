package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/backup"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/doctor"
	memorypkg "github.com/tanlian/agent_nova/internal/memory"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/workflows"
)

// DoctorFindingDTO 项目诊断项。
type DoctorFindingDTO struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Fix     string `json:"fix,omitempty"`
}

// DoctorReportDTO 项目体检报告（nova doctor）。
type DoctorReportDTO struct {
	OK       bool               `json:"ok"`
	Phase    string             `json:"phase"`
	Findings []DoctorFindingDTO `json:"findings"`
}

// PreflightDTO 写章前预检（nova preflight）。
type PreflightDTO struct {
	OK     bool     `json:"ok"`
	Title  string   `json:"title"`
	Phase  string   `json:"phase"`
	Issues []string `json:"issues"`
}

// LearnResultDTO 从反馈学习记忆结果（nova learn）。
type LearnResultDTO struct {
	OK       bool   `json:"ok"`
	Summary  string `json:"summary"`
	MemoryID string `json:"memory_id,omitempty"`
	Category string `json:"category,omitempty"`
	Subject  string `json:"subject,omitempty"`
}

// BackupItemDTO 备份目录项。
type BackupItemDTO struct {
	Name string `json:"name"`
}

// BootstrapResultDTO 设定集→记忆回填结果。
type BootstrapResultDTO struct {
	InsertedCount int `json:"inserted_count"`
}

// LearnFromFeedback 将作者反馈沉淀为长期记忆。
func (a *App) LearnFromFeedback(content string) (LearnResultDTO, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return LearnResultDTO{}, fmt.Errorf("反馈内容不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return LearnResultDTO{}, err
	}
	var out LearnResultDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := app.RequireAPIKey(actx.Config); err != nil {
			return err
		}
		wf := workflows.NewLearnWorkflow(actx.Config, actx.Project, actx.Store)
		rep, err := wf.Learn(context.Background(), actx.Store, content, actx.Project.Meta.CurrentChapter)
		if err != nil {
			return err
		}
		out.OK = rep.Status == report.StatusDone
		out.Summary = rep.Summary
		if len(rep.Artifacts) > 0 {
			out.MemoryID = rep.Artifacts[0]
		}
		return nil
	})
	return out, err
}

// RunProjectDoctor 运行项目结构诊断。
func (a *App) RunProjectDoctor(deep bool) (DoctorReportDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return DoctorReportDTO{}, err
	}
	var out DoctorReportDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		rep := doctor.Check(actx.Project, actx.Store, deep)
		out = DoctorReportDTO{OK: rep.OK, Phase: rep.Phase, Findings: make([]DoctorFindingDTO, len(rep.Findings))}
		for i, f := range rep.Findings {
			out.Findings[i] = DoctorFindingDTO{Level: f.Level, Message: f.Message, Fix: f.Fix}
		}
		return nil
	})
	return out, err
}

// RunPreflight 写章前预检（API Key + 项目结构）。
func (a *App) RunPreflight() (PreflightDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return PreflightDTO{}, err
	}
	var out PreflightDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		var issues []string
		if actx.Config.OpenAIAPIKey == "" {
			issues = append(issues, "OPENAI_API_KEY 未配置")
		}
		rep := doctor.Check(actx.Project, actx.Store, false)
		out.Title = actx.Project.Meta.Title
		out.Phase = actx.Project.Meta.Phase
		if !rep.OK {
			for _, f := range rep.Findings {
				if f.Level == "error" {
					issues = append(issues, f.Message)
				}
			}
		}
		out.Issues = issues
		out.OK = len(issues) == 0
		return nil
	})
	if err != nil {
		cfg, cfgErr := config.Load()
		if cfgErr == nil && cfg.OpenAIAPIKey == "" {
			return PreflightDTO{OK: false, Issues: []string{"OPENAI_API_KEY 未配置"}}, nil
		}
		return PreflightDTO{}, err
	}
	return out, nil
}

// CreateProjectBackup 创建项目备份。
func (a *App) CreateProjectBackup(label string) (BackupItemDTO, error) {
	if strings.TrimSpace(label) == "" {
		label = "manual"
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return BackupItemDTO{}, err
	}
	var name BackupItemDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		before, _ := backup.List(actx.Project)
		if err := backup.Create(actx.Project, label); err != nil {
			return err
		}
		after, err := backup.List(actx.Project)
		if err != nil {
			return err
		}
		name.Name = diffNewBackup(before, after)
		return nil
	})
	return name, err
}

// ListProjectBackups 列出备份目录。
func (a *App) ListProjectBackups() ([]BackupItemDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	var out []BackupItemDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		items, err := backup.List(actx.Project)
		if err != nil {
			return err
		}
		out = make([]BackupItemDTO, len(items))
		for i, n := range items {
			out[i] = BackupItemDTO{Name: n}
		}
		return nil
	})
	return out, err
}

// RestoreProjectBackup 从备份恢复（覆盖正文/设定等）。
func (a *App) RestoreProjectBackup(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("请指定备份名称")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := backup.Restore(actx.Project, name); err != nil {
			return err
		}
		a.session.invalidate()
		return a.syncChaptersFromDisk(actx)
	})
}

// BootstrapMemories 从设定集回填 world 类记忆。
func (a *App) BootstrapMemories() (BootstrapResultDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return BootstrapResultDTO{}, err
	}
	var out BootstrapResultDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		n, err := memorypkg.BootstrapFromSettings(actx.Project, actx.Store)
		if err != nil {
			return err
		}
		out.InsertedCount = n
		return nil
	})
	return out, err
}

func diffNewBackup(before, after []string) string {
	seen := map[string]struct{}{}
	for _, b := range before {
		seen[b] = struct{}{}
	}
	for _, a := range after {
		if _, ok := seen[a]; !ok {
			return a
		}
	}
	if len(after) > 0 {
		return after[len(after)-1]
	}
	return ""
}
