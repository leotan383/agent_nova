package workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/tools"
	"github.com/tanlian/agent_nova/internal/version"
)

// BatchPolishOptions 批量润色参数。
type BatchPolishOptions struct {
	Chapters []int
	Rule     string // person | naming | tone
}

// BatchPolishChapterResult 单章润色结果。
type BatchPolishChapterResult struct {
	Chapter     int    `json:"chapter"`
	Title       string `json:"title"`
	Original    string `json:"original"`
	Polished    string `json:"polished"`
	Error       string `json:"error,omitempty"`
}

// BatchPolishReport 批量润色报告。
type BatchPolishReport struct {
	Rule     string                     `json:"rule"`
	Results  []BatchPolishChapterResult `json:"results"`
}

// BatchPolishWorkflow 批量润色。
type BatchPolishWorkflow struct {
	Agent *agent.Agent
}

func NewBatchPolishWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *BatchPolishWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &BatchPolishWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

// Run 对指定章节批量润色（不自动写回）。
func (w *BatchPolishWorkflow) Run(ctx context.Context, p *project.Project, st *store.Store, opts BatchPolishOptions, onProgress func(chapter int, total int)) (*BatchPolishReport, error) {
	if len(opts.Chapters) == 0 {
		return nil, fmt.Errorf("请选择至少一章")
	}
	rule := strings.TrimSpace(opts.Rule)
	if rule == "" {
		rule = "person"
	}
	ruleDesc := polishRuleDesc(rule)
	var results []BatchPolishChapterResult
	total := len(opts.Chapters)
	for i, ch := range opts.Chapters {
		if ctx.Err() != nil {
			return &BatchPolishReport{Rule: rule, Results: results}, ctx.Err()
		}
		if onProgress != nil {
			onProgress(ch, total)
		}
		body, _, err := LoadChapterBundle(p, ch)
		if err != nil {
			results = append(results, BatchPolishChapterResult{Chapter: ch, Error: err.Error()})
			continue
		}
		title := pipeline.ParseChapterTitle(body)
		if title == "" {
			if c, err := st.GetChapter(ch); err == nil {
				title = c.Title
			}
		}
		polished, err := w.Agent.Run(ctx, agent.RunInput{
			SystemPrompt: prompts.BatchPolishSystem(coachAnchor(p, ch), ruleDesc),
			UserPrompt: fmt.Sprintf(`请按规则润色以下章节正文，输出完整 Markdown 正文（含 # 标题），不要解释。

【规则】%s

【正文】
%s`, ruleDesc, truncateCoachRunes(body, maxCoachContextRunes)),
		})
		if err != nil {
			results = append(results, BatchPolishChapterResult{Chapter: ch, Title: title, Original: body, Error: err.Error()})
			continue
		}
		results = append(results, BatchPolishChapterResult{
			Chapter: ch, Title: title, Original: body, Polished: strings.TrimSpace(polished),
		})
		if ctx.Err() != nil {
			break
		}
		_ = i
	}
	return &BatchPolishReport{Rule: rule, Results: results}, nil
}

// ApplyPolishChapter 应用单章润色结果。
func ApplyPolishChapter(p *project.Project, st *store.Store, chapter int, content string) error {
	_, err := ApplyCoachRevision(p, st, chapter, content)
	return err
}

func polishRuleDesc(rule string) string {
	switch rule {
	case "naming":
		return "统一人物称谓与专有名词写法，不改变情节"
	case "tone":
		return "统一叙述语气与文风，不改变情节与人称"
	default:
		return "统一人称（第三人称限知或全书既定视角），不改变情节"
	}
}

// PreviewPolishDiff 对比润色前后。
func PreviewPolishDiff(chapter int, original, polished string) version.DiffResult {
	return version.DiffTexts("original", "polished", fmt.Sprintf("第%d章·原稿", chapter), fmt.Sprintf("第%d章·润色", chapter), original, polished)
}
