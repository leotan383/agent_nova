package workflows

import (
	"strings"
	"testing"
)

func TestParseAIDetectMetrics(t *testing.T) {
	report := `# AI 味检测

综合评估：AI 味 4.5/10

` + "```json\n" + `{"ai_score":4.5,"human_score":5.5,"risk_level":"medium","publishable":true,"signals":["排比过多"],"hotspots":[{"excerpt":"他深吸一口气","reason":"套路开头","fix":"换具体动作"}]}` + "\n```"
	m := parseAIDetectMetrics(report)
	if m.AIScore != 4.5 {
		t.Fatalf("ai_score got %v", m.AIScore)
	}
	if m.RiskLevel != "medium" {
		t.Fatalf("risk_level got %q", m.RiskLevel)
	}
	if len(m.Signals) != 1 || m.Signals[0] != "排比过多" {
		t.Fatalf("signals got %v", m.Signals)
	}
}

func TestRiskLevelLabel(t *testing.T) {
	if riskLevelLabel("high") != "高风险" {
		t.Fatal("expected 高风险")
	}
	if riskLevelLabel("") != "待评估" {
		t.Fatal("expected 待评估")
	}
}

func TestFormatAIDetectSummary(t *testing.T) {
	s := formatAIDetectSummary(AIDetectMetrics{AIScore: 7.2, RiskLevel: "high"})
	if !strings.Contains(s, "7.2") || !strings.Contains(s, "高风险") {
		t.Fatalf("got %q", s)
	}
}
