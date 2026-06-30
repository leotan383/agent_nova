package workflows

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/tools"
	"github.com/tanlian/agent_nova/internal/version"
	openai "github.com/sashabaranov/go-openai"
)

const maxCoachContextRunes = 12000

// CoachTurn 改稿对话一轮。
type CoachTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CoachWorkflow struct {
	Agent *agent.Agent
}

func NewCoachWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *CoachWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &CoachWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func LoadChapterBundle(p *project.Project, chapter int) (body string, contextBlock string, err error) {
	path, _, err := p.FindChapterFile(chapter)
	if err != nil {
		return "", "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	body = string(data)
	var extras []string
	if data, err := os.ReadFile(p.SummaryPath(chapter)); err == nil && len(data) > 0 {
		extras = append(extras, "【本章摘要】\n"+string(data))
	}
	if data, err := os.ReadFile(p.ReviewPath(chapter)); err == nil && len(data) > 0 {
		extras = append(extras, "【审查报告】\n"+string(data))
	}
	contextBlock = strings.Join(extras, "\n\n")
	return body, contextBlock, nil
}

func truncateCoachRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n\n…（正文过长，已截断供讨论）"
}

func coachAnchor(p *project.Project, chapter int) prompts.BookContext {
	return prompts.BookContext{
		Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
		Protagonist: p.Meta.Protagonist, Cheat: p.Meta.Cheat, Synopsis: p.Meta.Synopsis,
		Chapter: chapter, Volume: p.Meta.CurrentVolume,
	}
}

func (w *CoachWorkflow) PrepareCoachMessages(p *project.Project, chapter int) ([]openai.ChatCompletionMessage, error) {
	body, ctxBlock, err := LoadChapterBundle(p, chapter)
	if err != nil {
		return nil, err
	}
	sys := prompts.ChapterCoachSystem(coachAnchor(p, chapter))
	contextMsg := fmt.Sprintf("以下是需要讨论的章节材料：\n\n%s\n\n---\n\n【正文】\n%s",
		ctxBlock, truncateCoachRunes(body, maxCoachContextRunes))
	if ctxBlock == "" {
		contextMsg = fmt.Sprintf("以下是需要讨论的章节正文：\n\n%s", truncateCoachRunes(body, maxCoachContextRunes))
	}
	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sys},
		{Role: openai.ChatMessageRoleUser, Content: contextMsg},
	}, nil
}

func (w *CoachWorkflow) InitMessages(p *project.Project, chapter int) ([]openai.ChatCompletionMessage, error) {
	return w.PrepareCoachMessages(p, chapter)
}

func (w *CoachWorkflow) InitMessagesStream(ctx context.Context, p *project.Project, chapter int, onChunk agent.ChatChunkHandler) ([]openai.ChatCompletionMessage, error) {
	messages, err := w.PrepareCoachMessages(p, chapter)
	if err != nil {
		return nil, err
	}
	_, updated, err := w.Agent.ChatStream(ctx, messages, onChunk)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (w *CoachWorkflow) Chat(ctx context.Context, messages []openai.ChatCompletionMessage, userMsg string) (string, []openai.ChatCompletionMessage, error) {
	return w.ChatStream(ctx, messages, userMsg, nil)
}

func (w *CoachWorkflow) ChatStream(ctx context.Context, messages []openai.ChatCompletionMessage, userMsg string, onChunk agent.ChatChunkHandler) (string, []openai.ChatCompletionMessage, error) {
	messages = append(messages, openai.ChatCompletionMessage{
		Role: openai.ChatMessageRoleUser, Content: userMsg,
	})
	return w.Agent.ChatStream(ctx, messages, onChunk)
}

func (w *CoachWorkflow) ReviseChapter(ctx context.Context, p *project.Project, chapter int, coachMessages []openai.ChatCompletionMessage, onDelta func(string) error) (string, error) {
	body, _, err := LoadChapterBundle(p, chapter)
	if err != nil {
		return "", err
	}
	transcript := formatCoachTranscript(coachMessages)
	userPrompt := fmt.Sprintf(`【讨论记录】
%s

【当前正文】
%s

请输出修改后的完整章节正文。`, transcript, truncateCoachRunes(body, maxCoachContextRunes))
	return w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.ChapterReviseSystem(coachAnchor(p, chapter)),
		UserPrompt:   userPrompt,
		Stream:       onDelta != nil,
		OnDelta:      onDelta,
	})
}

func formatCoachTranscript(messages []openai.ChatCompletionMessage) string {
	var b strings.Builder
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleSystem {
			continue
		}
		if m.Role == openai.ChatMessageRoleUser && strings.Contains(m.Content, "以下是需要讨论的") {
			continue
		}
		label := "作者"
		if m.Role == openai.ChatMessageRoleAssistant {
			label = "顾问"
		}
		b.WriteString(label)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}

func TurnsFromMessages(messages []openai.ChatCompletionMessage) []CoachTurn {
	var out []CoachTurn
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleSystem {
			continue
		}
		if m.Role == openai.ChatMessageRoleUser && strings.Contains(m.Content, "以下是需要讨论的") {
			continue
		}
		role := "user"
		if m.Role == openai.ChatMessageRoleAssistant {
			role = "assistant"
		}
		out = append(out, CoachTurn{Role: role, Content: m.Content})
	}
	return out
}

// CoachSessionOptions CLI 改稿讨论选项。
type CoachSessionOptions struct {
	StreamRevise bool
}

