package workflows

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/wiki"
)

// FillSettingResult 设定 AI 填充结果。
type FillSettingResult struct {
	SettingID string
	OldBody   string
	NewBody   string
}

// FillSettingFromPlot 根据已写正文摘要补全设定 Markdown 空白字段。
func FillSettingFromPlot(ctx context.Context, ag *agent.Agent, p *project.Project, st *store.Store, settingID string) (*FillSettingResult, error) {
	settingID = strings.TrimSpace(settingID)
	if settingID == "" {
		return nil, fmt.Errorf("请选择设定条目")
	}
	content, err := wiki.Get(p, st, settingID)
	if err != nil {
		return nil, err
	}
	if content.Kind != wiki.KindSetting || !content.Editable {
		return nil, fmt.Errorf("仅支持设定集 Markdown 文档")
	}

	written := p.Meta.CurrentChapter
	if written <= 0 {
		return nil, fmt.Errorf("尚无已写章节，请先完成正文写作后再填充设定")
	}

	summaries := collectWrittenSummaries(p, written, 14000)
	entityBlock := formatRelatedEntities(st, content.Title)
	memoryBlock := formatRelatedMemories(st, content.Title)
	master, _ := os.ReadFile(fmt.Sprintf("%s/大纲/总纲.md", p.Root))

	userPrompt := fmt.Sprintf(`请根据已写正文，补全以下设定 Markdown 文档中的空白字段。

## 设定文档
标题：%s
路径：%s

## 当前正文（请保留结构，补全空白）
%s

## 总纲
%s

## 已写章节摘要链（第 1–%d 章，事实依据）
%s

## 相关实体状态（AI 提取）
%s

## 相关记忆
%s

输出要求：
1. 输出完整 Markdown 正文（含顶部元信息行）
2. 保留原有 ## 标题结构；已有内容若无矛盾可保留，空白字段须填写
3. 只依据摘要与实体/记忆，不要编造未出现的重要情节
4. 性格、背景、目标等须与正文一致；不确定处可简短标注「待确认」
5. 不要输出解释，只输出 Markdown`,
		content.Title,
		content.Path,
		content.Body,
		string(master),
		written,
		ifEmpty("(暂无摘要)", summaries),
		ifEmpty("(无相关实体)", entityBlock),
		ifEmpty("(无相关记忆)", memoryBlock),
	)

	newBody, err := ag.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.FillSettingSystem(prompts.BookContext{
			Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
			Protagonist: p.Meta.Protagonist, Cheat: p.Meta.Cheat, Synopsis: p.Meta.Synopsis,
		}),
		UserPrompt: userPrompt,
	})
	if err != nil {
		return nil, err
	}
	newBody = strings.TrimSpace(newBody)
	if newBody == "" {
		return nil, fmt.Errorf("AI 未返回有效内容")
	}

	return &FillSettingResult{
		SettingID: settingID,
		OldBody:   content.Body,
		NewBody:   newBody,
	}, nil
}

func formatRelatedEntities(st *store.Store, title string) string {
	if st == nil {
		return ""
	}
	entities, err := st.ListEntities("", 200)
	if err != nil || len(entities) == 0 {
		return ""
	}
	key := normalizeSettingName(title)
	var parts []string
	for _, e := range entities {
		name := normalizeSettingName(e.Name)
		if name == "" {
			continue
		}
		if !namesRelated(key, name) {
			continue
		}
		state := truncateRunes(e.StateJSON, 320)
		parts = append(parts, fmt.Sprintf("- [%s] %s（更新至第%d章）：%s",
			e.Type, e.Name, e.LastChapter, state))
	}
	return strings.Join(parts, "\n")
}

func formatRelatedMemories(st *store.Store, title string) string {
	if st == nil {
		return ""
	}
	items, err := st.QueryMemories("", "", 80)
	if err != nil || len(items) == 0 {
		return ""
	}
	key := normalizeSettingName(title)
	var parts []string
	for _, m := range items {
		subject := normalizeSettingName(m.Subject)
		if subject != "" && !namesRelated(key, subject) {
			continue
		}
		if m.Category != "character" && m.Category != "world" && m.Category != "plot" {
			if subject == "" || !namesRelated(key, subject) {
				continue
			}
		}
		parts = append(parts, fmt.Sprintf("- [%s] %s：%s", m.Category, m.Subject, truncateRunes(m.Content, 200)))
	}
	return strings.Join(parts, "\n")
}

func normalizeSettingName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "卡")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func namesRelated(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	return strings.Contains(a, b) || strings.Contains(b, a)
}
