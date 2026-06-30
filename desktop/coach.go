package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/workflows"
	openai "github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventCoachStream = "coach:stream"
	eventCoachDone   = "coach:done"
	eventCoachError  = "coach:error"
	eventReviseDelta  = "revise:delta"
	eventReviseDone   = "revise:done"
	eventReviseError  = "revise:error"
	eventReviseStatus = "revise:status"
)

// CoachTurnDTO 改稿对话一轮。
type CoachTurnDTO struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ReviseJobInfo 改稿生成任务。
type ReviseJobInfo struct {
	ID      string `json:"id"`
	Chapter int    `json:"chapter"`
	Status  string `json:"status"`
}

type coachSession struct {
	projectRoot string
	chapter     int
	messages    []openai.ChatCompletionMessage
}

type coachManager struct {
	app      *App
	mu       sync.Mutex
	sessions map[string]*coachSession
	chatting map[string]bool
}

type reviseJob struct {
	info   ReviseJobInfo
	cancel context.CancelFunc
}

type reviseManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*reviseJob
}

func newCoachManager(app *App) *coachManager {
	return &coachManager{app: app, sessions: map[string]*coachSession{}, chatting: map[string]bool{}}
}

func newReviseManager(app *App) *reviseManager {
	return &reviseManager{app: app, jobs: map[string]*reviseJob{}}
}

func coachKey(root string, chapter int) string {
	return fmt.Sprintf("%s:%d", root, chapter)
}

func toCoachDTOs(turns []workflows.CoachTurn) []CoachTurnDTO {
	out := make([]CoachTurnDTO, len(turns))
	for i, t := range turns {
		out[i] = CoachTurnDTO{Role: t.Role, Content: t.Content}
	}
	return out
}

func (a *App) getCoachSession(chapter int) (*coachSession, string, error) {
	root, err := a.activeProjectRoot()
	if err != nil {
		return nil, "", err
	}
	key := coachKey(root, chapter)
	a.coach.mu.Lock()
	sess, ok := a.coach.sessions[key]
	a.coach.mu.Unlock()
	if !ok {
		return nil, "", fmt.Errorf("尚未开始讨论")
	}
	if sess.projectRoot != root || sess.chapter != chapter {
		return nil, "", fmt.Errorf("会话已过期，请重新开启讨论")
	}
	return sess, key, nil
}

// SendChapterCoachMessage 发送改稿讨论消息（流式推送 coach:stream / coach:done）。
func (a *App) SendChapterCoachMessage(chapter int, message string) error {
	message = trimCoachInput(message)
	if message == "" {
		return fmt.Errorf("请输入消息")
	}
	if chapter <= 0 {
		return fmt.Errorf("无效章号")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return err
	}
	root, err := a.activeProjectRoot()
	if err != nil {
		return err
	}
	sess, _, _ := a.getCoachSession(chapter)
	projectRoot := root
	if sess != nil {
		projectRoot = sess.projectRoot
	}
	return a.runCoachChat(projectRoot, chapter, message, sess)
}

func (a *App) runCoachChat(root string, chapter int, message string, prev *coachSession) error {
	key := coachKey(root, chapter)
	a.coach.mu.Lock()
	if a.coach.chatting[key] {
		a.coach.mu.Unlock()
		return fmt.Errorf("顾问正在回复，请稍候")
	}
	a.coach.chatting[key] = true
	a.coach.mu.Unlock()

	a.session.invalidate()
	go func(projectRoot string, userMsg string, existing *coachSession) {
		defer func() {
			a.coach.mu.Lock()
			delete(a.coach.chatting, coachKey(projectRoot, chapter))
			a.coach.mu.Unlock()
		}()

		ctx := context.Background()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.emitCoachError(chapter, err.Error())
			return
		}
		defer actx.Close()

		wf := workflows.NewCoachWorkflow(actx.Config, actx.Project, actx.Store)
		onChunk := func(phase, delta string) error {
			a.emitCoachStream(chapter, phase, delta)
			return nil
		}

		msgs := existing
		if msgs == nil {
			var prepared []openai.ChatCompletionMessage
			if err := a.session.withActive(projectRoot, func(syncCtx *app.Context) error {
				if err := a.syncChaptersFromDisk(syncCtx); err != nil {
					return err
				}
				var prepErr error
				prepared, prepErr = wf.PrepareCoachMessages(syncCtx.Project, chapter)
				return prepErr
			}); err != nil {
				a.emitCoachError(chapter, err.Error())
				return
			}
			msgs = &coachSession{projectRoot: projectRoot, chapter: chapter, messages: prepared}
		}

		_, updated, err := wf.ChatStream(ctx, msgs.messages, userMsg, onChunk)
		if err != nil {
			a.emitCoachError(chapter, err.Error())
			return
		}

		a.coach.mu.Lock()
		a.coach.sessions[coachKey(projectRoot, chapter)] = &coachSession{
			projectRoot: projectRoot, chapter: chapter, messages: updated,
		}
		a.coach.mu.Unlock()
		a.session.invalidate()
		a.emitCoachDone(chapter, toCoachDTOs(workflows.TurnsFromMessages(updated)))
	}(root, message, prev)

	return nil
}

// GetChapterCoachTurns 返回当前章节的改稿对话（若已开启）。
func (a *App) GetChapterCoachTurns(chapter int) ([]CoachTurnDTO, error) {
	sess, _, err := a.getCoachSession(chapter)
	if err != nil {
		return nil, nil
	}
	return toCoachDTOs(workflows.TurnsFromMessages(sess.messages)), nil
}

// ClearChapterCoach 清除改稿讨论会话。
func (a *App) ClearChapterCoach(chapter int) {
	root, err := a.activeProjectRoot()
	if err != nil {
		return
	}
	a.coach.mu.Lock()
	delete(a.coach.sessions, coachKey(root, chapter))
	a.coach.mu.Unlock()
}

