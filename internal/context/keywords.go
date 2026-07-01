package contextbuilder

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	cjkWordRe   = regexp.MustCompile(`[\p{Han}]{2,8}`)
	quotedRe    = regexp.MustCompile(`[「『"']([^」』"']{2,12})[」』"']`)
	outlineStop = map[string]struct{}{
		"本章": {}, "核心": {}, "冲突": {}, "爽点": {}, "伏笔": {}, "章节": {}, "任务": {},
		"目标": {}, "情节": {}, "描写": {}, "需要": {}, "建议": {}, "可以": {}, "应该": {},
		"通过": {}, "进行": {}, "出现": {}, "展示": {}, "设置": {}, "安排": {}, "完成": {},
		"第一": {}, "第二": {}, "第三": {}, "卷纲": {}, "章纲": {}, "回收": {}, "埋设": {},
	}
)

// extractKeywords 从章纲、摘要、书籍锚点信息提取检索关键词（去重、限长）。
func extractKeywords(chapterOutline, recentSummary, protagonist, cheat string, extra ...string) []string {
	seen := map[string]struct{}{}
	var out []string

	add := func(w string) {
		w = strings.TrimSpace(w)
		if w == "" || len([]rune(w)) < 2 {
			return
		}
		if _, skip := outlineStop[w]; skip {
			return
		}
		key := strings.ToLower(w)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, w)
	}

	for _, fixed := range []string{protagonist, cheat} {
		add(fixed)
	}
	for _, s := range extra {
		add(s)
	}

	corpus := chapterOutline + "\n" + recentSummary
	for _, m := range quotedRe.FindAllStringSubmatch(corpus, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, w := range cjkWordRe.FindAllString(corpus, -1) {
		add(w)
	}

	// 拉丁/数字混合专名（如 NPC 代号）
	for _, tok := range strings.FieldsFunc(corpus, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSpace(r)
	}) {
		if len(tok) >= 3 && len([]rune(tok)) <= 16 {
			add(tok)
		}
	}

	if len(out) > 24 {
		out = out[:24]
	}
	return out
}
