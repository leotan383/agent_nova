package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/library"
	"github.com/tanlian/agent_nova/internal/workflows"
	openai "github.com/sashabaranov/go-openai"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventDiscoverStream = "discover:stream"
	eventDiscoverDone   = "discover:done"
	eventDiscoverError  = "discover:error"
)

// DiscoverPreviewDTO 探讨结束后提炼的立项预览。
type DiscoverPreviewDTO struct {
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Style       string `json:"style"`
	Protagonist string `json:"protagonist"`
	Cheat       string `json:"cheat"`
	Pitch       string `json:"pitch"`
	Synopsis    string `json:"synopsis"`
	Transcript  string `json:"transcript"`
}

// CreateNovelFromDiscoverInput 从探讨结果创建新书。
type CreateNovelFromDiscoverInput struct {
	Dir          string `json:"dir"`
	Title        string `json:"title"`
	Genre        string `json:"genre"`
	Style        string `json:"style"`
	TargetWords  int    `json:"target_words"`
	ChapterWords int    `json:"chapter_words"`
	Protagonist  string `json:"protagonist"`
	Cheat        string `json:"cheat"`
	Synopsis     string `json:"synopsis"`
	Enrich         bool   `json:"enrich"`
	InspirationID  string `json:"inspiration_id"`
}

type discoverSession struct {
	genre    string
	messages []openai.ChatCompletionMessage
	preview  *DiscoverPreviewDTO
}

type discoverManager struct {
	app  *App
	mu   sync.Mutex
	sess *discoverSession
	busy bool
}

func newDiscoverManager(app *App) *discoverManager {
	return &discoverManager{app: app}
}

// StartDiscover 开始 AI 立项探讨（可选 seedGenre / seedPrompt）。
func (a *App) StartDiscover(seedGenre string, seedPrompt string) ([]CoachTurnDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return nil, err
	}

	a.discover.mu.Lock()
	if a.discover.busy {
		a.discover.mu.Unlock()
		return nil, fmt.Errorf("探讨进行中，请稍候")
	}
	a.discover.busy = true
	a.discover.mu.Unlock()
	defer func() {
		a.discover.mu.Lock()
		a.discover.busy = false
		a.discover.mu.Unlock()
	}()

	msgs := workflows.DiscoverInitialMessages(seedGenre, seedPrompt)
	wf := workflows.NewDiscoverWorkflow(cfg)

	if len(msgs) > 1 {
		ctx := context.Background()
		_, updated, err := wf.DiscoverChat(ctx, msgs[:1], msgs[1].Content, func(phase, delta string) error {
			a.emitDiscoverStream(phase, delta)
			return nil
		})
		if err != nil {
			a.emitDiscoverError(err.Error())
			return nil, err
		}
		msgs = updated
	}

	a.discover.mu.Lock()
	a.discover.sess = &discoverSession{genre: seedGenre, messages: msgs}
	a.discover.mu.Unlock()

	turns := toCoachDTOs(workflows.TurnsFromDiscoverMessages(msgs))
	a.emitDiscoverDone(turns)
	return turns, nil
}

// SendDiscoverMessage 发送探讨消息（流式）。
func (a *App) SendDiscoverMessage(message string) error {
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

	a.discover.mu.Lock()
	sess := a.discover.sess
	if sess == nil {
		a.discover.mu.Unlock()
		return fmt.Errorf("请先开始探讨")
	}
	if a.discover.busy {
		a.discover.mu.Unlock()
		return fmt.Errorf("顾问正在回复，请稍候")
	}
	a.discover.busy = true
	msgs := append([]openai.ChatCompletionMessage(nil), sess.messages...)
	a.discover.mu.Unlock()

	go func() {
		defer func() {
			a.discover.mu.Lock()
			a.discover.busy = false
			a.discover.mu.Unlock()
		}()
		wf := workflows.NewDiscoverWorkflow(cfg)
		ctx := context.Background()
		_, updated, err := wf.DiscoverChat(ctx, msgs, message, func(phase, delta string) error {
			a.emitDiscoverStream(phase, delta)
			return nil
		})
		if err != nil {
			a.emitDiscoverError(err.Error())
			return
		}
		a.discover.mu.Lock()
		if a.discover.sess != nil {
			a.discover.sess.messages = updated
			a.discover.sess.preview = nil
		}
		a.discover.mu.Unlock()
		a.emitDiscoverDone(toCoachDTOs(workflows.TurnsFromDiscoverMessages(updated)))
	}()
	return nil
}

