package workflows

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	openai "github.com/sashabaranov/go-openai"
)

// DiscoverResult 从探讨对话中提炼的立项信息。
type DiscoverResult struct {
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Style       string `json:"style"`
	Protagonist string `json:"protagonist"`
	Cheat       string `json:"cheat"`
	Pitch       string `json:"pitch"`
	Synopsis    string `json:"synopsis"`
}

// RunDiscoverySession 与 AI 多轮探讨小说方向，结束时提炼 InitInput。
// 返回 transcript 供存档。
func RunDiscoverySession(ctx context.Context, cfg *config.Config, seedGenre string) (project.InitInput, DiscoverResult, string, error) {
	ag := agent.New(agent.Options{Config: cfg})
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: prompts.DiscoverSystem()},
	}
	if seedGenre != "" && seedGenre != "玄幻" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("我初步想写%s题材，但细节都还没想好。", seedGenre),
		})
		reply, updated, err := ag.Chat(ctx, messages)
		if err != nil {
			return project.InitInput{}, DiscoverResult{}, "", err
		}
		messages = updated
		fmt.Println(reply)
		fmt.Println()
	}

	printDiscoverHelp()
	fmt.Println("先随便聊聊你想写什么样的故事；聊够了输入 /done 生成立项。")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("你> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return project.InitInput{}, DiscoverResult{}, "", err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "/") {
			cmd := strings.TrimPrefix(line, "/")
			switch strings.Fields(cmd)[0] {
			case "help", "h", "?":
				printDiscoverHelp()
				continue
			case "quit", "q", "exit":
				return project.InitInput{}, DiscoverResult{}, "", fmt.Errorf("已取消探讨")
			case "done":
				return extractDiscoverResult(ctx, ag, messages)
			default:
				fmt.Println("未知命令，输入 /help 查看")
				continue
			}
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser, Content: line,
		})
		reply, updated, err := ag.Chat(ctx, messages)
		if err != nil {
			return project.InitInput{}, DiscoverResult{}, "", err
		}
		messages = updated
		fmt.Println()
		fmt.Println("顾问>")
		fmt.Println(reply)
		fmt.Println()
	}
}

func printDiscoverHelp() {
	fmt.Println("命令：")
	fmt.Println("  /done   结束探讨，AI 提炼书名/题材/主角等并创建设定")
	fmt.Println("  /help   显示帮助")
	fmt.Println("  /quit   退出，不创建项目")
}

func extractDiscoverResult(ctx context.Context, ag *agent.Agent, messages []openai.ChatCompletionMessage) (project.InitInput, DiscoverResult, string, error) {
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
	fmt.Println()
	fmt.Println("--- 立项摘要 ---")
	fmt.Printf("书名：%s\n题材：%s\n风格：%s\n主角：%s\n金手指：%s\n", in.Title, in.Genre, in.Style, in.Protagonist, in.Cheat)
	if dr.Pitch != "" {
		fmt.Printf("梗概：%s\n", dr.Pitch)
	}
	fmt.Println("----------------")
	fmt.Print("确认以上信息创建设定？[Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	if confirm == "n" || confirm == "no" {
		return project.InitInput{}, DiscoverResult{}, transcript, fmt.Errorf("已取消立项")
	}
	return in, dr, transcript, nil
}

func formatTranscript(messages []openai.ChatCompletionMessage) string {
	var b strings.Builder
	for _, m := range messages {
		if m.Role == openai.ChatMessageRoleSystem {
			continue
		}
		label := "作者"
		if m.Role == openai.ChatMessageRoleAssistant {
			label = "顾问"
		}
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		b.WriteString(label)
		b.WriteString(": ")
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// SaveDiscoveryNotes 把探讨结论写入项目，供后续 plan/write 参考。
func SaveDiscoveryNotes(p *project.Project, dr DiscoverResult, transcript string) error {
	dir := filepath.Join(p.NovaDir(), "discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	notes := fmt.Sprintf(`# 构思探讨

## 一句话梗概
%s

## 故事核心
%s

## 讨论记录

%s
`, dr.Pitch, dr.Synopsis, transcript)
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(notes), 0o644); err != nil {
		return err
	}
	// 若有梗概，预填总纲「一句话梗概」段
	if dr.Pitch != "" || dr.Synopsis != "" {
		masterPath := filepath.Join(p.OutlineDir(), "总纲.md")
		data, err := os.ReadFile(masterPath)
		if err == nil {
			body := string(data)
			if dr.Pitch != "" {
				body = replaceSection(body, "一句话梗概", dr.Pitch)
			}
			if dr.Synopsis != "" {
				body = replaceSection(body, "核心冲突", dr.Synopsis)
			}
			_ = os.WriteFile(masterPath, []byte(body), 0o644)
		}
	}
	return nil
}

func replaceSection(md, heading, content string) string {
	marker := "## " + heading
	idx := strings.Index(md, marker)
	if idx < 0 {
		return md
	}
	rest := md[idx+len(marker):]
	next := strings.Index(rest, "\n## ")
	var tail string
	if next >= 0 {
		tail = rest[next:]
	} else {
		tail = ""
	}
	return md[:idx] + marker + "\n\n" + content + "\n" + tail
}
