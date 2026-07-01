package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	contextbuilder "github.com/tanlian/agent_nova/internal/context"
	"github.com/tanlian/agent_nova/internal/agent"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/prompts"
	"github.com/tanlian/agent_nova/internal/report"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/tools"
	"github.com/tanlian/agent_nova/internal/usage"
	"github.com/tanlian/agent_nova/internal/version"
)

type WriteWorkflow struct {
	Agent  *agent.Agent
	Config *config.Config
}

func NewWriteWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *WriteWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &WriteWorkflow{
		Agent:  agent.New(agent.Options{Config: cfg, Registry: reg}),
		Config: cfg,
	}
}

type WriteOptions struct {
	Chapter int
	Volume  int
	Resume  bool
	Stream  bool
	OnDelta func(string) error
	OnStep  func(step, message string) error
}

func (w *WriteWorkflow) WriteChapter(ctx context.Context, p *project.Project, st *store.Store, opts WriteOptions) (*report.Report, error) {
	emit := func(step, msg string) {
		if opts.OnStep != nil {
			_ = opts.OnStep(step, msg)
		}
	}
	var usageAcc agent.UsageAccumulator
	withUsage := func(in agent.RunInput) agent.RunInput {
		in.UsageAcc = &usageAcc
		return in
	}

	// 写前 gate：检查 phase、卷纲、上一章摘要、索引是否过期
	gate := pipeline.RunGate(p, st, opts.Chapter, pipeline.GatePrewrite)
	if !gate.OK {
		return &report.Report{
			Stage: fmt.Sprintf("写章 第%d章", opts.Chapter), Status: report.StatusNeedsAction,
			Summary: "写前检查未通过", Issues: gate.Issues,
			NextSteps: []string{"nova plan 1", "nova index rebuild", "补全上一章摘要"},
		}, nil
	}
	emit("gate", "写前检查通过")

	// 断点续跑：从 run_ledger 决定从 draft/review/summary 哪一步开始
	ledger, _ := pipeline.LoadLedger(p.RunLedgerPath())
	ledger.Chapter = opts.Chapter
	startStep := "draft"
	if opts.Resume {
		startStep = ledger.ResumeStep()
		if startStep == "done" {
			startStep = "draft"
		}
	}

	// 组装写章上下文：近章摘要、设定、卷纲、记忆、FTS 命中
	emit("context", "组装写作上下文")
	cb := contextbuilder.Builder{Proj: p, Store: st, Config: w.Config}
	snap, err := cb.Build(opts.Chapter, opts.Volume)
	if err != nil {
		return nil, err
	}
	var chapterPath string
	var content string

	// Step 1: 起草 — ContextAgent 任务书 → WriteAgent 正文
	if startStep == "draft" {
		anchor := cb.BookContext(opts.Chapter, opts.Volume)
		emit("taskbook", "生成写作任务书")
		taskBook, err := w.Agent.Run(ctx, withUsage(agent.RunInput{
			SystemPrompt: prompts.ContextSystem(anchor),
			UserPrompt:   snap.ToContextPrompt() + "\n请输出写作任务书。",
		}))
		if err != nil {
			return nil, err
		}

		emit("draft", "起草正文")
		content, err = w.Agent.Run(ctx, withUsage(agent.RunInput{
			SystemPrompt: prompts.WriteSystem(anchor),
			UserPrompt:   snap.ToWriteUserPrompt(taskBook),
			Stream:       opts.Stream,
			OnDelta:      opts.OnDelta,
		}))
		if err != nil {
			ledger.Record("draft", "failed", err.Error())
			_ = ledger.Save(p.RunLedgerPath())
			return nil, err
		}

		// 保存正文
		title := pipeline.ParseChapterTitle(content)
		chapterPath, err = pipeline.SaveChapterWithVersion(p, opts.Chapter, title, content, version.SourceWriteDraft, "写章起草")
		if err != nil {
			return nil, err
		}
		ledger.Record("draft", "done", chapterPath)
		startStep = "review"
	} else {
		// 续跑时跳过起草，加载已有正文
		chapterPath, content = loadChapterFile(p, opts.Chapter)
	}

	// Step 2: 审查 + 润色 — 内置轻量 review pass，覆盖正文文件
	if startStep == "review" || startStep == "polish" {
		anchor := cb.BookContext(opts.Chapter, opts.Volume)
		emit("review", "审查并润色")
		outlineRef := snap.ChapterOutline
		if outlineRef == "" {
			outlineRef = snap.VolumeOutline
		}
		reviewed, err := w.Agent.Run(ctx, withUsage(agent.RunInput{
			SystemPrompt: prompts.ReviewSystem(anchor),
			UserPrompt: fmt.Sprintf(`【本章章纲】
%s

【Open 伏笔】
%s

【正文】
%s`, outlineRef, snap.OpenForeshadows, content),
		}))
		if err != nil {
			ledger.Record("review", "failed", err.Error())
			_ = ledger.Save(p.RunLedgerPath())
			return nil, err
		}

		// 保存完整审查报告到审查目录
		_ = os.MkdirAll(p.ReviewsDir(), 0o755)
		_ = os.WriteFile(p.ReviewPath(opts.Chapter), []byte(reviewed), 0o644)
		persistReviewRecord(st, opts.Chapter, p.ReviewPath(opts.Chapter), reviewed)

		// 提取润色版全文写回正文
		ledger.Record("review", "done", "ok")

		polished := extractPolishedBody(reviewed, content)
		title := pipeline.ParseChapterTitle(polished)
		chapterPath, err = pipeline.SaveChapterWithVersion(p, opts.Chapter, title, polished, version.SourceWriteReview, "审查润色")
		if err != nil {
			return nil, err
		}
		content = polished
		ledger.Record("polish", "done", chapterPath)
	}

	// Step 3: 摘要 — 供后续章节上下文链使用
	emit("summary", "生成章节摘要")
	summary, err := w.Agent.Run(ctx, withUsage(agent.RunInput{
		SystemPrompt: prompts.SummarySystem(),
		UserPrompt:   fmt.Sprintf("请为以下章节生成摘要：\n\n%s", content),
	}))
	if err != nil {
		return &report.Report{Stage: fmt.Sprintf("写章 第%d章", opts.Chapter), Status: report.StatusPartial,
			Summary: "正文已保存，摘要失败", Artifacts: []string{chapterPath}, Issues: []string{err.Error()}}, nil
	}
	_ = pipeline.SaveSummary(p, opts.Chapter, summary)
	ledger.Record("summary", "done", "ok")

	// Step 4: 写后沉淀 — 备份、更新进度、FTS 索引、实体/伏笔/记忆提取
	emit("extract", "沉淀记忆与索引")
	_ = pipeline.UpdateProjectProgress(p, opts.Chapter)
	_ = pipeline.PostWriteIndex(p, st, opts.Chapter, chapterPath)
	if err := ExtractAndPersistFacts(ctx, w.Agent, st, opts.Chapter, content, summary); err != nil {
		ledger.Record("extract", "failed", err.Error())
	} else {
		ledger.Record("extract", "done", "ok")
	}
	ledger.Record("commit", "done", "ok")
	_ = ledger.Save(p.RunLedgerPath())
	emit("done", "写章完成")
	tu := usageAcc.Snapshot()
	tokenUsage := &report.TokenUsage{
		PromptTokens:     tu.PromptTokens,
		CompletionTokens: tu.CompletionTokens,
		TotalTokens:      tu.Total(),
		EstimatedCostUSD: usage.EstimateCostUSD(w.Agent.Model(), tu.PromptTokens, tu.CompletionTokens),
	}
	_, _ = usage.AddWriteRun(p.Root, tu.PromptTokens, tu.CompletionTokens)
	return &report.Report{
		Stage: fmt.Sprintf("写章 第%d章", opts.Chapter), Status: report.StatusDone,
		Summary: fmt.Sprintf("第 %d 章已完成，约 %d 字", opts.Chapter, utf8.RuneCountInString(content)),
		Artifacts: []string{chapterPath, p.SummaryPath(opts.Chapter)},
		NextSteps: []string{fmt.Sprintf("nova review %d", opts.Chapter), fmt.Sprintf("nova write %d", opts.Chapter+1)},
		TokenUsage: tokenUsage,
	}, nil
}

