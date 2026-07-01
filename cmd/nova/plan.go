package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/version"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var planCmd = &cobra.Command{
	Use:   "plan [卷号]",
	Short: "生成卷纲与章纲",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPlan,
}

var planShowCmd = &cobra.Command{
	Use:   "show [卷号]",
	Short: "查看卷纲",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vols, err := project.ParseVolumeRange(args[0])
		if err != nil {
			return err
		}
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			return err
		}
		defer actx.Close()
		path := actx.Project.VolumeOutlinePath(vols[0])
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	},
}

var (
	replanFromChapter int
	replanApply       bool
	replanYes         bool
	replanNotes       string
)

var planReplanCmd = &cobra.Command{
	Use:   "replan [卷号]",
	Short: "基于已写内容重新规划卷纲",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanReplan,
}

func runPlanReplan(cmd *cobra.Command, args []string) error {
	vols, err := project.ParseVolumeRange(args[0])
	if err != nil {
		return err
	}
	if len(vols) != 1 {
		return fmt.Errorf("replan 一次只能指定一个卷号")
	}
	vol := vols[0]

	actx, err := app.LoadContext(projectRoot)
	if err != nil {
		return err
	}
	defer actx.Close()

	if err := app.RequireAPIKey(actx.Config); err != nil {
		return err
	}

	wf := workflows.NewPlanWorkflow(actx.Config, actx.Project, actx.Store)
	result, err := wf.ReplanVolume(context.Background(), actx.Project, actx.Store, workflows.ReplanOptions{
		Volume: vol, FromChapter: replanFromChapter, Notes: replanNotes,
	})
	if err != nil {
		return err
	}

	diff := version.DiffTexts("current", "proposed", "当前卷纲", "新卷纲", result.OldContent, result.ProposedContent)
	printOutlineDiff(diff)

	if err := result.Report.Print(outputFmt); err != nil {
		return err
	}

	if !replanApply {
		fmt.Println("\n预览模式：未写入文件。确认后执行：nova plan replan", vol, "--apply")
		return nil
	}

	if !replanYes {
		fmt.Print("\n确认应用新卷纲？[y/N] ")
		reader := bufio.NewReader(os.Stdin)
		confirm, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(confirm)) != "y" {
			fmt.Println("已取消")
			return nil
		}
	}

	path := actx.Project.VolumeOutlinePath(vol)
	if err := os.WriteFile(path, []byte(result.ProposedContent), 0o644); err != nil {
		return err
	}
	fmt.Printf("\n已应用新卷纲：%s\n", path)
	return nil
}

func printOutlineDiff(d version.DiffResult) {
	fmt.Printf("\n=== 卷纲 diff（+%d / -%d 字）===\n", d.AddedWords, d.RemovedWords)
	const maxLines = 80
	shown := 0
	for _, l := range d.Lines {
		if l.Type == "same" {
			continue
		}
		if shown >= maxLines {
			fmt.Println("…（diff 过长，已截断，请在 Studio 规划页查看完整对比）")
			break
		}
		switch l.Type {
		case "add":
			fmt.Printf("+ %s\n", l.Text)
		case "del":
			fmt.Printf("- %s\n", l.Text)
		}
		shown++
	}
}

func runPlan(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请指定卷号，如: nova plan 1")
	}
	vols, err := project.ParseVolumeRange(args[0])
	if err != nil {
		return err
	}

	// 加载项目上下文
	actx, err := app.LoadContext(projectRoot)
	if err != nil {
		return err
	}
	defer actx.Close()

	// 检查 API Key 是否配置
	if err := app.RequireAPIKey(actx.Config); err != nil {
		return err
	}

	// 调用 LLM 规划卷
	wf := workflows.NewPlanWorkflow(actx.Config, actx.Project, actx.Store)
	for _, vol := range vols {
		rep, err := wf.PlanVolume(context.Background(), actx.Project, vol)
		if err != nil {
			return err
		}
		if err := rep.Print(outputFmt); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planReplanCmd)
	planReplanCmd.Flags().IntVar(&replanFromChapter, "from-chapter", 0, "从第几章起重新规划（默认：已写章+1）")
	planReplanCmd.Flags().BoolVar(&replanApply, "apply", false, "确认后写入卷纲文件")
	planReplanCmd.Flags().BoolVarP(&replanYes, "yes", "y", false, "跳过确认提示（须与 --apply 同用）")
	planReplanCmd.Flags().StringVar(&replanNotes, "notes", "", "补充说明（可选）")
}
