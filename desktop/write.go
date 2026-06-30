package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/workflows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventWriteDelta  = "write:delta"
	eventWriteStep   = "write:step"
	eventWriteStatus = "write:status"
	eventWriteDone   = "write:done"
	eventWriteError  = "write:error"
)

// StartWriteInput 写章请求。
type StartWriteInput struct {
	Chapter int  `json:"chapter"`
	Volume  int  `json:"volume"`
	Resume  bool `json:"resume"`
}

// WriteJobInfo 写章任务信息（立即返回，进度走事件）。
type WriteJobInfo struct {
	ID      string `json:"id"`
	Chapter int    `json:"chapter"`
	Volume  int    `json:"volume"`
	Status  string `json:"status"`
}

// WriteReportDTO 写章结果。
type WriteReportDTO struct {
	Stage     string   `json:"stage"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Artifacts []string `json:"artifacts,omitempty"`
	Issues    []string `json:"issues,omitempty"`
	NextSteps []string `json:"next_steps,omitempty"`
}

type writeJob struct {
	info   WriteJobInfo
	cancel context.CancelFunc
}

type writeManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*writeJob
}

func newWriteManager(app *App) *writeManager {
	return &writeManager{app: app, jobs: map[string]*writeJob{}}
}

// StartWriteChapter 异步写章，流式 delta 与步骤通过 Wails 事件推送。
func (a *App) StartWriteChapter(in StartWriteInput) (WriteJobInfo, error) {
	if in.Chapter <= 0 {
		return WriteJobInfo{}, fmt.Errorf("请指定有效章号")
	}
	if in.Volume <= 0 {
		in.Volume = 1
	}

	reg, err := a.loadRegistry()
	if err != nil {
		return WriteJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return WriteJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return WriteJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return WriteJobInfo{}, err
	}

	a.write.mu.Lock()
	for _, j := range a.write.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.write.mu.Unlock()
			return WriteJobInfo{}, fmt.Errorf("已有写章任务进行中（第 %d 章）", j.info.Chapter)
		}
	}

	id := fmt.Sprintf("write-%d-%d", in.Chapter, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &writeJob{
		info: WriteJobInfo{
			ID: id, Chapter: in.Chapter, Volume: in.Volume, Status: "pending",
		},
		cancel: cancel,
	}
	a.write.jobs[id] = job
	a.write.mu.Unlock()

	a.emitWriteStatus(id, in.Chapter, "pending", "")

	// 释放 UI 侧 session，写章期间使用独立连接，避免 SQLITE_BUSY
	a.session.invalidate()

	go func(projectRoot string, chapter, volume int, resume bool) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failJob(id, chapter, err.Error())
			return
		}
		defer actx.Close()

		a.emitWriteStatus(id, chapter, "running", "")

		wf := workflows.NewWriteWorkflow(actx.Config, actx.Project, actx.Store)
		rep, err := wf.WriteChapter(ctx, actx.Project, actx.Store, workflows.WriteOptions{
			Chapter: chapter,
			Volume:  volume,
			Resume:  resume,
			Stream:  true,
			OnDelta: func(delta string) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				a.emitWriteDelta(id, chapter, delta)
				return nil
			},
			OnStep: func(step, message string) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				a.emitWriteStep(id, chapter, step, message)
				return nil
			},
		})

		a.write.mu.Lock()
		if j, ok := a.write.jobs[id]; ok {
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
		a.write.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitWriteStatus(id, chapter, "cancelled", "已取消")
				return
			}
			a.emitWriteError(id, chapter, err.Error())
			a.emitWriteStatus(id, chapter, "failed", err.Error())
			return
		}
		a.emitWriteDone(id, chapter, rep)
		a.emitWriteStatus(id, chapter, "done", rep.Summary)
		a.session.invalidate()
	}(root, in.Chapter, in.Volume, in.Resume)

	return job.info, nil
}

func (a *App) failJob(id string, chapter int, errMsg string) {
	a.write.mu.Lock()
	if j, ok := a.write.jobs[id]; ok {
		j.info.Status = "failed"
	}
	a.write.mu.Unlock()
	a.emitWriteError(id, chapter, errMsg)
	a.emitWriteStatus(id, chapter, "failed", errMsg)
}

// CancelWriteChapter 取消进行中的写章任务。
func (a *App) CancelWriteChapter(jobID string) error {
	a.write.mu.Lock()
	defer a.write.mu.Unlock()
	job, ok := a.write.jobs[jobID]
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

// GetWriteJob 查询任务状态。
func (a *App) GetWriteJob(jobID string) (WriteJobInfo, error) {
	a.write.mu.Lock()
	defer a.write.mu.Unlock()
	job, ok := a.write.jobs[jobID]
	if !ok {
		return WriteJobInfo{}, fmt.Errorf("任务不存在: %s", jobID)
	}
	return job.info, nil
}

// IsWriteRunning 是否有进行中的写章。
func (a *App) IsWriteRunning() bool {
	a.write.mu.Lock()
	defer a.write.mu.Unlock()
	for _, j := range a.write.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return true
		}
	}
	return false
}

func (a *App) emitWriteDelta(jobID string, chapter int, delta string) {
	runtime.EventsEmit(a.ctx, eventWriteDelta, map[string]any{
		"job_id": jobID, "chapter": chapter, "delta": delta,
	})
}

func (a *App) emitWriteStep(jobID string, chapter int, step, message string) {
	runtime.EventsEmit(a.ctx, eventWriteStep, map[string]any{
		"job_id": jobID, "chapter": chapter, "step": step, "message": message,
	})
}

func (a *App) emitWriteStatus(jobID string, chapter int, status, message string) {
	runtime.EventsEmit(a.ctx, eventWriteStatus, map[string]any{
		"job_id": jobID, "chapter": chapter, "status": status, "message": message,
	})
}

func (a *App) emitWriteError(jobID string, chapter int, errMsg string) {
	runtime.EventsEmit(a.ctx, eventWriteError, map[string]any{
		"job_id": jobID, "chapter": chapter, "error": errMsg,
	})
}

func (a *App) emitWriteDone(jobID string, chapter int, rep *report.Report) {
	dto := WriteReportDTO{
		Stage: rep.Stage, Status: string(rep.Status), Summary: rep.Summary,
		Artifacts: rep.Artifacts, Issues: rep.Issues, NextSteps: rep.NextSteps,
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventWriteDone, map[string]any{
		"job_id": jobID, "chapter": chapter, "report": string(raw),
	})
}
