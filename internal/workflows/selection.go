package workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/tools"
)

// SelectionWorkflow 章节片段改写。
type SelectionWorkflow struct {
	Agent *agent.Agent
}

func NewSelectionWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *SelectionWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &SelectionWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func surroundingContext(body, selected string, radius int) string {
	idx := strings.Index(body, selected)
	if idx < 0 {
		return truncateCoachRunes(body, radius*2)
	}
	start := idx - radius
	if start < 0 {
		start = 0
	}
	end := idx + len(selected) + radius
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}

// TransformSelection 对选中片段执行快捷改写。
func (w *SelectionWorkflow) TransformSelection(
	ctx context.Context,
	p *project.Project,
	chapter int,
	action, selected, customPrompt string,
	onDelta func(string) error,
) (string, error) {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		return "", fmt.Errorf("选中内容不能为空")
	}
	body, _, err := LoadChapterBundle(p, chapter)
	if err != nil {
		return "", err
	}
	ctxSnippet := surroundingContext(body, selected, 400)
	userPrompt := fmt.Sprintf(`【片段前后文（供语气与衔接参考）】
%s

【待处理片段】
%s`, ctxSnippet, selected)
	if action == "custom" && strings.TrimSpace(customPrompt) != "" {
		userPrompt += fmt.Sprintf("\n\n【作者要求】\n%s", strings.TrimSpace(customPrompt))
	}
	userPrompt += "\n\n请只输出修改后的片段正文，不要解释，不要用代码块或引号包裹。"

	out, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.SelectionTransformSystem(prompts.BookContext{
			Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
			Protagonist: p.Meta.Protagonist, Cheat: p.Meta.Cheat, Synopsis: p.Meta.Synopsis,
			Chapter: chapter, Volume: p.Meta.CurrentVolume,
		}, action),
		UserPrompt:   userPrompt,
		Stream:       onDelta != nil,
		OnDelta:      onDelta,
	})
	if err != nil {
		return "", err
	}
	return cleanSelectionOutput(out), nil
}

func cleanSelectionOutput(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s[3:], "\n"); i >= 0 {
			s = s[3+i+1:]
		}
		if j := strings.LastIndex(s, "```"); j >= 0 {
			s = s[:j]
		}
	}
	return strings.TrimSpace(s)
}
