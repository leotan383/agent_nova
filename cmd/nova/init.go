package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	memorypkg "github.com/tanlian/agent_nova/internal/memory"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/workflows"
)

var (
	initDir          string
	initGenre        string
	initTitle        string
	initStyle        string
	initTargetWords  int
	initChapterWords int
	initSynopsis     string
	initTone         string
	initProtagonist  string
	initCheat        string
	initInteractive  bool
	initDiscover     bool
	initSkipLLM      bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化新书项目",
	RunE: func(cmd *cobra.Command, args []string) error {
		if initDir == "" {
			return fmt.Errorf("请指定 --dir")
		}
		if initDiscover && initSkipLLM {
			return fmt.Errorf("--discover 需要调用 LLM，不能与 --skip-llm 同时使用")
		}
		if initDiscover && initInteractive {
			return fmt.Errorf("--discover 与 --interactive 二选一")
		}

		var (
			discoverResult workflows.DiscoverResult
			discoverNotes  string
		)

		in := project.InitInput{
			Dir: initDir, Title: initTitle, Genre: initGenre,
			Style: initStyle, TargetWords: initTargetWords, ChapterWords: initChapterWords,
			Synopsis: initSynopsis, Tone: initTone,
			Protagonist: initProtagonist, Cheat: initCheat, Interactive: initInteractive, SkipLLM: initSkipLLM,
		}

		if initDiscover {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if err := app.RequireAPIKey(cfg); err != nil {
				return fmt.Errorf("构思探讨需要 API Key: %w", err)
			}
			fmt.Println("=== 构思探讨 ===")
			fmt.Println("和 AI 聊聊你想写的故事；聊够了输入 /done 生成立项。")
			fmt.Println()
			var dr workflows.DiscoverResult
			var transcript string
			in, dr, transcript, err = workflows.RunDiscoverySession(context.Background(), cfg, initGenre)
			if err != nil {
				return err
			}
			discoverResult = dr
			discoverNotes = transcript
			in.Dir = initDir
			in.SkipLLM = initSkipLLM
		} else if initInteractive {
			in = runInteractiveInit(in)
		}

		if !initDiscover && strings.TrimSpace(in.Title) == "" {
			return fmt.Errorf("请指定书名 --title")
		}
		if in.Title == "" {
			in.Title = "未命名"
		}
		if in.Genre == "" {
			in.Genre = "玄幻"
		}
		if in.Style == "" && in.Tone != "" {
			in.Style = in.Tone
		}
		if in.Style == "" {
			in.Style = "热血"
		}
		if in.TargetWords <= 0 {
			in.TargetWords = project.DefaultTargetWords
		}
		if in.ChapterWords <= 0 {
			in.ChapterWords = project.DefaultChapterWords
		}

		// 初始化数据库、创建表结构
		dbPath := fmt.Sprintf("%s/.nova/nova.db", initDir)
		st, err := store.Open(dbPath)
		if err != nil {
			return err
		}
		defer st.Close()

		// 初始化项目
		res, err := project.InitProject(in)
		if err != nil {
			return err
		}

		if initDiscover {
			p := &project.Project{Root: res.Root, Meta: res.Meta}
			if err := workflows.SaveDiscoveryNotes(p, discoverResult, discoverNotes); err != nil {
				fmt.Println("警告: 保存探讨记录失败:", err)
			}
		}

		// 把项目元数据插入到数据库
		_ = st.InitProject(res.Root, res.Meta)

		// 设置当前项目，把当前项目路径写入到 ~/.config/nova/current 文件
		_ = project.SetCurrentProject(res.Root)

		// 如果跳过 LLM，则直接返回
		if initSkipLLM {
			fmt.Printf("项目已创建于 %s\n", res.Root)
			return nil
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// 检查 API Key 是否配置
		if err := app.RequireAPIKey(cfg); err != nil {
			fmt.Println("提示:", err)
			fmt.Printf("项目骨架已创建于 %s，配置 API Key 后可完善设定\n", res.Root)
			return nil
		}

		// 加载项目上下文
		actx, err := app.LoadContext(res.Root)
		if err != nil {
			return err
		}
		defer actx.Close()

		// 调用 LLM 完善设定
		wf := workflows.NewInitWorkflow(actx.Config, res.Root, actx.Store)
		rep, err := wf.EnrichSettings(context.Background(), actx.Project)
		if err != nil {
			return err
		}

		// 回填记忆
		n, _ := memorypkg.BootstrapFromSettings(actx.Project, actx.Store)
		if n > 0 {
			rep.Artifacts = append(rep.Artifacts, fmt.Sprintf("记忆回填 %d 条", n))
		}
		return rep.Print(outputFmt)
	},
}

func runInteractiveInit(in project.InitInput) project.InitInput {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("交互式立项（直接回车可跳过该项，稍后再补）")
	ask := func(prompt, current string) string {
		if current != "" {
			fmt.Printf("%s [%s]: ", prompt, current)
		} else {
			fmt.Printf("%s (可留空): ", prompt)
		}
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return current
		}
		return line
	}
	in.Title = ask("书名", in.Title)
	in.Genre = ask("题材", in.Genre)
	in.Style = ask("写作风格", in.Style)
	in.Synopsis = ask("故事简介", in.Synopsis)
	in.Protagonist = ask("主角", in.Protagonist)
	in.Cheat = ask("金手指", in.Cheat)
	if in.TargetWords <= 0 {
		line := ask("目标字数(10/20/30/50/100 万)", "30")
		in.TargetWords = parseTargetWords(line)
	}
	if in.ChapterWords <= 0 {
		line := ask("每章字数(3000/4000/5000)", "4000")
		in.ChapterWords = parseChapterWords(line)
	}
	return in
}

