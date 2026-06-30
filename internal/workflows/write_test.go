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
