package main

import (
	"context"
	"fmt"
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
	eventEntityHistoryStatus = "entity_history:status"
	eventEntityHistoryDone   = "entity_history:done"
	eventEntityHistoryError  = "entity_history:error"
)

// EntityHistoryBackfillJobInfo 实体历史回溯任务。
type EntityHistoryBackfillJobInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type entityHistoryBackfillJob struct {
	info   EntityHistoryBackfillJobInfo
	cancel context.CancelFunc
}

type entityHistoryBackfillManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*entityHistoryBackfillJob
}

func newEntityHistoryBackfillManager(app *App) *entityHistoryBackfillManager {
	return &entityHistoryBackfillManager{app: app, jobs: map[string]*entityHistoryBackfillJob{}}
}

// StartEntityHistoryBackfill 从已写章节重新提取实体状态，补齐状态时间线。
func (a *App) StartEntityHistoryBackfill() (EntityHistoryBackfillJobInfo, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return EntityHistoryBackfillJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return EntityHistoryBackfillJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return EntityHistoryBackfillJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return EntityHistoryBackfillJobInfo{}, err
	}

	a.entityHistoryBackfill.mu.Lock()
	for _, j := range a.entityHistoryBackfill.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.entityHistoryBackfill.mu.Unlock()
			return EntityHistoryBackfillJobInfo{}, fmt.Errorf("已有历史回溯任务进行中")
		}
	}

	id := fmt.Sprintf("entity-history-%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &entityHistoryBackfillJob{
		info:   EntityHistoryBackfillJobInfo{ID: id, Status: "pending"},
		cancel: cancel,
	}
	a.entityHistoryBackfill.jobs[id] = job
	a.entityHistoryBackfill.mu.Unlock()

	a.emitEntityHistoryStatus(id, "pending", "准备回溯…")

	go func(projectRoot string) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failEntityHistoryBackfill(id, err.Error())
			return
		}
		defer actx.Close()

		reg := tools.NewRegistry()
		reg.BindProject(actx.Project.Root, actx.Store)
		ag := agent.New(agent.Options{Config: actx.Config, Registry: reg})

		a.emitEntityHistoryStatus(id, "running", "正在回溯历史章节…")
		result, err := workflows.BackfillEntityStateHistory(ctx, ag, actx.Store, actx.Project, 0, func(chapter int, message string) {
			a.emitEntityHistoryStatus(id, "running", message)
			_ = chapter
		})

		a.entityHistoryBackfill.mu.Lock()
		if j, ok := a.entityHistoryBackfill.jobs[id]; ok {
			if err != nil {
				j.info.Status = "failed"
			} else {
				j.info.Status = "done"
			}
		}
		a.entityHistoryBackfill.mu.Unlock()

		if err != nil {
			a.failEntityHistoryBackfill(id, err.Error())
			return
		}
		a.emitEntityHistoryDone(id, result.Processed, result.Skipped)
	}(root)

	return job.info, nil
}

// GetActiveEntityHistoryBackfillJob 返回进行中的回溯任务。
func (a *App) GetActiveEntityHistoryBackfillJob() ActiveEntityHistoryBackfillJobDTO {
	a.entityHistoryBackfill.mu.Lock()
	defer a.entityHistoryBackfill.mu.Unlock()
	for _, j := range a.entityHistoryBackfill.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return ActiveEntityHistoryBackfillJobDTO{Active: true, Job: j.info}
		}
	}
	return ActiveEntityHistoryBackfillJobDTO{}
}

// ActiveEntityHistoryBackfillJobDTO 进行中的回溯任务。
type ActiveEntityHistoryBackfillJobDTO struct {
	Active bool                          `json:"active"`
	Job    EntityHistoryBackfillJobInfo `json:"job"`
}

func (a *App) emitEntityHistoryStatus(jobID, status, message string) {
	runtime.EventsEmit(a.ctx, eventEntityHistoryStatus, map[string]any{
		"job_id": jobID, "status": status, "message": message,
	})
}

func (a *App) emitEntityHistoryDone(jobID string, chaptersProcessed int, skipped []string) {
	runtime.EventsEmit(a.ctx, eventEntityHistoryDone, map[string]any{
		"job_id": jobID, "chapters_processed": chaptersProcessed, "skipped": skipped,
	})
}

func (a *App) failEntityHistoryBackfill(jobID, errMsg string) {
	a.entityHistoryBackfill.mu.Lock()
	if j, ok := a.entityHistoryBackfill.jobs[jobID]; ok {
		j.info.Status = "failed"
	}
	a.entityHistoryBackfill.mu.Unlock()
	runtime.EventsEmit(a.ctx, eventEntityHistoryError, map[string]any{
		"job_id": jobID, "error": errMsg,
	})
}
