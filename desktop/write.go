package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/usage"
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
	Chapter         int  `json:"chapter"`
	EndChapter      int  `json:"end_chapter"` // 0 或与 chapter 相同表示单章；大于 chapter 时连续写多章
	Volume          int  `json:"volume"`
	Resume          bool `json:"resume"`
	ContinueOnError bool `json:"continue_on_error"`
}

// WriteJobInfo 写章任务信息（立即返回，进度走事件）。
type WriteJobInfo struct {
	ID            string `json:"id"`
	Chapter       int    `json:"chapter"`
	Volume        int    `json:"volume"`
	Status        string `json:"status"`
	TotalInBatch  int    `json:"total_in_batch"`
	BatchIndex    int    `json:"batch_index"`
}

// WriteJobStateDTO 写章任务实时状态（用于 UI 重连恢复流式内容）。
type WriteJobStateDTO struct {
	StreamText  string `json:"stream_text"`
	Step        string `json:"step"`
	StepMessage string `json:"step_message"`
}

// TokenUsageDTO LLM token 用量。
type TokenUsageDTO struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
}

// WriteReportDTO 写章结果。
type WriteReportDTO struct {
	Stage      string         `json:"stage"`
	Status     string         `json:"status"`
	Summary    string         `json:"summary"`
	Artifacts  []string       `json:"artifacts,omitempty"`
	Issues     []string       `json:"issues,omitempty"`
	NextSteps  []string       `json:"next_steps,omitempty"`
	TokenUsage *TokenUsageDTO `json:"token_usage,omitempty"`
}

