package workflows

import (
	"strings"
	"testing"
)

func TestStripReviewMetricsSuffix(t *testing.T) {
	chapter := "# 第4章 暗流\n\n正文段落。\n"
	jsonTail := `{"hook_score":7,"cool_point":"爽点","debt":"伏笔","issues":["【修正】问题"]}`

	got := stripReviewMetricsSuffix(chapter + "\n" + jsonTail)
	if got != strings.TrimRight(chapter, "\n") {
		t.Fatalf("bare json: got %q want %q", got, strings.TrimRight(chapter, "\n"))
	}

	fenced := chapter + "\n```json\n" + jsonTail + "\n```"
	got = stripReviewMetricsSuffix(fenced)
	if got != strings.TrimRight(chapter, "\n") {
		t.Fatalf("fenced json: got %q want %q", got, strings.TrimRight(chapter, "\n"))
	}

	plain := chapter + "没有 JSON 结尾。"
	if stripReviewMetricsSuffix(plain) != plain {
		t.Fatal("expected plain chapter unchanged")
	}
}

func TestExtractPolishedBodyStripsMetrics(t *testing.T) {
	longBody := strings.Repeat("这是一段足够长的正文内容，用于通过提取阈值。", 12)
	reviewed := "## 润色版正文\n\n# 第4章\n\n" + longBody + "\n\n```json\n{\"hook_score\":8,\"cool_point\":\"\",\"debt\":\"\",\"issues\":[]}\n```"
	got := extractPolishedBody(reviewed, "fallback")
	want := "# 第4章\n\n" + longBody
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestExtractPolishedBodyStripsPreamble(t *testing.T) {
	longBody := strings.Repeat("林枫深吸一口气，拨开了最后一片藤蔓。", 15)
	reviewed := "## 润色版正文\n\n以下为根据审查意见修改的润色版本，重点调整了：（1）情感铺垫。\n\n---\n\n# 第八章 洞穴暗影\n\n" + longBody
	got := extractPolishedBody(reviewed, "fallback")
	want := "# 第八章 洞穴暗影\n\n" + longBody
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestStripReviewAppendixSuffix(t *testing.T) {
	chapter := "# 第八章 洞穴暗影\n\n" + strings.Repeat("时间在洞穴外，那两个修士正在一步步逼近。", 12)
	appendix := "\n\n---\n\n## 润色说明\n\n| 修改点 | 原文 | 润色后 | 理由 |\n|---|---|---|---|\n| 章首衔接 | 原句 | 新句 | 过渡 |"
	got := normalizeChapterBody(chapter + appendix)
	if got != chapter {
		t.Fatalf("got %q want %q", got, chapter)
	}
}

func TestExtractPolishedBodyStripsAppendix(t *testing.T) {
	longBody := strings.Repeat("时间在洞穴外，那两个修士正在一步步逼近。", 15)
	appendix := "\n\n---\n\n## 润色说明\n\n| 修改点 | 原文 | 润色后 | 理由 |\n|---|---|---|---|\n| 节奏 | a | b | c |"
	reviewed := "## 润色版正文\n\n# 第八章 洞穴暗影\n\n" + longBody + appendix
	got := extractPolishedBody(reviewed, "fallback")
	want := "# 第八章 洞穴暗影\n\n" + longBody
	if got != want {
		t.Fatalf("got %q", got)
	}
}
