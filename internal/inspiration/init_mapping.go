package inspiration

import (
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
)

// Prefill 立项预填字段。
type Prefill struct {
	Title       string `json:"title"`
	Genre       string `json:"genre"`
	Style       string `json:"style"`
	Synopsis    string `json:"synopsis"`
	Protagonist string `json:"protagonist"`
	Cheat       string `json:"cheat"`
	SeedPrompt  string `json:"seed_prompt"`
}

// ToPrefill 将灵感转为创建书预填。
func ToPrefill(insp Inspiration) Prefill {
	genre := strings.TrimSpace(insp.Genre)
	if genre == "" {
		genre = "玄幻"
	}
	style := strings.TrimSpace(insp.Style)
	if style == "" {
		style = "热血"
	}
	title := strings.TrimSpace(insp.Title)
	if title == "" {
		title = DisplayTitle(insp)
	}
	synopsis := strings.TrimSpace(insp.Synopsis)
	if synopsis == "" {
		synopsis = strings.TrimSpace(insp.Spark)
	}
	return Prefill{
		Title:       title,
		Genre:       genre,
		Style:       style,
		Synopsis:    synopsis,
		Protagonist: strings.TrimSpace(insp.Protagonist),
		Cheat:       strings.TrimSpace(insp.Cheat),
		SeedPrompt:  DiscoverSeedPrompt(insp),
	}
}

// ToInitInput 映射到 project.InitInput（仅填有值字段，调用方补默认值）。
func ToInitInput(insp Inspiration, dir string) project.InitInput {
	p := ToPrefill(insp)
	return project.InitInput{
		Dir:          dir,
		Title:        p.Title,
		Genre:        p.Genre,
		Style:        p.Style,
		Synopsis:     p.Synopsis,
		Protagonist:  p.Protagonist,
		Cheat:        p.Cheat,
		TargetWords:  project.DefaultTargetWords,
		ChapterWords: project.DefaultChapterWords,
	}
}

// DiscoverSeedPrompt 生成 AI 探讨首轮种子。
func DiscoverSeedPrompt(insp Inspiration) string {
	title := DisplayTitle(insp)
	var b strings.Builder
	b.WriteString("我想基于下面这个灵感写书，请帮我一起完善细节：\n\n")
	if title != "" {
		fmt.Fprintf(&b, "【标题】%s\n", title)
	}
	if g := strings.TrimSpace(insp.Genre); g != "" {
		fmt.Fprintf(&b, "【题材】%s\n", g)
	}
	if s := strings.TrimSpace(insp.Spark); s != "" {
		fmt.Fprintf(&b, "【核心想法】\n%s\n", s)
	}
	if p := strings.TrimSpace(insp.Protagonist); p != "" {
		fmt.Fprintf(&b, "【主角】%s\n", p)
	}
	if c := strings.TrimSpace(insp.Cheat); c != "" {
		fmt.Fprintf(&b, "【金手指】%s\n", c)
	}
	if syn := strings.TrimSpace(insp.Synopsis); syn != "" && syn != strings.TrimSpace(insp.Spark) {
		fmt.Fprintf(&b, "【简介】%s\n", syn)
	}
	return strings.TrimSpace(b.String())
}
