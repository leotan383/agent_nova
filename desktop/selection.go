package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/workflows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventSelectionDelta  = "selection:delta"
	eventSelectionDone   = "selection:done"
	eventSelectionError  = "selection:error"
	eventSelectionStatus = "selection:status"
)

// SelectionTransformInput 片段改写请求。
type SelectionTransformInput struct {
	Chapter      int    `json:"chapter"`
	Action       string `json:"action"`
	SelectedText string `json:"selected_text"`
	CustomPrompt string `json:"custom_prompt"`
}

// SelectionJobInfo 片段改写任务。
type SelectionJobInfo struct {
	ID      string `json:"id"`
	Chapter int    `json:"chapter"`
	Action  string `json:"action"`
	Status  string `json:"status"`
}

type selectionJob struct {
	info   SelectionJobInfo
	cancel context.CancelFunc
}

type selectionManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*selectionJob
}

func newSelectionManager(app *App) *selectionManager {
	return &selectionManager{app: app, jobs: map[string]*selectionJob{}}
}

var allowedSelectionActions = map[string]bool{
	"polish": true, "expand": true, "shorten": true, "dialogue": true, "custom": true,
}

// StartSelectionTransform 对选中片段发起 AI 改写（流式）。
func (a *App) StartSelectionTransform(in SelectionTransformInput) (SelectionJobInfo, error) {
	if in.Chapter <= 0 {
		return SelectionJobInfo{}, fmt.Errorf("无效章号")
	}
	if !allowedSelectionActions[in.Action] {
		return SelectionJobInfo{}, fmt.Errorf("不支持的操作：%s", in.Action)
	}
	if in.Action == "custom" && in.CustomPrompt == "" {
		return SelectionJobInfo{}, fmt.Errorf("请填写自定义改写要求")
	}
	selected := in.SelectedText
	if selected == "" {
		return SelectionJobInfo{}, fmt.Errorf("选中内容不能为空")
	}

	cfg, err := config.Load()
	if err != nil {
		return SelectionJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return SelectionJobInfo{}, err
	}

	a.selection.mu.Lock()
	for _, j := range a.selection.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.selection.mu.Unlock()
			return SelectionJobInfo{}, fmt.Errorf("已有片段改写任务进行中")
		}
	}

	id := fmt.Sprintf("selection-%d-%d", in.Chapter, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &selectionJob{
		info: SelectionJobInfo{
			ID: id, Chapter: in.Chapter, Action: in.Action, Status: "pending",
		},
		cancel: cancel,
	}
	a.selection.jobs[id] = job
	a.selection.mu.Unlock()

	a.emitSelectionStatus(id, in.Chapter, in.Action, "pending", "")

	projectRoot, err := a.activeProjectRoot()
	if err != nil {
		a.failSelectionJob(id, in.Chapter, in.Action, err.Error())
		return SelectionJobInfo{}, err
	}

	go func() {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failSelectionJob(id, in.Chapter, in.Action, err.Error())
			return
		}
		a.emitSelectionStatus(id, in.Chapter, in.Action, "running", "")

		wf := workflows.NewSelectionWorkflow(cfg, actx.Project, actx.Store)
		result, err := wf.TransformSelection(ctx, actx.Project, in.Chapter, in.Action, selected, in.CustomPrompt, func(delta string) error {
			a.emitSelectionDelta(id, in.Chapter, in.Action, delta)
			return nil
		})
		if err != nil {
			if ctx.Err() != nil {
				a.emitSelectionStatus(id, in.Chapter, in.Action, "cancelled", "已取消")
			} else {
				a.failSelectionJob(id, in.Chapter, in.Action, err.Error())
			}
			return
		}

		a.selection.mu.Lock()
		if j, ok := a.selection.jobs[id]; ok {
			j.info.Status = "done"
		}
		a.selection.mu.Unlock()

		a.emitSelectionDone(id, in.Chapter, in.Action, result)
	}()

	return job.info, nil
}

// CancelSelectionTransform 取消片段改写任务。
func (a *App) CancelSelectionTransform(jobID string) error {
	a.selection.mu.Lock()
	job, ok := a.selection.jobs[jobID]
	if !ok {
		a.selection.mu.Unlock()
		return fmt.Errorf("任务不存在")
	}
	if job.info.Status != "running" && job.info.Status != "pending" {
		a.selection.mu.Unlock()
		return fmt.Errorf("任务已结束")
	}
	job.cancel()
	job.info.Status = "cancelled"
	a.selection.mu.Unlock()
	return nil
}

func (a *App) failSelectionJob(id string, chapter int, action, errMsg string) {
	a.selection.mu.Lock()
	if j, ok := a.selection.jobs[id]; ok {
		j.info.Status = "failed"
	}
	a.selection.mu.Unlock()
	a.emitSelectionError(id, chapter, action, errMsg)
}

func (a *App) emitSelectionDelta(jobID string, chapter int, action, delta string) {
	runtime.EventsEmit(a.ctx, eventSelectionDelta, map[string]interface{}{
		"job_id": jobID, "chapter": chapter, "action": action, "delta": delta,
	})
}

func (a *App) emitSelectionStatus(jobID string, chapter int, action, status, message string) {
	runtime.EventsEmit(a.ctx, eventSelectionStatus, map[string]interface{}{
		"job_id": jobID, "chapter": chapter, "action": action, "status": status, "message": message,
	})
}

func (a *App) emitSelectionError(jobID string, chapter int, action, errMsg string) {
	runtime.EventsEmit(a.ctx, eventSelectionError, map[string]interface{}{
		"job_id": jobID, "chapter": chapter, "action": action, "error": errMsg,
	})
}

func (a *App) emitSelectionDone(jobID string, chapter int, action, content string) {
	runtime.EventsEmit(a.ctx, eventSelectionDone, map[string]interface{}{
		"job_id": jobID, "chapter": chapter, "action": action, "content": content,
	})
}