func loadChapterFile(p *project.Project, chapter int) (path, content string) {
	path, _, err := p.FindChapterFile(chapter)
	if err != nil {
		return "", ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return path, ""
	}
	return path, string(data)
}

func extractPolishedBody(reviewed, fallback string) string {
	markers := []string{"润色版全文", "润色版正文", "润色版", "修订版全文", "修订后正文", "## 润色版正文", "## 润色", "## 润色后正文"}
	for _, m := range markers {
		if idx := strings.Index(reviewed, m); idx >= 0 {
			body := strings.TrimSpace(reviewed[idx+len(m):])
			body = strings.TrimLeft(body, "：:\n")
			if utf8.RuneCountInString(body) > 200 {
				return stripReviewMetricsSuffix(body)
			}
		}
	}
	if idx := findChapterHeadingStart(reviewed); idx >= 0 {
		body := strings.TrimSpace(reviewed[idx:])
		if utf8.RuneCountInString(body) > 200 {
			return stripReviewMetricsSuffix(body)
		}
	}
	if parts := strings.Split(reviewed, "\n---\n"); len(parts) >= 2 {
		last := strings.TrimSpace(parts[len(parts)-1])
		if utf8.RuneCountInString(last) > 200 {
			return stripReviewMetricsSuffix(last)
		}
	}
	return fallback
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
	lines := strings.Split(s, "\n")
	pos := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, "章") {
			return pos
		}
		pos += len(line) + 1
	}
	return -1
}