func parseTargetWords(s string) int {
	s = strings.TrimSpace(s)
	switch s {
	case "10", "10万", "100000":
		return 100000
	case "20", "20万", "200000":
		return 200000
	case "50", "50万", "500000":
		return 500000
	case "100", "100万", "1000000":
		return 1000000
	default:
		return project.DefaultTargetWords
	}
}

func parseChapterWords(s string) int {
	s = strings.TrimSpace(s)
	switch s {
	case "3000", "3k":
		return 3000
	case "5000", "5k":
		return 5000
	default:
		return project.DefaultChapterWords
	}
}

func init() {
	initCmd.Flags().StringVar(&initDir, "dir", "", "项目目录")
	initCmd.Flags().StringVar(&initGenre, "genre", "玄幻", "题材")
	initCmd.Flags().StringVar(&initTitle, "title", "", "书名（必填）")
	initCmd.Flags().StringVar(&initStyle, "style", "热血", "写作风格")
	initCmd.Flags().IntVar(&initTargetWords, "target-words", project.DefaultTargetWords, "目标总字数")
	initCmd.Flags().IntVar(&initChapterWords, "chapter-words", project.DefaultChapterWords, "每章目标字数")
	initCmd.Flags().StringVar(&initSynopsis, "synopsis", "", "故事简介")
	initCmd.Flags().StringVar(&initTone, "tone", "", "基调（兼容旧参数，等同 --style）")
	initCmd.Flags().StringVar(&initProtagonist, "protagonist", "", "主角")
	initCmd.Flags().StringVar(&initCheat, "cheat", "", "金手指")
	initCmd.Flags().BoolVar(&initInteractive, "interactive", false, "交互式问答（表单式补全字段）")
	initCmd.Flags().BoolVar(&initDiscover, "discover", false, "与 AI 探讨构思后再立项（适合只有模糊想法时）")
	initCmd.Flags().BoolVar(&initSkipLLM, "skip-llm", false, "仅创建骨架，不调用 LLM")
}
