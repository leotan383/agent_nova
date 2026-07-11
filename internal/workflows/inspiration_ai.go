package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/inspiration"
	"github.com/tanlian/agent_nova/internal/prompts"
	openai "github.com/sashabaranov/go-openai"
)

// InspirationEnrichResult AI 丰富后的灵感字段。
type InspirationEnrichResult struct {
	Title       string   `json:"title"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Spark       string   `json:"spark"`
	Synopsis    string   `json:"synopsis"`
	Protagonist string   `json:"protagonist"`
	Cheat       string   `json:"cheat"`
	Tags        []string `json:"tags"`
}

// InspirationWorkflow 灵感 AI 丰富。
type InspirationWorkflow struct {
	Agent *agent.Agent
}

func NewInspirationWorkflow(cfg *config.Config) *InspirationWorkflow {
	return &InspirationWorkflow{Agent: agent.New(agent.Options{Config: cfg})}
}

// InspirationDiscussInitialMessages 创建灵感探讨初始消息。
func InspirationDiscussInitialMessages(insp inspiration.Inspiration) []openai.ChatCompletionMessage {
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: prompts.InspirationDiscussSystem()},
	}
	seed := strings.TrimSpace(inspiration.DiscoverSeedPrompt(insp))
	if seed == "" {
		seed = "我想完善一个小说灵感，但目前只有零散想法，请帮我一起梳理世界观和核心设定。"
	}
	msgs = append(msgs, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser, Content: seed,
	})
	return msgs
}

// InspirationDiscussChat 灵感探讨对话（流式）。
func (w *InspirationWorkflow) InspirationDiscussChat(
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

// EnrichInspiration 一次性 AI 润色/扩写灵感。
func EnrichInspiration(ctx context.Context, cfg *config.Config, insp inspiration.Inspiration) (InspirationEnrichResult, error) {
	ag := agent.New(agent.Options{Config: cfg})
	userPrompt := buildInspirationEnrichPrompt(insp)
	raw, err := ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.InspirationEnrichSystem(),
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return InspirationEnrichResult{}, err
	}
	return parseInspirationEnrichJSON(raw)
}

// ExtractInspirationFromMessages 从探讨对话提炼设定。
func ExtractInspirationFromMessages(ctx context.Context, cfg *config.Config, messages []openai.ChatCompletionMessage) (InspirationEnrichResult, string, error) {
	ag := agent.New(agent.Options{Config: cfg})
	transcript := inspirationDiscussTranscript(messages)
	raw, err := ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.InspirationExtractSystem(),
		UserPrompt:   "以下是对话记录：\n\n" + transcript,
	})
	if err != nil {
		return InspirationEnrichResult{}, transcript, err
	}
	result, err := parseInspirationEnrichJSON(raw)
	return result, transcript, err
}

func buildInspirationEnrichPrompt(insp inspiration.Inspiration) string {
	var b strings.Builder
	b.WriteString("请基于以下灵感草稿，扩写为完整世界观与故事设定：\n\n")
	if t := strings.TrimSpace(insp.Title); t != "" {
		fmt.Fprintf(&b, "【标题】%s\n", t)
	}
	if g := strings.TrimSpace(insp.Genre); g != "" {
		fmt.Fprintf(&b, "【题材】%s\n", g)
	}
	if s := strings.TrimSpace(insp.Style); s != "" {
		fmt.Fprintf(&b, "【风格】%s\n", s)
	}
	if spark := strings.TrimSpace(insp.Spark); spark != "" {
		fmt.Fprintf(&b, "【核心想法】\n%s\n", spark)
	}
	if syn := strings.TrimSpace(insp.Synopsis); syn != "" {
		fmt.Fprintf(&b, "【简介】%s\n", syn)
	}
	if p := strings.TrimSpace(insp.Protagonist); p != "" {
		fmt.Fprintf(&b, "【主角】%s\n", p)
	}
	if c := strings.TrimSpace(insp.Cheat); c != "" {
		fmt.Fprintf(&b, "【金手指】%s\n", c)
	}
	if len(insp.Tags) > 0 {
		fmt.Fprintf(&b, "【标签】%s\n", strings.Join(insp.Tags, "、"))
	}
	return strings.TrimSpace(b.String())
}

func parseInspirationEnrichJSON(raw string) (InspirationEnrichResult, error) {
	jsonRaw, err := agent.ExtractJSONBlock(raw)
	if err != nil {
		return InspirationEnrichResult{}, fmt.Errorf("解析 AI 结果失败: %w", err)
	}
	var result InspirationEnrichResult
	if err := json.Unmarshal([]byte(jsonRaw), &result); err != nil {
		return InspirationEnrichResult{}, fmt.Errorf("解析设定 JSON 失败: %w", err)
	}
	result.Title = strings.TrimSpace(result.Title)
	result.Genre = strings.TrimSpace(result.Genre)
	result.Style = strings.TrimSpace(result.Style)
	result.Spark = strings.TrimSpace(result.Spark)
	result.Synopsis = strings.TrimSpace(result.Synopsis)
	result.Protagonist = strings.TrimSpace(result.Protagonist)
	result.Cheat = strings.TrimSpace(result.Cheat)
	if result.Genre == "" {
		result.Genre = "玄幻"
	}
	if result.Spark == "" {
		return InspirationEnrichResult{}, fmt.Errorf("AI 未生成有效设定内容")
	}
	return result, nil
}

func inspirationDiscussTranscript(messages []openai.ChatCompletionMessage) string {
	var b strings.Builder
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleSystem {
			continue
		}
		role := "作者"
		if m.Role == openai.ChatMessageRoleAssistant {
			role = "顾问"
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		fmt.Fprintf(&b, "%s> %s\n\n", role, content)
	}
	return strings.TrimSpace(b.String())
}

// TurnsFromInspirationDiscussMessages 转为前端展示轮次。
func TurnsFromInspirationDiscussMessages(messages []openai.ChatCompletionMessage) []CoachTurn {
	return TurnsFromDiscoverMessages(messages)
}
