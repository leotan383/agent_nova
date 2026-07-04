package agent

import (
	"encoding/json"
	"regexp"
	"strings"
)

var trailingCommaBeforeCloseRE = regexp.MustCompile(`,(\s*[}\]])`)

// SanitizeJSONObject 修复模型输出中常见的 JSON 语法问题（如尾随逗号）。
func SanitizeJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "\ufeff")
	raw = strings.ReplaceAll(raw, "\u201c", `"`)
	raw = strings.ReplaceAll(raw, "\u201d", `"`)
	raw = strings.ReplaceAll(raw, "\u2018", `'`)
	raw = strings.ReplaceAll(raw, "\u2019", `'`)
	for {
		next := trailingCommaBeforeCloseRE.ReplaceAllString(raw, `$1`)
		if next == raw {
			break
		}
		raw = next
	}
	return raw
}

// UnmarshalJSONObject 尝试解析 JSON 对象，必要时先做常见修复。
func UnmarshalJSONObject(raw string, dest any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return json.Unmarshal([]byte(raw), dest)
	}
	candidates := []string{raw, SanitizeJSONObject(raw)}
	var lastErr error
	for _, candidate := range candidates {
		if err := json.Unmarshal([]byte(candidate), dest); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}
