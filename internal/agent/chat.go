package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// ChatChunkHandler 流式对话回调；phase 为 thinking 或 content。
type ChatChunkHandler func(phase string, delta string) error

// Chat 多轮对话，返回助手回复并更新 messages 历史。
func (a *Agent) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (reply string, updated []openai.ChatCompletionMessage, err error) {
	return a.ChatStream(ctx, messages, nil)
}

// ChatStream 流式多轮对话；handler 收到 thinking/content 分段 delta。
func (a *Agent) ChatStream(ctx context.Context, messages []openai.ChatCompletionMessage, handler ChatChunkHandler) (reply string, updated []openai.ChatCompletionMessage, err error) {
	stream, err := a.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    a.model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		return "", messages, fmt.Errorf("openai: %w", err)
	}
	defer stream.Close()

	parser := newThinkingStreamParser()
	var raw strings.Builder

	emit := func(phase, delta string) error {
		if handler != nil {
			return handler(phase, delta)
		}
		return nil
	}

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", messages, fmt.Errorf("openai stream: %w", err)
		}
		if len(resp.Choices) == 0 {
			continue
		}
		delta := resp.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		raw.WriteString(delta)
		if err := parser.Feed(delta,
			func(d string) error { return emit("thinking", d) },
			func(d string) error { return emit("content", d) },
		); err != nil {
			return "", messages, err
		}
	}
	if err := parser.Flush(
		func(d string) error { return emit("thinking", d) },
		func(d string) error { return emit("content", d) },
	); err != nil {
		return "", messages, err
	}

	reply = parser.Content()
	if reply == "" {
		reply = strings.TrimSpace(raw.String())
	}
	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleAssistant, Content: reply,
	})
	return reply, messages, nil
}
