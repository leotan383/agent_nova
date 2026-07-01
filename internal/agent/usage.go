package agent

import (
	"unicode/utf8"

	openai "github.com/sashabaranov/go-openai"
)

// UsageStats 单次 LLM 调用的 token 用量。
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func (u UsageStats) Total() int {
	return u.PromptTokens + u.CompletionTokens
}

// UsageAccumulator 累计多次调用的 token 用量。
type UsageAccumulator struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func (a *UsageAccumulator) Add(u UsageStats) {
	if a == nil {
		return
	}
	a.PromptTokens += u.PromptTokens
	a.CompletionTokens += u.CompletionTokens
}

func (a *UsageAccumulator) Total() int {
	if a == nil {
		return 0
	}
	return a.PromptTokens + a.CompletionTokens
}

func (a *UsageAccumulator) Snapshot() UsageStats {
	if a == nil {
		return UsageStats{}
	}
	return UsageStats{PromptTokens: a.PromptTokens, CompletionTokens: a.CompletionTokens}
}

// EstimateTokens 按字符数粗估 token（中文为主，Phase 0 够用）。
func EstimateTokens(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	return int(float64(n) * 1.2)
}

func messagesPromptTokens(messages []openai.ChatCompletionMessage) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content)
	}
	return total
}
