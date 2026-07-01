package contextbuilder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/store"
)

type Builder struct {
	Proj        *project.Project
	Store       *store.Store
	Config      *config.Config // 可选；有 API Key 时启用语义召回
	MemoryPrefs MemoryPrefs
}

// MemoryPrefs 写前记忆注入偏好（pin 固定 / exclude 排除）。
type MemoryPrefs struct {
	PinnedIDs   []string
	ExcludedIDs []string
}

type Snapshot struct {
	Chapter         int    `json:"chapter"`
	Volume          int    `json:"volume"`
	BookAnchor      string `json:"book_anchor"`
	ChapterOutline  string `json:"chapter_outline"`  // 从卷纲中提取的本章段落
	VolumeOutline   string `json:"volume_outline"`   // 卷纲全文（可能截断）
	RecentSummary   string `json:"recent_summary"`
	Settings        string `json:"settings"`
	Memories        string            `json:"memories"`
	MemoryRecalls   []MemoryRecallHit `json:"memory_recalls,omitempty"`
	OpenForeshadows string            `json:"open_foreshadows"`
	FTSHits         string            `json:"fts_hits"`
}

// Build 组装写章上下文快照。
// 策略：书籍锚点 → system prompt；本章章纲 + 动态事实 → user prompt 靠前位置。
func (b *Builder) Build(chapter, volume int) (*Snapshot, error) {
	if volume <= 0 {
		volume = 1
	}
	snap := &Snapshot{Chapter: chapter, Volume: volume}

	anchor := prompts.BookContext{
		Title:       b.Proj.Meta.Title,
		Genre:       b.Proj.Meta.Genre,
		Style:       b.Proj.Meta.WritingStyle(),
		Protagonist: b.Proj.Meta.Protagonist,
		Cheat:       b.Proj.Meta.Cheat,
		Synopsis:    b.Proj.Meta.Synopsis,
		Chapter:     chapter,
		Volume:      volume,
	}
	snap.BookAnchor = prompts.BookAnchor(anchor)

	volPath := b.Proj.VolumeOutlinePath(volume)
	if data, err := os.ReadFile(volPath); err == nil {
		full := string(data)
		snap.ChapterOutline = extractChapterOutline(full, chapter)
		snap.VolumeOutline = truncateRunes(full, 4000)
	}

	snap.RecentSummary = b.recentSummaries(chapter, 3)
	snap.Settings = b.settingsDigest()
	snap.OpenForeshadows = b.openForeshadows()

	if b.Store != nil {
		keywords := extractKeywords(
			snap.ChapterOutline, snap.RecentSummary,
			b.Proj.Meta.Protagonist, b.Proj.Meta.Cheat,
		)
		entityNames := relatedEntityNames(b.Store, keywords)
		recalls := RecallMemories(context.Background(), b.Store, b.Config, RecallInput{
			Chapter: chapter, Outline: snap.ChapterOutline, RecentSummary: snap.RecentSummary,
			Keywords: keywords, EntityNames: entityNames,
			Protagonist: b.Proj.Meta.Protagonist, Cheat: b.Proj.Meta.Cheat,
		})
		if len(recalls) == 0 {
			recalls = fallbackRecentMemories(b.Store)
		}
		recalls = applyMemoryPrefs(recalls, b.Store, b.MemoryPrefs)
		snap.MemoryRecalls = recalls
		snap.Memories = formatMemoryRecalls(recalls)

		ftsQuery := buildFTSQuery(keywords, chapter)
		hits, _ := b.Store.SearchFTS(ftsQuery, 5)
		for _, h := range hits {
			snap.FTSHits += fmt.Sprintf("%s: %s\n", h["kind"], h["snippet"])
		}
	}
	return snap, nil
}

// BookContext 返回供 system prompt 使用的书籍锚点结构。
func (b *Builder) BookContext(chapter, volume int) prompts.BookContext {
	if volume <= 0 {
		volume = 1
	}
	return prompts.BookContext{
		Title:       b.Proj.Meta.Title,
		Genre:       b.Proj.Meta.Genre,
		Style:       b.Proj.Meta.WritingStyle(),
		Protagonist: b.Proj.Meta.Protagonist,
		Cheat:       b.Proj.Meta.Cheat,
		Synopsis:    b.Proj.Meta.Synopsis,
		Chapter:     chapter,
		Volume:      volume,
	}
}

