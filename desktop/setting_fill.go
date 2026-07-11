package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/tools"
	"github.com/tanlian/agent_nova/internal/workflows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventSettingFillStatus = "setting_fill:status"
	eventSettingFillDone   = "setting_fill:done"
	eventSettingFillError  = "setting_fill:error"
)

// SettingFillJobInfo 设定填充任务。
type SettingFillJobInfo struct {
	ID        string `json:"id"`
	SettingID string `json:"setting_id"`
	Status    string `json:"status"`
}

// SettingFillDoneDTO 设定填充完成 payload。
type SettingFillDoneDTO struct {
	SettingID string `json:"setting_id"`
	Body      string `json:"body"`
}

type settingFillJob struct {
	info     SettingFillJobInfo
	settingID string
	cancel   context.CancelFunc
}

type settingFillManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*settingFillJob
}

func newSettingFillManager(app *App) *settingFillManager {
	return &settingFillManager{app: app, jobs: map[string]*settingFillJob{}}
}

// StartFillSettingFromPlot 根据已写剧情 AI 补全设定 Markdown。
func (a *App) StartFillSettingFromPlot(settingID string) (SettingFillJobInfo, error) {
	settingID = strings.TrimSpace(settingID)
	if settingID == "" {
		return SettingFillJobInfo{}, fmt.Errorf("请选择设定条目")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return SettingFillJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return SettingFillJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return SettingFillJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return SettingFillJobInfo{}, err
	}

	a.settingFill.mu.Lock()
	for _, j := range a.settingFill.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.settingFill.mu.Unlock()
			return SettingFillJobInfo{}, fmt.Errorf("已有设定填充任务进行中")
		}
	}
	id := fmt.Sprintf("setting-fill-%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &settingFillJob{
		info:      SettingFillJobInfo{ID: id, SettingID: settingID, Status: "pending"},
		settingID: settingID,
		cancel:    cancel,
	}
	a.settingFill.jobs[id] = job
	a.settingFill.mu.Unlock()

	a.emitSettingFillStatus(id, settingID, "pending", "准备分析已写剧情…")

	go func(projectRoot, wikiID string) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failSettingFillJob(id, wikiID, err.Error())
			return
		}
		defer actx.Close()

		reg := tools.NewRegistry()
		reg.BindProject(actx.Project.Root, actx.Store)
		ag := agent.New(agent.Options{Config: actx.Config, Registry: reg})

		a.emitSettingFillStatus(id, wikiID, "running", "正在根据正文摘要填充设定…")
		result, err := workflows.FillSettingFromPlot(ctx, ag, actx.Project, actx.Store, wikiID)

		a.settingFill.mu.Lock()
		if j, ok := a.settingFill.jobs[id]; ok {
			if err != nil {
				if ctx.Err() != nil {
					j.info.Status = "cancelled"
				} else {
					j.info.Status = "failed"
				}
			} else {
				j.info.Status = "done"
			}
		}
		a.settingFill.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitSettingFillStatus(id, wikiID, "cancelled", "已取消")
				return
			}
			a.failSettingFillJob(id, wikiID, err.Error())
			return
		}
		a.emitSettingFillDone(id, result.SettingID, result.NewBody)
	}(root, settingID)

	return job.info, nil
}

// GetActiveSettingFillJob 返回进行中的设定填充任务。
func (a *App) GetActiveSettingFillJob() (SettingFillJobInfo, error) {
	a.settingFill.mu.Lock()
	defer a.settingFill.mu.Unlock()
	for _, j := range a.settingFill.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return j.info, nil
		}
	}
	return SettingFillJobInfo{}, nil
}

// CancelFillSettingFromPlot 取消设定填充任务。
func (a *App) CancelFillSettingFromPlot(jobID string) error {
	a.settingFill.mu.Lock()
	j, ok := a.settingFill.jobs[jobID]
	if !ok {
		a.settingFill.mu.Unlock()
		return fmt.Errorf("任务不存在")
	}
	if j.info.Status != "running" && j.info.Status != "pending" {
		a.settingFill.mu.Unlock()
		return fmt.Errorf("任务已结束")
	}
	cancel := j.cancel
	a.settingFill.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (a *App) emitSettingFillStatus(jobID, settingID, status, message string) {
	runtime.EventsEmit(a.ctx, eventSettingFillStatus, map[string]any{
		"job_id": jobID, "setting_id": settingID, "status": status, "message": message,
	})
}

func (a *App) emitSettingFillDone(jobID, settingID, body string) {
	runtime.EventsEmit(a.ctx, eventSettingFillDone, map[string]any{
		"job_id": jobID, "setting_id": settingID, "body": body,
	})
}

func (a *App) failSettingFillJob(jobID, settingID, errMsg string) {
	a.settingFill.mu.Lock()
	if j, ok := a.settingFill.jobs[jobID]; ok {
		j.info.Status = "failed"
	}
	a.settingFill.mu.Unlock()
	runtime.EventsEmit(a.ctx, eventSettingFillError, map[string]any{
		"job_id": jobID, "setting_id": settingID, "error": errMsg,
	})
}
