package workflows

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/agent"
)

var (
	chapterHeadingMD    = regexp.MustCompile(`^#{1,6}\s*第\s*[0-9一二三四五六七八九十百千万零〇两\d]+\s*章`)
	chapterHeadingPlain = regexp.MustCompile(`^第\s*[0-9一二三四五六七八九十百千万零〇两\d]+\s*章`)
	reviewAppendixHead  = regexp.MustCompile(`^#{1,6}\s*(润色说明|修改说明|修订说明|润色记录|修改记录|修改对照)\s*$`)
)

// normalizeChapterBody 去掉审查报告前言与末尾说明，确保正文从章节标题或叙事开头起。
func normalizeChapterBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return body
	}
	if idx := findChapterContentStart(body); idx > 0 {
		body = strings.TrimSpace(body[idx:])
	}
	body = stripReviewAppendixSuffix(body)
	return stripReviewMetricsSuffix(body)
}

func findChapterContentStart(s string) int {
	lines := strings.Split(s, "\n")
	pos := 0
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			pos += len(line) + 1
			continue
		}
		if isChapterStartLine(trimmed) {
			return pos
		}
		if isBodyPreambleLine(trimmed) {
			pos += len(line) + 1
			continue
		}
		if trimmed == "---" || trimmed == "***" || trimmed == "___" {
			pos += len(line) + 1
			for j := i + 1; j < len(lines); j++ {
				next := strings.TrimSpace(lines[j])
				if next == "" {
					continue
				}
				if isChapterStartLine(next) {
					// 计算下一非空行的字节偏移
					offset := pos
					for k := i + 1; k < j; k++ {
						offset += len(lines[k]) + 1
					}
					return offset
				}
				break
			}
			continue
		}
		// 首段已是叙事正文则保留
		return 0
	}
	return 0
}

func isChapterStartLine(line string) bool {
	return chapterHeadingMD.MatchString(line) || chapterHeadingPlain.MatchString(line)
}

func isBodyPreambleLine(line string) bool {
	for _, k := range []string{
		"以下为", "以下是", "润色版本", "润色说明", "修改说明", "修订说明",
		"重点调整", "审查意见", "根据审查", "修改的润色", "润色后",
	} {
		if strings.Contains(line, k) {
			return true
		}
	}
	return false
}

func isReviewAppendixHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if reviewAppendixHead.MatchString(trimmed) {
		return true
	}
	for _, k := range []string{"润色说明", "修改说明", "修订说明", "修改对照", "润色记录", "修改记录"} {
		if trimmed == "**"+k+"**" {
			return true
		}
	}
	return false
}

// stripReviewAppendixSuffix 去掉正文末尾误写入的润色说明、修改对照表等审查附录。
func stripReviewAppendixSuffix(body string) string {
	lines := strings.Split(body, "\n")
	cutLine := -1
	for i, line := range lines {
		if isReviewAppendixHeading(line) {
			cutLine = i
			break
		}
	}
	if cutLine < 0 {
		return body
	}
	start := cutLine
	for start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start--
	}
	if start > 0 && strings.TrimSpace(lines[start-1]) == "---" {
		start--
	}
	for start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start--
	}
	return strings.TrimRight(strings.Join(lines[:start], "\n"), " \t\r\n")
}

func extractPolishedBody(reviewed, fallback string) string {
	markers := []string{"润色版全文", "润色版正文", "修订版全文", "修订后正文", "## 润色版正文", "## 润色后正文", "## 润色", "润色版", "修订版"}
	for _, m := range markers {
		if idx := strings.Index(reviewed, m); idx >= 0 {
			body := strings.TrimSpace(reviewed[idx+len(m):])
			body = strings.TrimLeft(body, "：:\n")
			if utf8.RuneCountInString(body) > 200 {
				return normalizeChapterBody(body)
			}
		}
	}
	if idx := findChapterHeadingStart(reviewed); idx >= 0 {
		body := strings.TrimSpace(reviewed[idx:])
		if utf8.RuneCountInString(body) > 200 {
			return normalizeChapterBody(body)
		}
	}
	if parts := strings.Split(reviewed, "\n---\n"); len(parts) >= 2 {
		last := strings.TrimSpace(parts[len(parts)-1])
		if utf8.RuneCountInString(last) > 200 {
			return normalizeChapterBody(last)
		}
	}
	return normalizeChapterBody(fallback)
}

func stripReviewMetricsSuffix(body string) string {
	body = strings.TrimRight(body, " \t\r\n")
	if idx := strings.LastIndex(body, "```json"); idx >= 0 {
		tail := body[idx:]
		if end := strings.LastIndex(tail, "```"); end > 7 {
			jsonRaw, err := agent.ExtractJSONBlock(tail)
			if err == nil && strings.Contains(jsonRaw, "hook_score") {
				return strings.TrimRight(body[:idx], " \t\r\n")
			}
		}
	}
	hookIdx := strings.LastIndex(body, `"hook_score"`)
	if hookIdx < 0 {
		return body
	}
	objStart := strings.LastIndex(body[:hookIdx], "{")
	objEnd := strings.LastIndex(body, "}")
	if objStart < 0 || objEnd <= objStart {
		return body
	}
	candidate := body[objStart : objEnd+1]
	var m map[string]any
	if err := json.Unmarshal([]byte(candidate), &m); err != nil {
		return body
	}
	if _, ok := m["hook_score"]; !ok {
		return body
	}
	return strings.TrimRight(body[:objStart], " \t\r\n")
}

func findChapterHeadingStart(s string) int {
	if idx := findChapterContentStart(s); idx >= 0 {
		return idx
	}
	return -1
}
