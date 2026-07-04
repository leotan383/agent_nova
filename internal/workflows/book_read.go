package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/tools"
)

// BookReadOptions 全书通读参数。
type BookReadOptions struct {
	FromChapter int
	ToChapter   int
	Focus       string // pace | repeat | foreshadow | all
}

// BookReadItem 通读报告条目。
type BookReadItem struct {
	Category   string `json:"category"`
	Severity   string `json:"severity"` // info | warn | critical
	Chapter    int    `json:"chapter"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt"`
	Suggestion string `json:"suggestion"`
}

// BookReadReport 全书通读报告。
type BookReadReport struct {
	Summary string         `json:"summary"`
	Items   []BookReadItem `json:"items"`
	Report  *report.Report `json:"-"`
}

type bookReadLLM struct {
	Items   []BookReadItem `json:"items"`
	Summary string         `json:"summary"`
}

// BookReadWorkflow 全书通读。
type BookReadWorkflow struct {
	Agent *agent.Agent
}

func NewBookReadWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *BookReadWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &BookReadWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

// Run 生成全书通读报告。
func (w *BookReadWorkflow) Run(ctx context.Context, p *project.Project, st *store.Store, opts BookReadOptions) (*BookReadReport, error) {
	written := p.Meta.CurrentChapter
	if written <= 0 {
		return nil, fmt.Errorf("尚无已写章节")
	}
	from := opts.FromChapter
	if from <= 0 {
		from = 1
	}
	to := opts.ToChapter
	if to <= 0 || to > written {
		to = written
	}
	if from > to {
		return nil, fmt.Errorf("无效章节范围")
	}
	focus := strings.TrimSpace(opts.Focus)
	if focus == "" {
		focus = "all"
	}

	summaries := collectSummariesRange(p, from, to, 16000)
	foreshadows := formatOpenForeshadows(st)
	userPrompt := fmt.Sprintf(`请通读第 %d–%d 章摘要链，输出结构化诊断报告（JSON）。

关注维度：%s
- pace：节奏是否拖沓/过密
- repeat：重复场景、重复描写、重复爽点模式
- foreshadow：未收伏笔、超期伏笔、矛盾伏笔
- all：以上全部

## 摘要链
%s

## 开放伏笔
%s

输出 JSON（不要 markdown 代码块）：
{
  "summary": "200字内总评",
  "items": [
    {
      "category": "pace|repeat|foreshadow",
      "severity": "info|warn|critical",
      "chapter": 12,
      "title": "短标题",
      "excerpt": "问题简述",
      "suggestion": "可执行建议"
    }
  ]
}
最多 20 条，按 severity 排序。`, from, to, focus, summaries, ifEmpty("(无)", foreshadows))

	raw, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.BookReadSystem(prompts.BookContext{
			Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
			Protagonist: p.Meta.Protagonist, Synopsis: p.Meta.Synopsis,
		}),
		UserPrompt: userPrompt,
	})
	if err != nil {
		return nil, err
	}
	rawJSON, err := agent.ExtractJSONBlock(raw)
	if err != nil {
		rawJSON = raw
	}
	var parsed bookReadLLM
	if err := json.Unmarshal([]byte(rawJSON), &parsed); err != nil {
		return &BookReadReport{
			Summary: "通读完成，但结果解析失败，请重试。",
			Items:   nil,
			Report: &report.Report{
				Stage: "全书通读", Status: report.StatusNeedsAction,
				Summary: truncateRunesBook(raw, 500),
			},
		}, nil
	}
	return &BookReadReport{
		Summary: parsed.Summary,
		Items:   parsed.Items,
		Report: &report.Report{
			Stage:   "全书通读",
			Status:  report.StatusDone,
			Summary: parsed.Summary,
		},
	}, nil
}

func collectSummariesRange(p *project.Project, from, to, maxRunes int) string {
	var blocks []string
	total := 0
	for i := from; i <= to; i++ {
		data, err := os.ReadFile(p.SummaryPath(i))
		if err != nil {
			continue
		}
		block := fmt.Sprintf("## 第%d章\n%s", i, strings.TrimSpace(string(data)))
		n := utf8.RuneCountInString(block)
		if total+n > maxRunes {
			blocks = append(blocks, fmt.Sprintf("> … 第 %d–%d 章摘要省略", i, to))
			break
		}
		blocks = append(blocks, block)
		total += n
	}
	if len(blocks) == 0 {
		return "(暂无摘要，建议先完成审查/提取)"
	}
	return strings.Join(blocks, "\n\n")
}

func truncateRunesBook(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
