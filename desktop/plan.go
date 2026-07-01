package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/index"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/workflows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventPlanStatus = "plan:status"
	eventPlanDone   = "plan:done"
	eventPlanError  = "plan:error"
)

// StartPlanInput 生成卷纲请求。
type StartPlanInput struct {
	Volume int `json:"volume"`
}

// PlanJobInfo 卷纲任务信息。
type PlanJobInfo struct {
	ID     string `json:"id"`
	Volume int    `json:"volume"`
	Status string `json:"status"`
}

// PlanReportDTO 卷纲生成结果。
type PlanReportDTO struct {
	Stage     string   `json:"stage"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Artifacts []string `json:"artifacts,omitempty"`
	NextSteps []string `json:"next_steps,omitempty"`
}

// VolumeOutlineDTO 卷纲内容与元数据。
type VolumeOutlineDTO struct {
	Volume int    `json:"volume"`
	Path   string `json:"path"`
	Body   string `json:"body"`
	Exists bool   `json:"exists"`
}

type planJob struct {
	info   PlanJobInfo
	cancel context.CancelFunc
}

type planManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*planJob
}

func newPlanManager(app *App) *planManager {
	return &planManager{app: app, jobs: map[string]*planJob{}}
}

// GetVolumeOutline 读取指定卷的卷纲 Markdown。
func (a *App) GetVolumeOutline(volume int) (VolumeOutlineDTO, error) {
	if volume <= 0 {
		return VolumeOutlineDTO{}, fmt.Errorf("无效卷号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return VolumeOutlineDTO{}, err
	}
	var out VolumeOutlineDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		path := actx.Project.VolumeOutlinePath(volume)
		out = VolumeOutlineDTO{Volume: volume, Path: path}
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		out.Body = string(data)
		out.Exists = true
		return nil
	})
	return out, err
}

// SaveVolumeOutline 保存卷纲 Markdown。
func (a *App) SaveVolumeOutline(volume int, body string) error {
	if volume <= 0 {
		return fmt.Errorf("无效卷号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		path := actx.Project.VolumeOutlinePath(volume)
		return os.WriteFile(path, []byte(body), 0o644)
	})
}

// RebuildProjectIndex 重建章节索引。
func (a *App) RebuildProjectIndex() error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		idx := index.New(actx.Project, actx.Store)
		return idx.RebuildChapters(0)
	})
}

// StartPlanVolume 异步生成卷纲。
func (a *App) StartPlanVolume(in StartPlanInput) (PlanJobInfo, error) {
	if in.Volume <= 0 {
		return PlanJobInfo{}, fmt.Errorf("请指定有效卷号")
	}

	reg, err := a.loadRegistry()
	if err != nil {
		return PlanJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return PlanJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return PlanJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return PlanJobInfo{}, err
	}

	a.plan.mu.Lock()
	for _, j := range a.plan.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.plan.mu.Unlock()
			return PlanJobInfo{}, fmt.Errorf("已有卷纲任务进行中（第 %d 卷）", j.info.Volume)
		}
	}

	id := fmt.Sprintf("plan-%d-%d", in.Volume, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &planJob{
		info: PlanJobInfo{
			ID: id, Volume: in.Volume, Status: "pending",
		},
		cancel: cancel,
	}
	a.plan.jobs[id] = job
	a.plan.mu.Unlock()

	a.emitPlanStatus(id, in.Volume, "pending", "")

	a.session.invalidate()

	go func(projectRoot string, volume int) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failPlanJob(id, volume, err.Error())
			return
		}
		defer actx.Close()

		a.emitPlanStatus(id, volume, "running", "正在生成卷纲…")

		wf := workflows.NewPlanWorkflow(actx.Config, actx.Project, actx.Store)
		rep, err := wf.PlanVolume(ctx, actx.Project, volume)

		a.plan.mu.Lock()
		if j, ok := a.plan.jobs[id]; ok {
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
		a.plan.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitPlanStatus(id, volume, "cancelled", "已取消")
				return
			}
			a.emitPlanError(id, volume, err.Error())
			a.emitPlanStatus(id, volume, "failed", err.Error())
			return
		}
		a.emitPlanDone(id, volume, rep)
		a.emitPlanStatus(id, volume, "done", rep.Summary)
		a.session.invalidate()
	}(root, in.Volume)

	return job.info, nil
}

func (a *App) failPlanJob(id string, volume int, errMsg string) {
	a.plan.mu.Lock()
	if j, ok := a.plan.jobs[id]; ok {
		j.info.Status = "failed"
	}
	a.plan.mu.Unlock()
	a.emitPlanError(id, volume, errMsg)
	a.emitPlanStatus(id, volume, "failed", errMsg)
}

// CancelPlanVolume 取消进行中的卷纲任务。
func (a *App) CancelPlanVolume(jobID string) error {
	a.plan.mu.Lock()
	defer a.plan.mu.Unlock()
	job, ok := a.plan.jobs[jobID]
	if !ok {
		return fmt.Errorf("任务不存在: %s", jobID)
	}
	if job.info.Status != "running" && job.info.Status != "pending" {
		return fmt.Errorf("任务已结束，无法取消")
	}
	job.cancel()
	job.info.Status = "cancelled"
	return nil
}

// IsPlanRunning 是否有进行中的卷纲任务。
func (a *App) IsPlanRunning() bool {
	a.plan.mu.Lock()
	defer a.plan.mu.Unlock()
	for _, j := range a.plan.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return true
		}
	}
	return false
}

// ActivePlanJobDTO 进行中的卷纲任务。
type ActivePlanJobDTO struct {
	Active bool        `json:"active"`
	Job    PlanJobInfo `json:"job"`
}

// GetActivePlanJob 返回进行中的卷纲任务。
func (a *App) GetActivePlanJob() ActivePlanJobDTO {
	a.plan.mu.Lock()
	defer a.plan.mu.Unlock()
	for _, j := range a.plan.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return ActivePlanJobDTO{Active: true, Job: j.info}
		}
	}
	return ActivePlanJobDTO{}
}

func (a *App) emitPlanStatus(jobID string, volume int, status, message string) {
	runtime.EventsEmit(a.ctx, eventPlanStatus, map[string]any{
		"job_id": jobID, "volume": volume, "status": status, "message": message,
	})
}

func (a *App) emitPlanError(jobID string, volume int, errMsg string) {
	runtime.EventsEmit(a.ctx, eventPlanError, map[string]any{
		"job_id": jobID, "volume": volume, "error": errMsg,
	})
}

func (a *App) emitPlanDone(jobID string, volume int, rep *report.Report) {
	dto := PlanReportDTO{
		Stage: rep.Stage, Status: string(rep.Status), Summary: rep.Summary,
		Artifacts: rep.Artifacts, NextSteps: rep.NextSteps,
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventPlanDone, map[string]any{
		"job_id": jobID, "volume": volume, "report": string(raw),
	})
}
