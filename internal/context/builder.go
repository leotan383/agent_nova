package contextbuilder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

type Builder struct {
	Proj  *project.Project
	Store *store.Store
}

type Snapshot struct {
	Chapter       int    `json:"chapter"`        // 目标章号
	Volume        int    `json:"volume"`         // 目标卷号
	Outline       string `json:"outline"`        // 卷纲/章纲全文（大纲/第NN卷.md）
	RecentSummary string `json:"recent_summary"` // 近 N 章摘要链
	Settings      string `json:"settings"`       // 设定集摘要（截断后）
	Memories      string `json:"memories"`       // 长期记忆 Top-K 拼接文本
	FTSHits       string `json:"fts_hits"`       // FTS 检索命中的章节/设定片段
}

// Build 组装写章上下文快照，供 ContextAgent / WriteAgent 注入 prompt。
func (b *Builder) Build(chapter, volume int) (*Snapshot, error) {
	if volume <= 0 {
		volume = 1
	}
	snap := &Snapshot{Chapter: chapter, Volume: volume}

	// 卷纲：本章所属卷的大纲与章纲（大纲/第NN卷.md）
	volPath := b.Proj.VolumeOutlinePath(volume)
	if data, err := os.ReadFile(volPath); err == nil {
		snap.Outline = string(data)
	}

	// 近章摘要链：向前取最多 3 章 summary，保证长篇连贯性
	snap.RecentSummary = b.recentSummaries(chapter, 3)

	// 设定摘要：设定集/*.md 截断至 800 字，控制 token 预算
	snap.Settings = b.settingsDigest()

	if b.Store != nil {
		// 长期记忆 Top-10：写前注入已沉淀的可复用知识
		memories, _ := b.Store.QueryMemories("", "", 10)
		var parts []string
		for _, m := range memories {
			parts = append(parts, fmt.Sprintf("[%s/%s] %s", m.Category, m.Subject, m.Content))
		}
		snap.Memories = strings.Join(parts, "\n")

		// FTS 检索：按章号关键词召回相关章节/设定片段
		hits, _ := b.Store.SearchFTS(fmt.Sprintf("第%d", chapter), 5)
		for _, h := range hits {
			snap.FTSHits += fmt.Sprintf("%s: %s\n", h["kind"], h["snippet"])
		}
	}
	return snap, nil
}

func (b *Builder) recentSummaries(beforeChapter, n int) string {
	var parts []string
	for i := beforeChapter - 1; i > 0 && len(parts) < n; i-- {
		path := b.Proj.SummaryPath(i)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parts = append([]string{fmt.Sprintf("## 第%d章摘要\n%s", i, string(data))}, parts...)
	}
	return strings.Join(parts, "\n\n")
}

func (b *Builder) settingsDigest() string {
	var parts []string
	_ = filepath.Walk(b.Proj.SettingsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(b.Proj.Root, path)
		content := string(data)
		if len([]rune(content)) > 800 {
			content = string([]rune(content)[:800]) + "..."
		}
		parts = append(parts, fmt.Sprintf("### %s\n%s", rel, content))
		return nil
	})
	return strings.Join(parts, "\n\n")
}

func (s *Snapshot) ToPrompt() string {
	return fmt.Sprintf(`# 写作上下文

## 目标章节
第 %d 章（第 %d 卷）

## 章纲/卷纲
%s

## 近章摘要
%s

## 设定摘要
%s

## 长期记忆
%s

## 检索命中
%s
`, s.Chapter, s.Volume, s.Outline, s.RecentSummary, s.Settings, s.Memories, s.FTSHits)
}
