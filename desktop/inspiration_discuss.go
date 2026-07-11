package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/workflows"
	openai "github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventInspirationStream = "inspiration:stream"
	eventInspirationDone   = "inspiration:done"
	eventInspirationError  = "inspiration:error"
)

// InspirationEnrichPreviewDTO AI 丰富结果预览。
type InspirationEnrichPreviewDTO struct {
	Title       string   `json:"title"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Spark       string   `json:"spark"`
	Synopsis    string   `json:"synopsis"`
	Protagonist string   `json:"protagonist"`
	Cheat       string   `json:"cheat"`
	Tags        []string `json:"tags"`
	Transcript  string   `json:"transcript,omitempty"`
}

// ApplyInspirationEnrichInput 应用 AI 丰富结果。
type ApplyInspirationEnrichInput struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Spark       string   `json:"spark"`
	Synopsis    string   `json:"synopsis"`
	Protagonist string   `json:"protagonist"`
	Cheat       string   `json:"cheat"`
	Tags        []string `json:"tags"`
}

type inspirationDiscussSession struct {
	inspirationID string
	messages      []openai.ChatCompletionMessage
	preview       *InspirationEnrichPreviewDTO
}

type inspirationDiscussManager struct {
	app  *App
	mu   sync.Mutex
	sess *inspirationDiscussSession
	busy bool
}

func newInspirationDiscussManager(app *App) *inspirationDiscussManager {
	return &inspirationDiscussManager{app: app}
}

func enrichPreviewFromResult(r workflows.InspirationEnrichResult, transcript string) InspirationEnrichPreviewDTO {
	return InspirationEnrichPreviewDTO{
		Title: r.Title, Genre: r.Genre, Style: r.Style, Spark: r.Spark,
		Synopsis: r.Synopsis, Protagonist: r.Protagonist, Cheat: r.Cheat,
		Tags: r.Tags, Transcript: transcript,
	}
}

// EnrichInspirationWithAI 一次性 AI 润色/扩写灵感。
func (a *App) EnrichInspirationWithAI(id string) (InspirationEnrichPreviewDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	store, err := loadInspirationStore()
	if err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	insp, err := store.Get(id)
	if err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	result, err := workflows.EnrichInspiration(context.Background(), cfg, insp)
	if err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	return enrichPreviewFromResult(result, ""), nil
}

// StartInspirationDiscuss 开始灵感 AI 探讨。
func (a *App) StartInspirationDiscuss(inspirationID string) ([]CoachTurnDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return nil, err
	}
	store, err := loadInspirationStore()
	if err != nil {
		return nil, err
	}
	insp, err := store.Get(inspirationID)
	if err != nil {
		return nil, err
	}

	a.inspirationDiscuss.mu.Lock()
	if a.inspirationDiscuss.busy {
		a.inspirationDiscuss.mu.Unlock()
		return nil, fmt.Errorf("探讨进行中，请稍候")
	}
	a.inspirationDiscuss.busy = true
	a.inspirationDiscuss.mu.Unlock()
	defer func() {
		a.inspirationDiscuss.mu.Lock()
		a.inspirationDiscuss.busy = false
		a.inspirationDiscuss.mu.Unlock()
	}()

	msgs := workflows.InspirationDiscussInitialMessages(insp)
	wf := workflows.NewInspirationWorkflow(cfg)
	if len(msgs) > 1 {
		ctx := context.Background()
		_, updated, err := wf.InspirationDiscussChat(ctx, msgs[:1], msgs[1].Content, func(phase, delta string) error {
			a.emitInspirationStream(phase, delta)
			return nil
		})
		if err != nil {
			a.emitInspirationError(err.Error())
			return nil, err
		}
		msgs = updated
	}

	a.inspirationDiscuss.mu.Lock()
	a.inspirationDiscuss.sess = &inspirationDiscussSession{inspirationID: inspirationID, messages: msgs}
	a.inspirationDiscuss.mu.Unlock()

	turns := toCoachDTOs(workflows.TurnsFromInspirationDiscussMessages(msgs))
	a.emitInspirationDone(turns)
	return turns, nil
}

// SendInspirationDiscussMessage 发送灵感探讨消息。
func (a *App) SendInspirationDiscussMessage(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("请输入消息")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return err
	}

	a.inspirationDiscuss.mu.Lock()
	sess := a.inspirationDiscuss.sess
	if sess == nil {
		a.inspirationDiscuss.mu.Unlock()
		return fmt.Errorf("请先开始 AI 探讨")
	}
	if a.inspirationDiscuss.busy {
		a.inspirationDiscuss.mu.Unlock()
		return fmt.Errorf("顾问正在回复，请稍候")
	}
	a.inspirationDiscuss.busy = true
	msgs := append([]openai.ChatCompletionMessage(nil), sess.messages...)
	a.inspirationDiscuss.mu.Unlock()

	go func() {
		defer func() {
			a.inspirationDiscuss.mu.Lock()
			a.inspirationDiscuss.busy = false
			a.inspirationDiscuss.mu.Unlock()
		}()
		wf := workflows.NewInspirationWorkflow(cfg)
		ctx := context.Background()
		_, updated, err := wf.InspirationDiscussChat(ctx, msgs, message, func(phase, delta string) error {
			a.emitInspirationStream(phase, delta)
			return nil
		})
		if err != nil {
			a.emitInspirationError(err.Error())
			return
		}
		a.inspirationDiscuss.mu.Lock()
		if a.inspirationDiscuss.sess != nil {
			a.inspirationDiscuss.sess.messages = updated
			a.inspirationDiscuss.sess.preview = nil
		}
		a.inspirationDiscuss.mu.Unlock()
		a.emitInspirationDone(toCoachDTOs(workflows.TurnsFromInspirationDiscussMessages(updated)))
	}()
	return nil
}

// FinishInspirationDiscuss 结束探讨并提炼设定。
func (a *App) FinishInspirationDiscuss() (InspirationEnrichPreviewDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}

	a.inspirationDiscuss.mu.Lock()
	sess := a.inspirationDiscuss.sess
	if sess == nil || len(sess.messages) < 2 {
		a.inspirationDiscuss.mu.Unlock()
		return InspirationEnrichPreviewDTO{}, fmt.Errorf("请先与顾问探讨几轮")
	}
	msgs := append([]openai.ChatCompletionMessage(nil), sess.messages...)
	a.inspirationDiscuss.mu.Unlock()

	result, transcript, err := workflows.ExtractInspirationFromMessages(context.Background(), cfg, msgs)
	if err != nil {
		return InspirationEnrichPreviewDTO{}, err
	}
	preview := enrichPreviewFromResult(result, transcript)
	a.inspirationDiscuss.mu.Lock()
	if a.inspirationDiscuss.sess != nil {
		a.inspirationDiscuss.sess.preview = &preview
	}
	a.inspirationDiscuss.mu.Unlock()
	return preview, nil
}

// ApplyInspirationEnrich 将 AI 丰富结果写回灵感。
func (a *App) ApplyInspirationEnrich(in ApplyInspirationEnrichInput) (InspirationDTO, error) {
	if strings.TrimSpace(in.ID) == "" {
		return InspirationDTO{}, fmt.Errorf("灵感 ID 不能为空")
	}
	if strings.TrimSpace(in.Spark) == "" {
		return InspirationDTO{}, fmt.Errorf("设定内容不能为空")
	}
	dto, err := a.UpdateInspiration(UpdateInspirationInput{
		ID: in.ID, Title: in.Title, Spark: in.Spark, Genre: in.Genre, Style: in.Style,
		Synopsis: in.Synopsis, Protagonist: in.Protagonist, Cheat: in.Cheat, Tags: in.Tags,
	})
	if err != nil {
		return InspirationDTO{}, err
	}
	a.ClearInspirationDiscuss()
	return dto, nil
}

// GetInspirationDiscussTurns 当前探讨轮次。
func (a *App) GetInspirationDiscussTurns() []CoachTurnDTO {
	a.inspirationDiscuss.mu.Lock()
	defer a.inspirationDiscuss.mu.Unlock()
	if a.inspirationDiscuss.sess == nil {
		return nil
	}
	return toCoachDTOs(workflows.TurnsFromInspirationDiscussMessages(a.inspirationDiscuss.sess.messages))
}

// ClearInspirationDiscuss 清除探讨会话。
func (a *App) ClearInspirationDiscuss() {
	a.inspirationDiscuss.mu.Lock()
	a.inspirationDiscuss.sess = nil
	a.inspirationDiscuss.mu.Unlock()
}

func (a *App) emitInspirationStream(phase, delta string) {
	runtime.EventsEmit(a.ctx, eventInspirationStream, map[string]interface{}{
		"phase": phase, "delta": delta,
	})
}

func (a *App) emitInspirationDone(turns []CoachTurnDTO) {
	raw, _ := json.Marshal(turns)
	runtime.EventsEmit(a.ctx, eventInspirationDone, map[string]interface{}{
		"turns": string(raw),
	})
}

func (a *App) emitInspirationError(errMsg string) {
	runtime.EventsEmit(a.ctx, eventInspirationError, map[string]interface{}{
		"error": errMsg,
	})
}
