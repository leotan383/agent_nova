package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/tools"
)

// AIDetectMetrics AI 味检测结构化指标。
type AIDetectMetrics struct {
	AIScore     float64          `json:"ai_score"`
	HumanScore  float64          `json:"human_score"`
	RiskLevel   string           `json:"risk_level"`
	Publishable bool             `json:"publishable"`
	Signals     []string         `json:"signals"`
	Hotspots    []AIDetectHotspot `json:"hotspots"`
}

type AIDetectHotspot struct {
	Excerpt string `json:"excerpt"`
	Reason  string `json:"reason"`
	Fix     string `json:"fix"`
}

type AIDetectWorkflow struct {
	Agent *agent.Agent
}

func NewAIDetectWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *AIDetectWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &AIDetectWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func (w *AIDetectWorkflow) DetectChapter(ctx context.Context, p *project.Project, chapter int) (*report.Report, error) {
	path, body := loadChapterFile(p, chapter)
	if path == "" {
		return &report.Report{
			Stage: "AI味检测", Status: report.StatusFailed,
			Summary: fmt.Sprintf("第 %d 章正文不存在", chapter),
		}, nil
	}
	body = normalizeChapterBody(body)
	if strings.TrimSpace(body) == "" {
		return &report.Report{
			Stage: "AI味检测", Status: report.StatusFailed,
			Summary: "正文为空，无法检测",
		}, nil
	}

	content, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.AIDetectSystem(),
		UserPrompt:   fmt.Sprintf("请检测以下网文章节正文的 AI 生成痕迹：\n\n%s", body),
		Tools:        false,
	})
	if err != nil {
		return nil, err
	}

	checkPath := p.AICheckPath(chapter)
	if err := os.MkdirAll(filepath.Dir(checkPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(checkPath, []byte(content), 0o644); err != nil {
		return nil, err
	}

	metrics := parseAIDetectMetrics(content)
	summary := formatAIDetectSummary(metrics)
	return &report.Report{
		Stage:     fmt.Sprintf("AI味检测 第%d章", chapter),
		Status:    report.StatusDone,
		Summary:   summary,
		Artifacts: []string{checkPath},
	}, nil
}

func parseAIDetectMetrics(content string) AIDetectMetrics {
	var m AIDetectMetrics
	jsonRaw, err := agent.ExtractJSONBlock(content)
	if err != nil {
		return m
	}
	_ = json.Unmarshal([]byte(jsonRaw), &m)
	return m
}

func formatAIDetectSummary(m AIDetectMetrics) string {
	if m.AIScore <= 0 && m.RiskLevel == "" {
		return "AI 味检测完成"
	}
	risk := riskLevelLabel(m.RiskLevel)
	if m.AIScore > 0 {
		return fmt.Sprintf("AI 味 %.1f/10 · %s", m.AIScore, risk)
	}
	return fmt.Sprintf("风险 %s", risk)
}

func riskLevelLabel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "low":
		return "低风险"
	case "medium":
		return "中风险"
	case "high":
		return "高风险"
	default:
		return "待评估"
	}
}