// ProjectTokenUsageDTO 项目累计 token 用量。
type ProjectTokenUsageDTO struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	WriteRuns        int     `json:"write_runs"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
}

type writeJob struct {
	info   WriteJobInfo
	cancel context.CancelFunc
	mu     sync.Mutex
	buf    strings.Builder
	step   string
	stepMsg string
	chapters        []int
	chapterIdx      int
	continueOnError bool
	volume          int
	resume          bool
}

func (j *writeJob) appendDelta(delta string) {
	j.mu.Lock()
	j.buf.WriteString(delta)
	j.mu.Unlock()
}

func (j *writeJob) setStep(step, message string) {
	j.mu.Lock()
	j.step = step
	j.stepMsg = message
	j.mu.Unlock()
}

func (j *writeJob) snapshot() WriteJobStateDTO {
	j.mu.Lock()
	defer j.mu.Unlock()
	return WriteJobStateDTO{
		StreamText:  j.buf.String(),
		Step:        j.step,
		StepMessage: j.stepMsg,
	}
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
	end := in.Chapter
	if in.EndChapter > in.Chapter {
		end = in.EndChapter
	} else if in.EndChapter > 0 && in.EndChapter < in.Chapter {
		return WriteJobInfo{}, fmt.Errorf("结束章号不能小于起始章号")
	}
	var chapters []int
	for ch := in.Chapter; ch <= end; ch++ {
		chapters = append(chapters, ch)
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
		chapters: chapters, chapterIdx: 0,
		continueOnError: in.ContinueOnError, volume: in.Volume, resume: in.Resume,
		info: WriteJobInfo{
			ID: id, Chapter: chapters[0], Volume: in.Volume, Status: "pending",
			TotalInBatch: len(chapters), BatchIndex: 1,
		},
		cancel: cancel,
	}
	a.write.jobs[id] = job
	a.write.mu.Unlock()

	a.emitWriteStatus(id, chapters[0], "pending", "")

	a.session.invalidate()

	go a.runWriteBatch(ctx, root, id, job)

	return job.info, nil
}

func (a *App) runWriteBatch(ctx context.Context, projectRoot, id string, job *writeJob) {
	defer job.cancel()
	for i, chapter := range job.chapters {
		select {
		case <-ctx.Done():
			return
		default:
		}

		a.write.mu.Lock()
		job.chapterIdx = i
		job.info.Chapter = chapter
		job.info.BatchIndex = i + 1
		job.info.TotalInBatch = len(job.chapters)
		job.mu.Lock()
		job.buf.Reset()
		job.step = ""
		job.stepMsg = ""
		job.mu.Unlock()
		a.write.mu.Unlock()

		a.emitWriteStatus(id, chapter, "running", fmt.Sprintf("批量 %d/%d", i+1, len(job.chapters)))

		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failJob(id, chapter, err.Error())
			return
		}

		wf := workflows.NewWriteWorkflow(actx.Config, actx.Project, actx.Store)
		useResume := job.resume && i == 0
		rep, err := wf.WriteChapter(ctx, actx.Project, actx.Store, workflows.WriteOptions{
			Chapter: chapter,
			Volume:  job.volume,
			Resume:  useResume,
			Stream:  true,
			OnDelta: func(delta string) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				job.appendDelta(delta)
				a.emitWriteDelta(id, chapter, delta)
				return nil
			},
			OnStep: func(step, message string) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				job.setStep(step, message)
				a.emitWriteStep(id, chapter, step, message)
				return nil
			},
		})
		actx.Close()

		if err != nil {
			if ctx.Err() != nil {
				a.setJobStatus(id, "cancelled")
				a.emitWriteStatus(id, chapter, "cancelled", "已取消")
				return
			}
			a.emitWriteError(id, chapter, err.Error())
			a.emitWriteStatus(id, chapter, "failed", err.Error())
			if job.continueOnError && i+1 < len(job.chapters) {
				continue
			}
			a.setJobStatus(id, "failed")
			return
		}

		a.emitWriteDone(id, chapter, rep, i+1 >= len(job.chapters), i+1, len(job.chapters))
		if rep.Status == report.StatusNeedsAction || rep.Status == report.StatusFailed {
			a.emitWriteStatus(id, chapter, "failed", rep.Summary)
			if !job.continueOnError {
				a.setJobStatus(id, "failed")
				return
			}
			continue
		}

		if i+1 < len(job.chapters) {
			continue
		}

		a.setJobStatus(id, "done")
		a.emitWriteStatus(id, chapter, "done", rep.Summary)
		a.session.invalidate()
		return
	}
	a.setJobStatus(id, "done")
	a.session.invalidate()
}

func (a *App) setJobStatus(id, status string) {
	a.write.mu.Lock()
	if j, ok := a.write.jobs[id]; ok {
		j.info.Status = status
	}
	a.write.mu.Unlock()
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

// GetWriteJobState 返回任务流式缓冲与步骤（切换 Tab 后恢复用）。
func (a *App) GetWriteJobState(jobID string) (WriteJobStateDTO, error) {
	a.write.mu.Lock()
	defer a.write.mu.Unlock()
	job, ok := a.write.jobs[jobID]
	if !ok {
		return WriteJobStateDTO{}, fmt.Errorf("任务不存在: %s", jobID)
	}
	return job.snapshot(), nil
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

// ActiveWriteJobDTO 进行中的写章任务（用于 UI 恢复）。
type ActiveWriteJobDTO struct {
	Active bool             `json:"active"`
	Job    WriteJobInfo     `json:"job"`
	State  WriteJobStateDTO `json:"state"`
}

// GetActiveWriteJob 返回进行中的写章任务，若无则 active=false。
func (a *App) GetActiveWriteJob() ActiveWriteJobDTO {
	a.write.mu.Lock()
	defer a.write.mu.Unlock()
	for _, j := range a.write.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			return ActiveWriteJobDTO{Active: true, Job: j.info, State: j.snapshot()}
		}
	}
	return ActiveWriteJobDTO{}
}

// GetProjectTokenUsage 返回当前小说累计 LLM token 用量。
func (a *App) GetProjectTokenUsage() (ProjectTokenUsageDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return ProjectTokenUsageDTO{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return ProjectTokenUsageDTO{}, errNoActiveProject
	}
	stats, err := usage.Load(root)
	if err != nil {
		return ProjectTokenUsageDTO{}, err
	}
	cfg, _ := config.Load()
	model := ""
	if cfg != nil {
		model = cfg.Model
	}
	total := stats.TotalTokens()
	return ProjectTokenUsageDTO{
		PromptTokens:     stats.PromptTokens,
		CompletionTokens: stats.CompletionTokens,
		TotalTokens:      total,
		WriteRuns:        stats.WriteRuns,
		EstimatedCostUSD: usage.EstimateCostUSD(model, stats.PromptTokens, stats.CompletionTokens),
	}, nil
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

func (a *App) emitWriteDone(jobID string, chapter int, rep *report.Report, batchComplete bool, batchIndex, totalInBatch int) {
	dto := WriteReportDTO{
		Stage: rep.Stage, Status: string(rep.Status), Summary: rep.Summary,
		Artifacts: rep.Artifacts, Issues: rep.Issues, NextSteps: rep.NextSteps,
	}
	if rep.TokenUsage != nil {
		dto.TokenUsage = &TokenUsageDTO{
			PromptTokens:     rep.TokenUsage.PromptTokens,
			CompletionTokens: rep.TokenUsage.CompletionTokens,
			TotalTokens:      rep.TokenUsage.TotalTokens,
			EstimatedCostUSD: rep.TokenUsage.EstimatedCostUSD,
		}
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventWriteDone, map[string]any{
		"job_id": jobID, "chapter": chapter, "report": string(raw),
		"batch_complete": batchComplete, "batch_index": batchIndex, "total_in_batch": totalInBatch,
	})
}