// ApplyCoachRevision 将修改稿写回正文并重建索引。
func ApplyCoachRevision(p *project.Project, st *store.Store, chapter int, content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("正文不能为空")
	}
	title := pipeline.ParseChapterTitle(content)
	if title == "" {
		if ch, err := st.GetChapter(chapter); err == nil {
			title = ch.Title
		}
	}
	path, err := pipeline.SaveChapterWithVersion(p, chapter, title, content, version.SourceCoachApply, "改稿应用")
	if err != nil {
		return "", err
	}
	if err := pipeline.PostWriteIndex(p, st, chapter, path); err != nil {
		return "", err
	}
	return path, nil
}

// RunCoachSession 与 AI 多轮讨论已写章节，支持生成并应用修改稿。
func RunCoachSession(ctx context.Context, cfg *config.Config, p *project.Project, st *store.Store, chapter int, opts CoachSessionOptions) error {
	wf := NewCoachWorkflow(cfg, p, st)

	printCoachHelp()
	fmt.Printf("正在讨论《%s》第 %d 章。输入消息开始，或 /revise 生成修改稿。\n\n", p.Meta.Title, chapter)

	var messages []openai.ChatCompletionMessage
	reader := bufio.NewReader(os.Stdin)
	var lastDraft string

	for {
		fmt.Print("你> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			cmdLine := strings.TrimSpace(strings.TrimPrefix(line, "/"))
			cmd := strings.Fields(cmdLine)[0]
			switch cmd {
			case "help", "h", "?":
				printCoachHelp()
			case "quit", "q", "exit":
				return nil
			case "revise", "r":
				if messages == nil || userTurnCount(messages) < 1 {
					fmt.Println("请先说说想改什么，再生成修改稿。")
					continue
				}
				fmt.Fprintln(os.Stderr, "正在生成修改稿…")
				draft, err := wf.ReviseChapter(ctx, p, chapter, messages, func(delta string) error {
					if opts.StreamRevise {
						_, werr := os.Stdout.WriteString(delta)
						return werr
					}
					return nil
				})
				if err != nil {
					return err
				}
				lastDraft = draft
				if opts.StreamRevise {
					fmt.Println()
				} else {
					fmt.Println()
					fmt.Println("--- 修改稿预览 ---")
					fmt.Println(draft)
					fmt.Println("----------------")
				}
				fmt.Println("满意可 /apply 写回正文，或 /save 保存到文件。")
			case "apply", "a":
				if lastDraft == "" {
					fmt.Println("请先 /revise 生成修改稿。")
					continue
				}
				fmt.Print("确认将修改稿写回正文？[Y/n]: ")
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm == "n" || confirm == "no" {
					fmt.Println("已取消。")
					continue
				}
				path, err := ApplyCoachRevision(p, st, chapter, lastDraft)
				if err != nil {
					return err
				}
				fmt.Printf("已写回: %s\n", path)
				return nil
			case "save", "s":
				if lastDraft == "" {
					fmt.Println("请先 /revise 生成修改稿。")
					continue
				}
				outPath := coachDraftPath(p, chapter)
				if len(strings.Fields(cmdLine)) > 1 {
					outPath = strings.TrimSpace(strings.TrimPrefix(cmdLine, cmd))
				}
				if err := os.WriteFile(outPath, []byte(lastDraft), 0o644); err != nil {
					return err
				}
				fmt.Printf("已保存: %s\n", outPath)
			default:
				fmt.Println("未知命令，输入 /help 查看。")
			}
			continue
		}

		if messages == nil {
			messages, err = wf.PrepareCoachMessages(p, chapter)
			if err != nil {
				return err
			}
		}
		reply, updated, err := wf.ChatStream(ctx, messages, line, cliCoachStreamHandler())
		if err != nil {
			return err
		}
		messages = updated
		_ = reply
		fmt.Println()
	}
}

type cliCoachPrinter struct {
	thinkingStarted bool
	contentStarted  bool
}

func cliCoachStreamHandler() agent.ChatChunkHandler {
	pr := &cliCoachPrinter{}
	return func(phase, delta string) error {
		switch phase {
		case "thinking":
			if !pr.thinkingStarted {
				fmt.Print("【思考】")
				pr.thinkingStarted = true
			}
			_, err := os.Stdout.WriteString(delta)
			return err
		case "content":
			if !pr.contentStarted {
				if pr.thinkingStarted {
					fmt.Println()
				}
				fmt.Println()
				fmt.Println("顾问>")
				pr.contentStarted = true
			}
			_, err := os.Stdout.WriteString(delta)
			return err
		default:
			return nil
		}
	}
}

func userTurnCount(messages []openai.ChatCompletionMessage) int {
	n := 0
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleUser && !strings.Contains(m.Content, "以下是需要讨论的") {
			n++
		}
	}
	return n
}

func printCoachTurns(turns []CoachTurn) {
	for _, t := range turns {
		if t.Role != "assistant" {
			continue
		}
		fmt.Println("顾问>")
		fmt.Println(t.Content)
		fmt.Println()
	}
}

func printCoachHelp() {
	fmt.Println("命令：")
	fmt.Println("  /revise  根据讨论生成修改稿（配合 --stream 流式输出）")
	fmt.Println("  /apply   确认后将修改稿写回正文/（自动备份）")
	fmt.Println("  /save [路径]  保存修改稿到文件（默认 .nova/coach/第N章-draft.md）")
	fmt.Println("  /help    显示帮助")
	fmt.Println("  /quit    退出，不写回")
}

func coachDraftPath(p *project.Project, chapter int) string {
	dir := filepath.Join(p.NovaDir(), "coach")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, fmt.Sprintf("第%03d章-draft.md", chapter))
}