// ToContextPrompt 供 ContextAgent 生成任务书（参考材料，不含书籍锚点）。
func (s *Snapshot) ToContextPrompt() string {
	return fmt.Sprintf(`# 写作参考材料

## 目标
第 %d 章（第 %d 卷）

## 本章章纲（优先执行）
%s

## 近章摘要（衔接事实，不可矛盾）
%s

## 设定摘要
%s

## Open 伏笔（可择机回收，勿提前剧透无关伏笔）
%s

## 长期记忆
%s

## 检索命中
%s
`, s.Chapter, s.Volume,
		fallback(s.ChapterOutline, "（未找到本章章纲，请参考卷纲）\n"+truncateRunes(s.VolumeOutline, 2000)),
		fallback(s.RecentSummary, "（暂无前章摘要）"),
		fallback(s.Settings, "（暂无设定摘要）"),
		fallback(s.OpenForeshadows, "（暂无 open 伏笔）"),
		fallback(s.Memories, "（暂无记忆）"),
		fallback(s.FTSHits, "（无检索命中）"),
	)
}

// ToPrompt 完整上下文（CLI 展示用）。
func (s *Snapshot) ToPrompt() string {
	return s.BookAnchor + "\n\n---\n\n" + s.ToContextPrompt()
}

// ToWriteUserPrompt 供 WriteAgent：任务书 + 参考材料（书籍锚点已在 system）。
func (s *Snapshot) ToWriteUserPrompt(taskBook string) string {
	return fmt.Sprintf(`# 写作任务书（必须严格执行）

%s

---

%s`, strings.TrimSpace(taskBook), s.ToContextPrompt())
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

func (b *Builder) openForeshadows() string {
	if b.Store == nil {
		return ""
	}
	fs, err := b.Store.ListForeshadows("open")
	if err != nil || len(fs) == 0 {
		return ""
	}
	var parts []string
	for _, f := range fs {
		parts = append(parts, fmt.Sprintf("- [%s] 第%d章埋设：%s", f.ID, f.PlantedChapter, f.Description))
	}
	return strings.Join(parts, "\n")
}

var settingsPriority = []string{"主角", "世界观", "力量", "金手指", "反派", "设定"}

func (b *Builder) settingsDigest() string {
	type item struct {
		path    string
		content string
		priority int
	}
	var items []item

	_ = filepath.Walk(b.Proj.SettingsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(b.Proj.Root, path)
		base := filepath.Base(path)
		pri := len(settingsPriority) + 1
		for i, kw := range settingsPriority {
			if strings.Contains(base, kw) {
				pri = i
				break
			}
		}
		items = append(items, item{
			path: rel, content: truncateRunes(string(data), 800), priority: pri,
		})
		return nil
	})

	sort.Slice(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].path < items[j].path
	})

	var parts []string
	total := 0
	const maxTotal = 3500
	for _, it := range items {
		block := fmt.Sprintf("### %s\n%s", it.path, it.content)
		blockLen := len([]rune(block))
		if total+blockLen > maxTotal {
			break
		}
		parts = append(parts, block)
		total += blockLen
	}
	return strings.Join(parts, "\n\n")
}

var chapterHeaderRe = regexp.MustCompile(`(?m)^#{1,4}\s*第\s*0*(\d+)\s*章`)

func extractChapterOutline(full string, chapter int) string {
	if full == "" {
		return ""
	}
	matches := chapterHeaderRe.FindAllStringSubmatchIndex(full, -1)
	if len(matches) == 0 {
		return truncateRunes(full, 1500)
	}
	target := -1
	for i, loc := range matches {
		num := full[loc[2]:loc[3]]
		var n int
		fmt.Sscanf(num, "%d", &n)
		if n == chapter {
			target = i
			break
		}
	}
	if target < 0 {
		return truncateRunes(full, 1500)
	}
	start := matches[target][0]
	end := len(full)
	if target+1 < len(matches) {
		end = matches[target+1][0]
	}
	return strings.TrimSpace(full[start:end])
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func fallback(s, def string) string {
	if strings.TrimSpace(s) != "" {
		return s
	}
	return def
}

// fallbackRecentMemories 无关键词命中时回退为最近记忆（兼容旧行为）。
func fallbackRecentMemories(st *store.Store) []MemoryRecallHit {
	memories, _ := st.QueryMemories("", "", recallDefaultTopK)
	out := make([]MemoryRecallHit, 0, len(memories))
	for _, m := range memories {
		out = append(out, MemoryRecallHit{
			ID: m.ID, Category: m.Category, Subject: m.Subject, Content: m.Content,
			Source: "fallback", Reason: "近期写入", Score: 0,
		})
	}
	return applyMemoryBudget(out, recallMaxRunes)
}