// StartChapterRevision 根据讨论记录异步生成修改稿（流式）。
func (a *App) StartChapterRevision(chapter int) (ReviseJobInfo, error) {
	if chapter <= 0 {
		return ReviseJobInfo{}, fmt.Errorf("无效章号")
	}
	cfg, err := config.Load()
	if err != nil {
		return ReviseJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return ReviseJobInfo{}, err
	}
	sess, _, err := a.getCoachSession(chapter)
	if err != nil {
		return ReviseJobInfo{}, fmt.Errorf("请先发送至少一条讨论消息")
	}

	a.revise.mu.Lock()
	for _, j := range a.revise.jobs {
		if j.info.Status == "running" || j.info.Status == "pending" {
			a.revise.mu.Unlock()
			return ReviseJobInfo{}, fmt.Errorf("已有改稿任务进行中（第 %d 章）", j.info.Chapter)
		}
	}

	id := fmt.Sprintf("revise-%d-%d", chapter, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	job := &reviseJob{
		info: ReviseJobInfo{ID: id, Chapter: chapter, Status: "pending"},
		cancel: cancel,
	}
	a.revise.jobs[id] = job
	a.revise.mu.Unlock()

	a.emitReviseStatus(id, chapter, "pending", "")

	msgs := append([]openai.ChatCompletionMessage(nil), sess.messages...)
	projectRoot := sess.projectRoot

	a.session.invalidate()

	go func() {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failReviseJob(id, chapter, err.Error())
			return
		}
		defer actx.Close()

		a.emitReviseStatus(id, chapter, "running", "")
		a.revise.mu.Lock()
		if j, ok := a.revise.jobs[id]; ok {
			j.info.Status = "running"
		}
		a.revise.mu.Unlock()

		wf := workflows.NewCoachWorkflow(actx.Config, actx.Project, actx.Store)
		content, err := wf.ReviseChapter(ctx, actx.Project, chapter, msgs, func(delta string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			a.emitReviseDelta(id, chapter, delta)
			return nil
		})

		a.revise.mu.Lock()
		if j, ok := a.revise.jobs[id]; ok {
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
		a.revise.mu.Unlock()

		if err != nil {
			if ctx.Err() != nil {
				a.emitReviseStatus(id, chapter, "cancelled", "已取消")
				return
			}
			a.emitReviseError(id, chapter, err.Error())
			a.emitReviseStatus(id, chapter, "failed", err.Error())
			return
		}
		a.emitReviseDone(id, chapter, content)
		a.emitReviseStatus(id, chapter, "done", "")
		a.session.invalidate()
	}()

	return job.info, nil
}

// CancelChapterRevision 取消进行中的改稿生成。
func (a *App) CancelChapterRevision(jobID string) error {
	a.revise.mu.Lock()
	defer a.revise.mu.Unlock()
	job, ok := a.revise.jobs[jobID]
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

// ApplyChapterContent 将修改稿写回正文目录并重建索引。
func (a *App) ApplyChapterContent(chapter int, content string) error {
	content = trimCoachInput(content)
	if chapter <= 0 {
		return fmt.Errorf("无效章号")
	}
	if content == "" {
		return fmt.Errorf("正文不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	root := reg.ActivePath()
	if root == "" {
		return errNoActiveProject
	}
	return a.session.withActive(root, func(actx *app.Context) error {
		path, err := workflows.ApplyCoachRevision(actx.Project, actx.Store, chapter, content)
		if err != nil {
			return err
		}
		_ = path
		return nil
	})
}

func trimCoachInput(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\n' && last != '\t' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

func (a *App) failReviseJob(id string, chapter int, errMsg string) {
	a.revise.mu.Lock()
	if j, ok := a.revise.jobs[id]; ok {
		j.info.Status = "failed"
	}
	a.revise.mu.Unlock()
	a.emitReviseError(id, chapter, errMsg)
	a.emitReviseStatus(id, chapter, "failed", errMsg)
}

func (a *App) emitCoachStream(chapter int, phase, delta string) {
	runtime.EventsEmit(a.ctx, eventCoachStream, map[string]any{
		"chapter": chapter, "phase": phase, "delta": delta,
	})
}

func (a *App) emitCoachDone(chapter int, turns []CoachTurnDTO) {
	raw, _ := json.Marshal(turns)
	runtime.EventsEmit(a.ctx, eventCoachDone, map[string]any{
		"chapter": chapter, "turns": string(raw),
	})
}

func (a *App) emitCoachError(chapter int, errMsg string) {
	runtime.EventsEmit(a.ctx, eventCoachError, map[string]any{
		"chapter": chapter, "error": errMsg,
	})
}

func (a *App) emitReviseDelta(jobID string, chapter int, delta string) {
	runtime.EventsEmit(a.ctx, eventReviseDelta, map[string]any{
		"job_id": jobID, "chapter": chapter, "delta": delta,
	})
}

func (a *App) emitReviseStatus(jobID string, chapter int, status, message string) {
	runtime.EventsEmit(a.ctx, eventReviseStatus, map[string]any{
		"job_id": jobID, "chapter": chapter, "status": status, "message": message,
	})
}

func (a *App) emitReviseError(jobID string, chapter int, errMsg string) {
	runtime.EventsEmit(a.ctx, eventReviseError, map[string]any{
		"job_id": jobID, "chapter": chapter, "error": errMsg,
	})
}

func (a *App) emitReviseDone(jobID string, chapter int, content string) {
	runtime.EventsEmit(a.ctx, eventReviseDone, map[string]any{
		"job_id": jobID, "chapter": chapter, "content": content,
	})
}
