package outline

import (
	"strings"
	"testing"
)

func TestParseVolumeOutline(t *testing.T) {
	body := `# 第一卷

### 第1章 · 穿越
- 核心冲突：觉醒

> 状态：已完成

### 第2章 · 拜师
- 爽点：展示金手指

### 第3章 · 试炼
> 状态：偏离
> 正文改走支线

`
	entries := ParseVolumeOutline(1, body)
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
	if entries[0].Chapter != 1 || entries[0].PlanStatus != "done" {
		t.Fatalf("ch1: %+v", entries[0])
	}
	if entries[1].PlanStatus != "planned" {
		t.Fatalf("ch2 plan status = %q", entries[1].PlanStatus)
	}
	if entries[2].PlanStatus != "deviated" {
		t.Fatalf("ch3 plan status = %q", entries[2].PlanStatus)
	}
}

func TestExtractChapterSection(t *testing.T) {
	body := "### 第1章 · A\nline1\n\n### 第2章 · B\nline2"
	got := ExtractChapterSection(body, 2)
	if got == "" || !strings.Contains(got, "line2") {
		t.Fatalf("got %q", got)
	}
}
