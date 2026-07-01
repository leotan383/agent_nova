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
	eventReviewStatus = "review:status"
	eventReviewDone   = "review:done"
	eventReviewError  = "review:error"
)

// StartReviewInput 审查请求。
type StartReviewInput struct {
	Chapter int `json:"chapter"`
}

// ReviewJobInfo 审查任务信息。
type ReviewJobInfo struct {
	ID      string `json:"id"`
	Chapter int    `json:"chapter"`
	Status  string `json:"status"`
}

// ReviewReportDTO 审查结果。
type ReviewReportDTO struct {
	Stage     string   `json:"stage"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Artifacts []string `json:"artifacts,omitempty"`
	Issues    []string `json:"issues,omitempty"`
	NextSteps []string `json:"next_steps,omitempty"`
}

type reviewJob struct {
	info   ReviewJobInfo
	cancel context.CancelFunc
}

type reviewManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*reviewJob
}

func newReviewManager(app *App) *reviewManager {
	return &reviewManager{app: app, jobs: map[string]*reviewJob{}}
}

// StartReviewChapter 异步审查章节。
func (a *App) StartReviewChapter(in StartReviewInput) (ReviewJobInfo, error) {
	if in.Chapter <= 0 {
		return ReviewJobInfo{}, fmt.Errorf("请指定有效章号")
	}

	reg, err := a.loadRegistry()
	if err != nil {
		return ReviewJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return ReviewJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return ReviewJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return ReviewJobInfo{}, err
	}

	a.review.mu.Lock()
	for _, j := range a.review.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.review.mu.Unlock()
			return ReviewJobInfo{}, fmt.Errorf("已有审查任务进行中（第 %d 章）", j.info.Chapter)
		}
	}

	id := fmt.Sprintf("review-%d-%d", in.Chapter, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &reviewJob{
		info: ReviewJobInfo{
			ID: id, Chapter: in.Chapter, Status: "pending",
		},
		cancel: cancel,
	}
	a.review.jobs[id] = job
	a.review.mu.Unlock()

	a.emitReviewStatus(id, in.Chapter, "pending", "")

	a.session.invalidate()

	go func(projectRoot string, chapter int) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failReviewJob(id, chapter, err.Error())
			return
		}
		defer actx.Close()

		a.emitReviewStatus(id, chapter, "running", "正在审查本章…")

		wf := workflows.NewReviewWorkflow(actx.Config, actx.Project, actx.Store)
		rep, err := wf.ReviewChapter(ctx, actx.Project, actx.Store, chapter)

		a.review.mu.Lock()
		if j, ok := a.review.jobs[id]; ok {
			if err != nil {
				if ctx.Err() != nil {
					j.info.Status = "cancelled"
				} else {
					j.info.Status = "failed"
				}
			} else if rep != nil && rep.Status == report.StatusFailed {
				j.info.Status = "failed"
			} else {
				j.info.Status = "done"
			}
		}
		a.review.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitReviewStatus(id, chapter, "cancelled", "已取消")
				return
			}
			a.emitReviewError(id, chapter, err.Error())
			a.emitReviewStatus(id, chapter, "failed", err.Error())
			return
		}
		if rep != nil && rep.Status == report.StatusFailed {
			msg := rep.Summary
			if len(rep.Issues) > 0 {
				msg = rep.Issues[0]
			}
			a.emitReviewError(id, chapter, msg)
			a.emitReviewStatus(id, chapter, "failed", msg)
			return
		}
		a.emitReviewDone(id, chapter, rep)
		a.emitReviewStatus(id, chapter, "done", rep.Summary)
		a.session.invalidate()
	}(root, in.Chapter)

	return job.info, nil
}

func (a *App) failReviewJob(id string, chapter int, errMsg string) {
	a.review.mu.Lock()
	if j, ok := a.review.jobs[id]; ok {
		j.info.Status = "failed"
	}
	a.review.mu.Unlock()
	a.emitReviewError(id, chapter, errMsg)
	a.emitReviewStatus(id, chapter, "failed", errMsg)
}

// CancelReviewChapter 取消进行中的审查任务。
func (a *App) CancelReviewChapter(jobID string) error {
	a.review.mu.Lock()
	defer a.review.mu.Unlock()
	job, ok := a.review.jobs[jobID]
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

// IsReviewRunning 是否有进行中的审查。
func (a *App) IsReviewRunning() bool {
	a.review.mu.Lock()
	defer a.review.mu.Unlock()
	for _, j := range a.review.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return true
		}
	}
	return false
}

// ActiveReviewJobDTO 进行中的审查任务。
type ActiveReviewJobDTO struct {
	Active bool          `json:"active"`
	Job    ReviewJobInfo `json:"job"`
}

// GetActiveReviewJob 返回进行中的审查任务。
func (a *App) GetActiveReviewJob() ActiveReviewJobDTO {
	a.review.mu.Lock()
	defer a.review.mu.Unlock()
	for _, j := range a.review.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return ActiveReviewJobDTO{Active: true, Job: j.info}
		}
	}
	return ActiveReviewJobDTO{}
}

// ChapterReviewMetricsDTO 章节审查结构化指标。
type ChapterReviewMetricsDTO struct {
	Chapter   int      `json:"chapter"`
	Exists    bool     `json:"exists"`
	HookScore float64  `json:"hook_score"`
	CoolPoint string   `json:"cool_point"`
	Debt      string   `json:"debt"`
	Issues    []string `json:"issues"`
}

// GetChapterReviewMetrics 读取章节审查指标（来自数据库）。
func (a *App) GetChapterReviewMetrics(chapter int) (ChapterReviewMetricsDTO, error) {
	if chapter <= 0 {
		return ChapterReviewMetricsDTO{}, fmt.Errorf("无效章号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return ChapterReviewMetricsDTO{}, err
	}
	var out ChapterReviewMetricsDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		r, err := actx.Store.GetReview(chapter)
		if err != nil {
			return nil
		}
		out = ChapterReviewMetricsDTO{
			Chapter: chapter, Exists: true,
			HookScore: r.HookScore, CoolPoint: r.CoolPoint, Debt: r.Debt,
		}
		if r.ReportJSON != "" {
			var payload struct {
				Issues []string `json:"issues"`
			}
			if json.Unmarshal([]byte(r.ReportJSON), &payload) == nil {
				out.Issues = payload.Issues
			}
		}
		return nil
	})
	return out, err
}

func (a *App) emitReviewStatus(jobID string, chapter int, status, message string) {
	runtime.EventsEmit(a.ctx, eventReviewStatus, map[string]any{
		"job_id": jobID, "chapter": chapter, "status": status, "message": message,
	})
}

func (a *App) emitReviewError(jobID string, chapter int, errMsg string) {
	runtime.EventsEmit(a.ctx, eventReviewError, map[string]any{
		"job_id": jobID, "chapter": chapter, "error": errMsg,
	})
}

func (a *App) emitReviewDone(jobID string, chapter int, rep *report.Report) {
	dto := ReviewReportDTO{
		Stage: rep.Stage, Status: string(rep.Status), Summary: rep.Summary,
		Artifacts: rep.Artifacts, Issues: rep.Issues, NextSteps: rep.NextSteps,
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventReviewDone, map[string]any{
		"job_id": jobID, "chapter": chapter, "report": string(raw),
	})
}
