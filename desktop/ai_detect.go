package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/workflows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventAIDetectStatus = "ai_detect:status"
	eventAIDetectDone   = "ai_detect:done"
	eventAIDetectError  = "ai_detect:error"
)

// StartAIDetectInput AI 味检测请求。
type StartAIDetectInput struct {
	Chapter int `json:"chapter"`
}

// AIDetectJobInfo AI 味检测任务信息。
type AIDetectJobInfo struct {
	ID      string `json:"id"`
	Chapter int    `json:"chapter"`
	Status  string `json:"status"`
}

// AIDetectReportDTO AI 味检测结果。
type AIDetectReportDTO struct {
	Stage     string   `json:"stage"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Artifacts []string `json:"artifacts,omitempty"`
}

// ChapterAIDetectMetricsDTO AI 味检测结构化指标。
type ChapterAIDetectMetricsDTO struct {
	Chapter     int                    `json:"chapter"`
	Exists      bool                   `json:"exists"`
	AIScore     float64                `json:"ai_score"`
	HumanScore  float64                `json:"human_score"`
	RiskLevel   string                 `json:"risk_level"`
	Publishable bool                   `json:"publishable"`
	Signals     []string               `json:"signals"`
	Hotspots    []AIDetectHotspotDTO   `json:"hotspots"`
	ReportBody  string                 `json:"report_body,omitempty"`
}

// AIDetectHotspotDTO 高风险片段。
type AIDetectHotspotDTO struct {
	Excerpt string `json:"excerpt"`
	Reason  string `json:"reason"`
	Fix     string `json:"fix"`
}

type aiDetectJob struct {
	info   AIDetectJobInfo
	cancel context.CancelFunc
}

type aiDetectManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*aiDetectJob
}

func newAIDetectManager(app *App) *aiDetectManager {
	return &aiDetectManager{app: app, jobs: map[string]*aiDetectJob{}}
}

// StartAIDetectChapter 异步检测章节 AI 味。
func (a *App) StartAIDetectChapter(in StartAIDetectInput) (AIDetectJobInfo, error) {
	if in.Chapter <= 0 {
		return AIDetectJobInfo{}, fmt.Errorf("请指定有效章号")
	}

	reg, err := a.loadRegistry()
	if err != nil {
		return AIDetectJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return AIDetectJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return AIDetectJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return AIDetectJobInfo{}, err
	}

	a.aiDetect.mu.Lock()
	for _, j := range a.aiDetect.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.aiDetect.mu.Unlock()
			return AIDetectJobInfo{}, fmt.Errorf("已有 AI 味检测任务进行中（第 %d 章）", j.info.Chapter)
		}
	}

	id := fmt.Sprintf("ai-detect-%d-%d", in.Chapter, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &aiDetectJob{
		info: AIDetectJobInfo{
			ID: id, Chapter: in.Chapter, Status: "pending",
		},
		cancel: cancel,
	}
	a.aiDetect.jobs[id] = job
	a.aiDetect.mu.Unlock()

	a.emitAIDetectStatus(id, in.Chapter, "pending", "")

	go func(projectRoot string, chapter int) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failAIDetectJob(id, chapter, err.Error())
			return
		}
		defer actx.Close()

		a.emitAIDetectStatus(id, chapter, "running", "正在分析 AI 味…")

		wf := workflows.NewAIDetectWorkflow(actx.Config, actx.Project, actx.Store)
		rep, err := wf.DetectChapter(ctx, actx.Project, chapter)

		a.aiDetect.mu.Lock()
		if j, ok := a.aiDetect.jobs[id]; ok {
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
		a.aiDetect.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitAIDetectStatus(id, chapter, "cancelled", "已取消")
				return
			}
			a.emitAIDetectError(id, chapter, err.Error())
			a.emitAIDetectStatus(id, chapter, "failed", err.Error())
			return
		}
		if rep != nil && rep.Status == report.StatusFailed {
			a.emitAIDetectError(id, chapter, rep.Summary)
			a.emitAIDetectStatus(id, chapter, "failed", rep.Summary)
			return
		}
		a.emitAIDetectDone(id, chapter, rep)
		a.emitAIDetectStatus(id, chapter, "done", rep.Summary)
	}(root, in.Chapter)

	return job.info, nil
}

func (a *App) failAIDetectJob(id string, chapter int, errMsg string) {
	a.aiDetect.mu.Lock()
	if j, ok := a.aiDetect.jobs[id]; ok {
		j.info.Status = "failed"
	}
	a.aiDetect.mu.Unlock()
	a.emitAIDetectError(id, chapter, errMsg)
	a.emitAIDetectStatus(id, chapter, "failed", errMsg)
}

// CancelAIDetectChapter 取消进行中的 AI 味检测。
func (a *App) CancelAIDetectChapter(jobID string) error {
	a.aiDetect.mu.Lock()
	defer a.aiDetect.mu.Unlock()
	job, ok := a.aiDetect.jobs[jobID]
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

// GetActiveAIDetectJob 返回进行中的 AI 味检测任务。
func (a *App) GetActiveAIDetectJob() (ActiveAIDetectJobDTO, error) {
	a.aiDetect.mu.Lock()
	defer a.aiDetect.mu.Unlock()
	for _, j := range a.aiDetect.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return ActiveAIDetectJobDTO{Active: true, Job: j.info}, nil
		}
	}
	return ActiveAIDetectJobDTO{}, nil
}

// ActiveAIDetectJobDTO 进行中的 AI 味检测任务。
type ActiveAIDetectJobDTO struct {
	Active bool            `json:"active"`
	Job    AIDetectJobInfo `json:"job"`
}

// GetChapterAIDetectMetrics 读取章节 AI 味检测指标。
func (a *App) GetChapterAIDetectMetrics(chapter int) (ChapterAIDetectMetricsDTO, error) {
	if chapter <= 0 {
		return ChapterAIDetectMetricsDTO{}, fmt.Errorf("无效章号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return ChapterAIDetectMetricsDTO{}, err
	}
	var out ChapterAIDetectMetricsDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		path := actx.Project.AICheckPath(chapter)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		body := string(data)
		if strings.TrimSpace(body) == "" {
			return nil
		}
		metrics := parseAIDetectMetricsFromReport(body)
		out = ChapterAIDetectMetricsDTO{
			Chapter: chapter, Exists: true,
			AIScore: metrics.AIScore, HumanScore: metrics.HumanScore,
			RiskLevel: metrics.RiskLevel, Publishable: metrics.Publishable,
			Signals: metrics.Signals, ReportBody: body,
		}
		for _, h := range metrics.Hotspots {
			out.Hotspots = append(out.Hotspots, AIDetectHotspotDTO{
				Excerpt: h.Excerpt, Reason: h.Reason, Fix: h.Fix,
			})
		}
		return nil
	})
	return out, err
}

func parseAIDetectMetricsFromReport(content string) workflows.AIDetectMetrics {
	jsonRaw, err := agent.ExtractJSONBlock(content)
	if err != nil {
		return workflows.AIDetectMetrics{}
	}
	var m workflows.AIDetectMetrics
	_ = json.Unmarshal([]byte(jsonRaw), &m)
	return m
}

func (a *App) emitAIDetectStatus(jobID string, chapter int, status, message string) {
	runtime.EventsEmit(a.ctx, eventAIDetectStatus, map[string]any{
		"job_id": jobID, "chapter": chapter, "status": status, "message": message,
	})
}

func (a *App) emitAIDetectError(jobID string, chapter int, errMsg string) {
	runtime.EventsEmit(a.ctx, eventAIDetectError, map[string]any{
		"job_id": jobID, "chapter": chapter, "error": errMsg,
	})
}

func (a *App) emitAIDetectDone(jobID string, chapter int, rep *report.Report) {
	dto := AIDetectReportDTO{
		Stage: rep.Stage, Status: string(rep.Status), Summary: rep.Summary,
		Artifacts: rep.Artifacts,
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventAIDetectDone, map[string]any{
		"job_id": jobID, "chapter": chapter, "report": string(raw),
	})
}
