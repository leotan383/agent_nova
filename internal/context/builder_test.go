package contextbuilder

import "testing"

func TestExtractChapterOutline(t *testing.T) {
	full := `# 第一卷

### 第1章 · 开局
- 核心冲突：被退婚

### 第2章 · 觉醒
- 核心冲突：测试天赋
- 爽点：金手指激活

### 第3章 · 反击
- 核心冲突：家族大会
`
	got := extractChapterOutline(full, 2)
	if got == "" {
		t.Fatal("expected non-empty outline")
	}
	if !contains(got, "觉醒") {
		t.Fatalf("expected chapter 2 content, got: %q", got)
	}
	if contains(got, "第3章") {
		t.Fatalf("should not include next chapter, got: %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
