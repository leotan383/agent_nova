package workflows

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/store"
)

// ReplanOptions 动态卷纲 Replan 参数。
type ReplanOptions struct {
	Volume      int
	FromChapter int    // 0 = 自动（CurrentChapter+1）
	Notes       string // 作者补充说明
}

// ReplanResult Replan 产出（不直接写文件，由调用方 diff 确认后应用）。
type ReplanResult struct {
	ProposedContent string
	OldContent      string
	FromChapter     int
	WrittenThrough  int
	Report          *report.Report
}

// ReplanVolume 基于已写内容重新规划指定卷纲。
func (w *PlanWorkflow) ReplanVolume(ctx context.Context, p *project.Project, st *store.Store, opts ReplanOptions) (*ReplanResult, error) {
	if opts.Volume <= 0 {
		return nil, fmt.Errorf("无效卷号")
	}
	written := p.Meta.CurrentChapter
	if written <= 0 {
		return nil, fmt.Errorf("尚无已写章节，请先完成至少 1 章再 Replan")
	}
	fromChapter := opts.FromChapter
	if fromChapter <= 0 {
		fromChapter = written + 1
	}
	if fromChapter <= written {
		return nil, fmt.Errorf("起始章 %d 须大于已写章 %d", fromChapter, written)
	}

	path := p.VolumeOutlinePath(opts.Volume)
	oldBody, _ := os.ReadFile(path)
	if len(oldBody) == 0 {
		return nil, fmt.Errorf("第 %d 卷尚无卷纲，请先用 nova plan %d 生成", opts.Volume, opts.Volume)
	}

	master, _ := os.ReadFile(fmt.Sprintf("%s/大纲/总纲.md", p.Root))
	settings := readDirConcat(p.SettingsDir())
	summaries := collectWrittenSummaries(p, written, 12000)
	entities := formatEntities(st, 50)
	foreshadows := formatOpenForeshadows(st)

	notes := strings.TrimSpace(opts.Notes)
	if notes == "" {
		notes = "(无)"
	}

	userPrompt := fmt.Sprintf(`请基于已写正文事实，重新规划第 %d 卷卷纲。

## 规划范围
- 已写至第 %d 章（这些章的事实不可推翻）
- 从第 %d 章起重新规划后续章纲
- 当前卷号：第 %d 卷

## 作者备注
%s

## 当前卷纲（待调整）
%s

## 总纲
%s

## 设定摘要
%s

## 已写章节摘要链（第 1–%d 章，事实依据）
%s

## 实体状态
%s

## 开放伏笔
%s

输出要求：
1. 输出完整的新卷纲 Markdown（含已写章 + 后续章）
2. 第 1–%d 章：保留与已写正文一致的内容，每章标题行后标注「> 状态：已完成」
3. 第 %d 章起：按新规划编写，格式同原卷纲（### 第N章 · 标题 + 核心冲突/爽点/伏笔）
4. 若原卷纲某章已偏离正文，标注「> 状态：偏离」并简述原因
5. 若原卷纲某章被废弃，标注「> 状态：废弃」
6. 不得与摘要链、实体、开放伏笔矛盾；新章纲须可执行，不写「待定」`,
		opts.Volume,
		written, fromChapter, opts.Volume,
		notes,
		string(oldBody),
		string(master),
		settings,
		written,
		ifEmpty("(暂无摘要)", summaries),
		ifEmpty("(暂无实体)", entities),
		ifEmpty("(无开放伏笔)", foreshadows),
		written,
		fromChapter,
	)

	content, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.ReplanSystem(prompts.BookContext{
			Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
			Protagonist: p.Meta.Protagonist, Cheat: p.Meta.Cheat, Synopsis: p.Meta.Synopsis,
			Volume: opts.Volume, Chapter: fromChapter,
		}),
		UserPrompt: userPrompt,
		Tools:      true,
	})
	if err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)

	return &ReplanResult{
		ProposedContent: content,
		OldContent:      string(oldBody),
		FromChapter:     fromChapter,
		WrittenThrough:  written,
		Report: &report.Report{
			Stage:   fmt.Sprintf("卷纲 Replan 第%d卷", opts.Volume),
			Status:  report.StatusNeedsAction,
			Summary: fmt.Sprintf("已生成第 %d 卷新卷纲草案（从第 %d 章起调整，已写至第 %d 章）", opts.Volume, fromChapter, written),
			NextSteps: []string{
				"预览 diff 并确认后应用",
				fmt.Sprintf("nova write %d", fromChapter),
			},
		},
	}, nil
}

func collectWrittenSummaries(p *project.Project, throughChapter, maxRunes int) string {
	var blocks []string
	total := 0
	omitted := 0
	for i := throughChapter; i >= 1; i-- {
		data, err := os.ReadFile(p.SummaryPath(i))
		if err != nil {
			continue
		}
		block := fmt.Sprintf("## 第%d章摘要\n%s", i, strings.TrimSpace(string(data)))
		n := utf8.RuneCountInString(block)
		if total+n > maxRunes {
			omitted = i
			break
		}
		blocks = append([]string{block}, blocks...)
		total += n
	}
	if omitted > 0 {
		blocks = append([]string{fmt.Sprintf("> … 第 1–%d 章摘要已省略（token 预算）", omitted)}, blocks...)
	}
	return strings.Join(blocks, "\n\n")
}

func formatEntities(st *store.Store, limit int) string {
	if st == nil {
		return ""
	}
	entities, err := st.ListEntities("", limit)
	if err != nil || len(entities) == 0 {
		return ""
	}
	var parts []string
	for _, e := range entities {
		state := truncateRunes(e.StateJSON, 280)
		parts = append(parts, fmt.Sprintf("- [%s] %s（%s，更新至第%d章）：%s",
			e.Type, e.Name, e.ID, e.LastChapter, state))
	}
	return strings.Join(parts, "\n")
}

func formatOpenForeshadows(st *store.Store) string {
	if st == nil {
		return ""
	}
	fs, err := st.ListForeshadows("open")
	if err != nil || len(fs) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fs {
		parts = append(parts, fmt.Sprintf("- [%s] 第%d章埋设：%s", f.ID, f.PlantedChapter, f.Description))
	}
	return strings.Join(parts, "\n")
}

func ifEmpty(fallback, s string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
