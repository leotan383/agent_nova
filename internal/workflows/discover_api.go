package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	openai "github.com/sashabaranov/go-openai"
)

// DiscoverWorkflow 桌面端 AI 立项探讨。
type DiscoverWorkflow struct {
	Agent *agent.Agent
}

func NewDiscoverWorkflow(cfg *config.Config) *DiscoverWorkflow {
	return &DiscoverWorkflow{Agent: agent.New(agent.Options{Config: cfg})}
}

// DiscoverInitialMessages 创建探讨会话初始消息；可选 seedGenre / seedPrompt 触发首轮顾问回复。
func DiscoverInitialMessages(seedGenre, seedPrompt string) []openai.ChatCompletionMessage {
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: prompts.DiscoverSystem()},
	}
	seed := strings.TrimSpace(seedPrompt)
	if seed != "" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: seed,
		})
		return msgs
	}
	g := strings.TrimSpace(seedGenre)
	if g != "" && g != "玄幻" {
		msgs = append(msgs, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("我初步想写%s题材，但细节都还没想好。", g),
		})
	}
	return msgs
}

// DiscoverChat 发送用户消息并流式获取顾问回复。
func (w *DiscoverWorkflow) DiscoverChat(
	ctx context.Context,
	messages []openai.ChatCompletionMessage,
	userMsg string,
	onChunk agent.ChatChunkHandler,
) (string, []openai.ChatCompletionMessage, error) {
	if strings.TrimSpace(userMsg) != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser, Content: strings.TrimSpace(userMsg),
		})
	}
	return w.Agent.ChatStream(ctx, messages, onChunk)
}

// ExtractDiscoverFromMessages 从对话提炼立项信息。
func ExtractDiscoverFromMessages(ctx context.Context, cfg *config.Config, messages []openai.ChatCompletionMessage) (project.InitInput, DiscoverResult, string, error) {
	ag := agent.New(agent.Options{Config: cfg})
	transcript := formatTranscript(messages)
	raw, err := ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.DiscoverExtractSystem(),
		UserPrompt:   "以下是对话记录：\n\n" + transcript,
	})
	if err != nil {
		return project.InitInput{}, DiscoverResult{}, transcript, err
	}
	jsonRaw, err := agent.ExtractJSONBlock(raw)
	if err != nil {
		return project.InitInput{}, DiscoverResult{}, transcript, fmt.Errorf("提炼立项信息失败: %w", err)
	}
	var dr DiscoverResult
	if err := json.Unmarshal([]byte(jsonRaw), &dr); err != nil {
		return project.InitInput{}, DiscoverResult{}, transcript, fmt.Errorf("解析立项 JSON 失败: %w", err)
	}
	in := project.InitInput{
		Title:       strings.TrimSpace(dr.Title),
		Genre:       strings.TrimSpace(dr.Genre),
		Style:       strings.TrimSpace(dr.Style),
		Protagonist: strings.TrimSpace(dr.Protagonist),
		Cheat:       strings.TrimSpace(dr.Cheat),
		Synopsis:    strings.TrimSpace(dr.Synopsis),
	}
	if in.Title == "" {
		in.Title = "未命名"
	}
	if in.Genre == "" {
		in.Genre = "玄幻"
	}
	if in.Style == "" {
		in.Style = "热血"
	}
	return in, dr, transcript, nil
}

// TurnsFromDiscoverMessages 转为前端展示轮次。
func TurnsFromDiscoverMessages(messages []openai.ChatCompletionMessage) []CoachTurn {
	var out []CoachTurn
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleSystem {
			continue
		}
		role := "user"
		if m.Role == openai.ChatMessageRoleAssistant {
			role = "assistant"
		}
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		out = append(out, CoachTurn{Role: role, Content: m.Content})
	}
	return out
}
