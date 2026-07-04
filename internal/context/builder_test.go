package contextbuilder

import (
	"strings"
	"testing"

	"github.com/tanlian/agent_nova/internal/outline"
)

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
	got := outline.ExtractChapterSection(full, 2)
	if got == "" {
		t.Fatal("expected non-empty outline")
	}
	if !strings.Contains(got, "觉醒") {
		t.Fatalf("expected chapter 2 content, got: %q", got)
	}
	if strings.Contains(got, "第3章") {
		t.Fatalf("should not include next chapter, got: %q", got)
	}
}
