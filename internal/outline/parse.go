package outline

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	chapterHeaderRe = regexp.MustCompile(`(?m)^#{1,4}\s*第\s*0*(\d+)\s*章(?:\s*[·•·\-—]\s*(.+))?\s*$`)
	planStatusRe    = regexp.MustCompile(`(?m)^>\s*状态[：:]\s*(已完成|偏离|废弃)`)
)

// Entry 卷纲中的章条目。
type Entry struct {
	Chapter        int
	Title          string
	Volume         int
	Preview        string
	PlanStatus     string // done | deviated | abandoned | planned
	PlanStatusNote string
}

// ParseVolumeOutline 解析卷纲 Markdown 中的章条目。
func ParseVolumeOutline(volume int, body string) []Entry {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	loc := chapterHeaderRe.FindAllStringSubmatchIndex(body, -1)
	if len(loc) == 0 {
		return nil
	}
	var out []Entry
	for i, idx := range loc {
		numStr := body[idx[2]:idx[3]]
		num, _ := strconv.Atoi(numStr)
		if num <= 0 {
			continue
		}
		title := ""
		if idx[4] >= 0 && idx[5] > idx[4] {
			title = strings.TrimSpace(body[idx[4]:idx[5]])
		}
		start := idx[1]
		end := len(body)
		if i+1 < len(loc) {
			end = loc[i+1][0]
		}
		block := strings.TrimSpace(body[start:end])
		planStatus, note := parsePlanStatus(block)
		out = append(out, Entry{
			Chapter:        num,
			Title:          title,
			Volume:         volume,
			Preview:        previewBlock(block, 120),
			PlanStatus:     planStatus,
			PlanStatusNote: note,
		})
	}
	return out
}

func parsePlanStatus(block string) (status, note string) {
	loc := planStatusRe.FindStringSubmatchIndex(block)
	if loc == nil {
		return "planned", ""
	}
	raw := block[loc[2]:loc[3]]
	switch raw {
	case "已完成":
		status = "done"
	case "偏离":
		status = "deviated"
	case "废弃":
		status = "abandoned"
	default:
		status = "planned"
	}
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ">") && !planStatusRe.MatchString(line) {
			note = strings.TrimPrefix(line, ">")
			note = strings.TrimSpace(strings.TrimPrefix(note, "状态："))
			break
		}
	}
	return status, note
}

func previewBlock(block string, maxRunes int) string {
	lines := strings.Split(block, "\n")
	var kept []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, ">") {
			continue
		}
		kept = append(kept, line)
	}
	s := strings.Join(kept, " ")
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return s
}

// ExtractChapterSection 提取卷纲中某一章的段落（供写章上下文复用）。
func ExtractChapterSection(full string, chapter int) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return ""
	}
	matches := chapterHeaderRe.FindAllStringSubmatchIndex(full, -1)
	if len(matches) == 0 {
		return truncateRunes(full, 1500)
	}
	target := -1
	for i, loc := range matches {
		numStr := full[loc[2]:loc[3]]
		n, _ := strconv.Atoi(numStr)
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

// FormatChapterHeader 格式化章标题行。
func FormatChapterHeader(chapter int, title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Sprintf("### 第%d章", chapter)
	}
	return fmt.Sprintf("### 第%d章 · %s", chapter, title)
}
