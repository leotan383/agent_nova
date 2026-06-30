package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/logger"
	"github.com/tanlian/agent_nova/internal/tools"
	openai "github.com/sashabaranov/go-openai"
)

const maxToolLoops = 8

type Agent struct {
	client   *openai.Client
	model    string
	registry *tools.Registry
}

type Options struct {
	Config   *config.Config
	Registry *tools.Registry
}

func New(opts Options) *Agent {
	clientCfg := openai.DefaultConfig(opts.Config.OpenAIAPIKey)
	if opts.Config.OpenAIBaseURL != "" {
		clientCfg.BaseURL = opts.Config.OpenAIBaseURL
	}
	reg := opts.Registry
	if reg == nil {
		reg = tools.NewRegistry()
	}
	return &Agent{
		client:   openai.NewClientWithConfig(clientCfg),
		model:    opts.Config.Model,
		registry: reg,
	}
}

type RunInput struct {
	SystemPrompt string
	UserPrompt   string
	Tools        bool
	Stream       bool
	OnDelta      func(string) error
}

func (a *Agent) Run(ctx context.Context, in RunInput) (string, error) {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: in.SystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: in.UserPrompt},
	}
	toolDefs := a.registry.Tools()
	if !in.Tools {
		toolDefs = nil
	}
	if in.Stream && len(toolDefs) == 0 {
		return a.runStream(ctx, messages, in.OnDelta)
	}
	for i := 0; i < maxToolLoops; i++ {
		logger.Debug("agent round=%d messages=%d tools=%d", i+1, len(messages), len(toolDefs))
		resp, err := a.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    a.model,
			Messages: messages,
			Tools:    toolDefs,
		})
		if err != nil {
			return "", fmt.Errorf("openai: %w", err)
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("openai: empty choices")
		}
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			return msg.Content, nil
		}
		messages = append(messages, msg)
		for _, tc := range msg.ToolCalls {
			result, err := a.registry.Execute(tc.Function.Name, []byte(tc.Function.Arguments))
			if err != nil {
				result = fmt.Sprintf(`{"error":%q}`, err.Error())
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}
	return "", errors.New("tool loop exceeded maximum iterations")
}

func (a *Agent) runStream(ctx context.Context, messages []openai.ChatCompletionMessage, onDelta func(string) error) (string, error) {
	stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return "", err
	}
	defer stream.Close()
	var b strings.Builder
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			continue
		}
		delta := resp.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		b.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return "", err
			}
		}
	}
	return b.String(), nil
}

func ExtractJSONBlock(content string) (string, error) {
	start := strings.Index(content, "```json")
	if start >= 0 {
		content = content[start+7:]
		if end := strings.Index(content, "```"); end >= 0 {
			return strings.TrimSpace(content[:end]), nil
		}
	}
	start = strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		return content[start : end+1], nil
	}
	start = strings.Index(content, "[")
	end = strings.LastIndex(content, "]")
	if start >= 0 && end > start {
		return content[start : end+1], nil
	}
	return "", errors.New("no json block found")
}

func ParseJSONArray[T any](content string) ([]T, error) {
	raw, err := ExtractJSONBlock(content)
	if err != nil {
		return nil, err
	}
	var out []T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}