type ReviewWorkflow struct {
	Agent *agent.Agent
}

func NewReviewWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *ReviewWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &ReviewWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func (w *ReviewWorkflow) ReviewChapter(ctx context.Context, p *project.Project, st *store.Store, chapter int) (*report.Report, error) {
	path, body := loadChapterFile(p, chapter)
	if path == "" {
		return &report.Report{Stage: "审查", Status: report.StatusFailed, Summary: fmt.Sprintf("第 %d 章正文不存在", chapter)}, nil
	}
	settings := readDirConcat(p.SettingsDir())
	anchor := prompts.BookContext{
		Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
		Protagonist: p.Meta.Protagonist, Cheat: p.Meta.Cheat, Synopsis: p.Meta.Synopsis,
		Chapter: chapter, Volume: p.Meta.CurrentVolume,
	}
	content, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.ReviewSystem(anchor),
		UserPrompt:   fmt.Sprintf("设定：\n%s\n\n正文：\n%s", settings, body),
		Tools:        true,
	})
	if err != nil {
		return nil, err
	}
	reviewPath := p.ReviewPath(chapter)
	_ = os.WriteFile(reviewPath, []byte(content), 0o644)
	hookScore := persistReviewRecord(st, chapter, reviewPath, content)
	_ = st.UpsertChapter(store.Chapter{Number: chapter, Status: "reviewed", UpdatedAt: project.Timestamp()})
	summaryPath := p.SummaryPath(chapter)
	summary, _ := os.ReadFile(summaryPath)
	_ = ExtractAndPersistFacts(ctx, w.Agent, st, chapter, body, string(summary))
	return &report.Report{
		Stage: fmt.Sprintf("审查 第%d章", chapter), Status: report.StatusDone,
		Summary: fmt.Sprintf("审查完成，追读力 %.1f/10", hookScore),
		Artifacts: []string{reviewPath},
		NextSteps: []string{"nova learn", fmt.Sprintf("nova query 第%d章", chapter)},
	}, nil
}

