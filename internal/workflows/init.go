package workflows

import (
	"context"
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

type InitWorkflow struct {
	Agent *agent.Agent
}

func NewInitWorkflow(cfg *config.Config, root string, st *store.Store) *InitWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(root, st)
	return &InitWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func (w *InitWorkflow) EnrichSettings(ctx context.Context, p *project.Project) (*report.Report, error) {
	var settings []string
	_ = filepath.Walk(p.SettingsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, err := filepath.Rel(p.Root, path)
		if err != nil {
			return err
		}
		settings = append(settings, filepath.ToSlash(rel))
		return nil
	})
	userPrompt := fmt.Sprintf(`请完善以下设定文件内容（保留 Markdown 结构）：
书名：%s
题材：%s
主角：%s
金手指：%s
基调：%s
文件：%s
请分别输出每个文件的完整 Markdown，用 ===FILE:路径=== 分隔。`,
		p.Meta.Title, p.Meta.Genre, p.Meta.Protagonist, p.Meta.Cheat, p.Meta.WritingStyle(), strings.Join(settings, ", "))
	content, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.InitSystem(p.Meta.Genre),
		UserPrompt:   userPrompt,
	})
	if err != nil {
		return nil, err
	}
	w.writeSplitFiles(p.Root, content)
	masterOutline, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.InitSystem(p.Meta.Genre),
		UserPrompt: fmt.Sprintf(`基于书名《%s》题材%s，生成总纲 Markdown（含分卷规划表）。`, p.Meta.Title, p.Meta.Genre),
	})
	if err != nil {
		return &report.Report{
			Stage: "初始化", Status: report.StatusPartial, Summary: "设定已生成，总纲生成失败",
			Issues: []string{err.Error()}, NextSteps: []string{"手动编辑 大纲/总纲.md"},
		}, nil
	}
	_ = os.WriteFile(fmt.Sprintf("%s/大纲/总纲.md", p.Root), []byte(masterOutline), 0o644)
	return &report.Report{
		Stage: "初始化", Status: report.StatusDone,
		Summary: fmt.Sprintf("项目《%s》初始化完成", p.Meta.Title),
		Artifacts: []string{"设定集/", "大纲/总纲.md", "nova.yaml"},
		NextSteps: []string{"nova plan 1", "nova status"},
	}, nil
}

func (w *InitWorkflow) writeSplitFiles(root, content string) {
	parts := strings.Split(content, "===FILE:")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lines := strings.SplitN(part, "===", 2)
		if len(lines) < 2 {
			continue
		}
		path := strings.TrimSpace(lines[0])
		body := strings.TrimSpace(lines[1])
		full := fmt.Sprintf("%s/%s", root, path)
		_ = os.WriteFile(full, []byte(body), 0o644)
	}
}

type PlanWorkflow struct {
	Agent *agent.Agent
}

func NewPlanWorkflow(cfg *config.Config, p *project.Project, st *store.Store) *PlanWorkflow {
	reg := tools.NewRegistry()
	reg.BindProject(p.Root, st)
	return &PlanWorkflow{Agent: agent.New(agent.Options{Config: cfg, Registry: reg})}
}

func (w *PlanWorkflow) PlanVolume(ctx context.Context, p *project.Project, vol int) (*report.Report, error) {
	master, _ := os.ReadFile(fmt.Sprintf("%s/大纲/总纲.md", p.Root))
	settings := readDirConcat(p.SettingsDir())
	userPrompt := fmt.Sprintf(`请为第 %d 卷生成详细卷纲 Markdown。
每章格式：
### 第N章 · 标题
- 核心冲突：
- 爽点：
- 伏笔：

总纲：
%s

设定摘要：
%s`, vol, string(master), settings)
	content, err := w.Agent.Run(ctx, agent.RunInput{
		SystemPrompt: prompts.PlanSystem(prompts.BookContext{
			Title: p.Meta.Title, Genre: p.Meta.Genre, Style: p.Meta.WritingStyle(),
			Protagonist: p.Meta.Protagonist, Cheat: p.Meta.Cheat, Synopsis: p.Meta.Synopsis,
			Volume: vol,
		}),
		UserPrompt:   userPrompt,
		Tools:        true,
	})
	if err != nil {
		return nil, err
	}
	path := p.VolumeOutlinePath(vol)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	if p.Meta.CurrentVolume < vol {
		p.Meta.CurrentVolume = vol
	}
	if p.Meta.Phase == project.PhaseInitDone {
		p.Meta.Phase = project.PhasePlanning
	}
	_ = p.Save()
	return &report.Report{
		Stage: fmt.Sprintf("卷纲规划 第%d卷", vol), Status: report.StatusDone,
		Summary: fmt.Sprintf("第 %d 卷纲已生成", vol),
		Artifacts: []string{path},
		NextSteps: []string{fmt.Sprintf("nova write %d", (vol-1)*30+1), "nova plan show " + fmt.Sprint(vol)},
	}, nil
}

func readDirConcat(dir string) string {
	var parts []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, _ := os.ReadFile(path)
		rel, _ := filepath.Rel(dir, path)
		parts = append(parts, fmt.Sprintf("### %s\n%s", filepath.ToSlash(rel), string(data)))
		return nil
	})
	return strings.Join(parts, "\n---\n")
}