// FinishDiscover 结束探讨并提炼立项信息。
func (a *App) FinishDiscover() (DiscoverPreviewDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return DiscoverPreviewDTO{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return DiscoverPreviewDTO{}, err
	}

	a.discover.mu.Lock()
	sess := a.discover.sess
	if sess == nil || len(sess.messages) < 2 {
		a.discover.mu.Unlock()
		return DiscoverPreviewDTO{}, fmt.Errorf("请先与顾问探讨几轮")
	}
	msgs := append([]openai.ChatCompletionMessage(nil), sess.messages...)
	a.discover.mu.Unlock()

	in, dr, transcript, err := workflows.ExtractDiscoverFromMessages(context.Background(), cfg, msgs)
	if err != nil {
		return DiscoverPreviewDTO{}, err
	}
	style := in.Style
	if style == "" {
		style = "热血"
	}
	preview := DiscoverPreviewDTO{
		Title: in.Title, Genre: in.Genre, Style: style,
		Protagonist: in.Protagonist, Cheat: in.Cheat,
		Pitch: dr.Pitch, Synopsis: pickSynopsis(in.Synopsis, dr.Synopsis),
		Transcript: transcript,
	}
	a.discover.mu.Lock()
	if a.discover.sess != nil {
		a.discover.sess.preview = &preview
	}
	a.discover.mu.Unlock()
	return preview, nil
}

// CreateNovelFromDiscover 根据探讨结果创建新书并加入书库。
func (a *App) CreateNovelFromDiscover(in CreateNovelFromDiscoverInput) (library.NovelCard, error) {
	if strings.TrimSpace(in.Dir) == "" {
		return library.NovelCard{}, fmt.Errorf("请指定保存目录")
	}
	if strings.TrimSpace(in.Title) == "" {
		return library.NovelCard{}, fmt.Errorf("书名不能为空")
	}

	a.discover.mu.Lock()
	var preview *DiscoverPreviewDTO
	if a.discover.sess != nil {
		preview = a.discover.sess.preview
	}
	a.discover.mu.Unlock()

	card, err := a.CreateNovel(CreateNovelInput{
		Dir: in.Dir, Title: in.Title, Genre: in.Genre, Style: in.Style,
		TargetWords: in.TargetWords, ChapterWords: in.ChapterWords,
		Protagonist: in.Protagonist, Cheat: in.Cheat, Synopsis: in.Synopsis,
		InspirationID: in.InspirationID,
	})
	if err != nil {
		return library.NovelCard{}, err
	}

	if preview != nil && preview.Transcript != "" {
		reg, err := a.loadRegistry()
		if err == nil {
			_ = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
				dr := workflows.DiscoverResult{
					Title: in.Title, Genre: in.Genre, Style: in.Style,
					Protagonist: in.Protagonist, Cheat: in.Cheat,
					Pitch: preview.Pitch, Synopsis: in.Synopsis,
				}
				return workflows.SaveDiscoveryNotes(actx.Project, dr, preview.Transcript)
			})
		}
	}

	if in.Enrich {
		cfg, err := config.Load()
		if err == nil && app.RequireAPIKey(cfg) == nil {
			reg, err := a.loadRegistry()
			if err == nil {
				_ = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
					wf := workflows.NewInitWorkflow(cfg, actx.Project.Root, actx.Store)
					_, _ = wf.EnrichSettings(context.Background(), actx.Project)
					return nil
				})
			}
		}
	}

	a.ClearDiscover()
	return card, nil
}

func pickSynopsis(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

// GetDiscoverTurns 当前探讨对话轮次。
func (a *App) GetDiscoverTurns() []CoachTurnDTO {
	a.discover.mu.Lock()
	defer a.discover.mu.Unlock()
	if a.discover.sess == nil {
		return nil
	}
	return toCoachDTOs(workflows.TurnsFromDiscoverMessages(a.discover.sess.messages))
}

// ClearDiscover 清除探讨会话。
func (a *App) ClearDiscover() {
	a.discover.mu.Lock()
	a.discover.sess = nil
	a.discover.mu.Unlock()
}

func (a *App) emitDiscoverStream(phase, delta string) {
	runtime.EventsEmit(a.ctx, eventDiscoverStream, map[string]interface{}{
		"phase": phase, "delta": delta,
	})
}

func (a *App) emitDiscoverDone(turns []CoachTurnDTO) {
	raw, _ := json.Marshal(turns)
	runtime.EventsEmit(a.ctx, eventDiscoverDone, map[string]interface{}{
		"turns": string(raw),
	})
}

func (a *App) emitDiscoverError(errMsg string) {
	runtime.EventsEmit(a.ctx, eventDiscoverError, map[string]interface{}{
		"error": errMsg,
	})
}