func parseReviewMetrics(content string) (float64, string, string) {
	jsonRaw, err := agent.ExtractJSONBlock(content)
	if err != nil {
		return 0, "", ""
	}
	var m struct {
		HookScore float64 `json:"hook_score"`
		CoolPoint string  `json:"cool_point"`
		Debt      string  `json:"debt"`
	}
	if err := json.Unmarshal([]byte(jsonRaw), &m); err != nil {
		return 0, "", ""
	}
	return m.HookScore, m.CoolPoint, m.Debt
}

func persistReviewRecord(st *store.Store, chapter int, reviewPath, content string) float64 {
	hookScore, coolPoint, debt := parseReviewMetrics(content)
	reportJSON, _ := json.Marshal(map[string]any{
		"hook_score": hookScore, "cool_point": coolPoint, "debt": debt,
	})
	_ = st.UpsertReview(store.ReviewRecord{
		ChapterNumber: chapter, HookScore: hookScore, CoolPoint: coolPoint, Debt: debt,
		ReportJSON: string(reportJSON), Path: reviewPath,
	})
	return hookScore
}

type LearnWorkflow struct {
	Agent *agent.Agent
}

func NewLearnWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *LearnWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &LearnWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func (w *LearnWorkflow) Learn(ctx context.Context, st *store.Store, content string, chapter int) (*report.Report, error) {
	raw, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.LearnSystem(),
		UserPrompt:   content,
	})
	if err != nil {
		return nil, err
	}
	jsonRaw, err := agent.ExtractJSONBlock(raw)
	if err != nil {
		return &report.Report{Stage: "学习", Status: report.StatusFailed, Summary: "无法解析 LLM 输出"}, nil
	}
	var item struct {
		Category string `json:"category"`
		Subject  string `json:"subject"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(jsonRaw), &item); err != nil {
		return nil, err
	}
	id := project.NewMemoryID()
	_, _ = st.UpsertMemory(store.Memory{
		ID: id, Category: item.Category, Subject: item.Subject, Content: item.Content,
		SourceChapter: chapter, Status: "active", CreatedAt: project.Timestamp(),
	})
	return &report.Report{
		Stage: "学习", Status: report.StatusDone,
		Summary: "写作模式已沉淀到长期记忆",
		Artifacts: []string{id},
	}, nil
}

type QueryResult struct {
	Entities    []store.Entity      `json:"entities,omitempty"`
	Foreshadows []store.Foreshadow    `json:"foreshadows,omitempty"`
	FTS         []map[string]string `json:"fts,omitempty"`
	Memories    []store.Memory      `json:"memories,omitempty"`
	CoolPoints  []store.CoolPoint   `json:"cool_points,omitempty"`
}

func Query(st *store.Store, keyword, foreshadowStatus string) (*QueryResult, error) {
	res := &QueryResult{}
	if keyword != "" {
		if keyword == "伏笔" || stringsContains(keyword, "伏笔") {
			items, _ := st.ListForeshadows(foreshadowStatus)
			res.Foreshadows = items
		}
		if keyword == "爽点" || stringsContains(keyword, "爽点") {
			items, _ := st.ListCoolPoints(0)
			res.CoolPoints = items
		}
		entities, _ := st.SearchEntities(keyword, 20)
		res.Entities = entities
		hits, _ := st.SearchFTS(keyword, 10)
		res.FTS = hits
		memories, _ := st.QueryMemories("", keyword, 10)
		res.Memories = memories
	} else if foreshadowStatus != "" {
		items, _ := st.ListForeshadows(foreshadowStatus)
		res.Foreshadows = items
	}
	return res, nil
}

func stringsContains(s, sub string) bool {
	return strings.Contains(s, sub)
}
