package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type InitInput struct {
	Dir          string // 项目根目录绝对或相对路径
	Title        string // 书名，写入 nova.yaml 与设定模板
	Genre        string // 题材，决定设定集模板与 InitAgent 提示词
	Style        string // 写作风格（热血/爽文等）
	TargetWords  int    // 目标总字数
	ChapterWords int    // 每章目标字数
	Synopsis     string // 故事简介
	Tone         string // 兼容旧 CLI；未设 Style 时写入 tone
	Protagonist  string // 主角名或简介，预填主角卡
	Cheat        string // 金手指设定，预填金手指.md
	Interactive  bool   // 是否由 CLI 层做交互式问答（本结构体仅传参）
	SkipLLM      bool   // 是否跳过 LLM 完善设定（由 CLI 层处理）
}

const (
	DefaultTargetWords  = 300000
	DefaultChapterWords = 4000
)

type InitResult struct {
	Root string // 项目的根目录绝对路径
	Meta Meta // 项目元数据
}

var settingTemplates = map[string][]string{
	"玄幻": {"世界观.md", "力量体系.md", "主角卡.md", "金手指.md", "反派设计.md"},
	"都市": {"世界观.md", "主角卡.md", "金手指.md", "势力关系.md"},
	"科幻": {"世界观.md", "科技体系.md", "主角卡.md", "势力关系.md"},
}

func DefaultSettingFiles(genre string) []string {
	if files, ok := settingTemplates[genre]; ok {
		return files
	}
	return settingTemplates["玄幻"]
}

func InitProject(in InitInput) (*InitResult, error) {
	root, err := filepath.Abs(in.Dir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, MetaFile)); err == nil {
		return nil, fmt.Errorf("项目已存在: %s", root)
	}

	// 创建项目目录结构
	dirs := []string{
		filepath.Join(root, NovaDir, "backups"),
		filepath.Join(root, "设定集"),
		filepath.Join(root, "大纲"),
		filepath.Join(root, "正文"),
		filepath.Join(root, "审查"),
		filepath.Join(root, "摘要"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}

	// 保存项目元数据
	style := strings.TrimSpace(in.Style)
	if style == "" {
		style = strings.TrimSpace(in.Tone)
	}
	targetWords := in.TargetWords
	if targetWords <= 0 {
		targetWords = DefaultTargetWords
	}
	chapterWords := in.ChapterWords
	if chapterWords <= 0 {
		chapterWords = DefaultChapterWords
	}
	meta := Meta{
		Title:          in.Title,
		Genre:          in.Genre,
		Phase:          PhaseInitDone,
		Style:          style,
		Tone:           style,
		TargetWords:    targetWords,
		ChapterWords:   chapterWords,
		Synopsis:       strings.TrimSpace(in.Synopsis),
		Protagonist:    in.Protagonist,
		Cheat:          in.Cheat,
	}
	p := &Project{Root: root, Meta: meta}
	if err := p.Save(); err != nil {
		return nil, err
	}

	// 创建设定集模板文件
	for _, f := range DefaultSettingFiles(in.Genre) {
		path := filepath.Join(p.SettingsDir(), f)
		if err := os.WriteFile(path, settingTemplateContent(f, meta), 0o644); err != nil {
			return nil, err
		}
	}

	// 创建总纲模板文件
	outlineFiles := map[string]string{
		"总纲.md":   outlineMasterTemplate(meta),
		"爽点规划.md": outlineCoolPointsTemplate(meta),
	}
	for name, content := range outlineFiles {
		if err := os.WriteFile(filepath.Join(p.OutlineDir(), name), []byte(content), 0o644); err != nil {
			return nil, err
		}
	}
	return &InitResult{Root: root, Meta: meta}, nil
}

func settingTemplateContent(name string, meta Meta) []byte {
	header := fmt.Sprintf("# %s\n\n> 书名：%s | 题材：%s | 风格：%s\n\n",
		strings.TrimSuffix(name, ".md"), meta.Title, meta.Genre, meta.WritingStyle())
	switch name {
	case "主角卡.md":
		body := fmt.Sprintf("## 姓名\n%s\n\n## 性格\n\n## 背景\n\n## 目标\n\n## 成长弧线\n", meta.Protagonist)
		return []byte(header + body)
	case "金手指.md":
		body := fmt.Sprintf("## 能力\n%s\n\n## 限制\n\n## 升级路线\n", meta.Cheat)
		return []byte(header + body)
	default:
		return []byte(header + "## 待补充\n")
	}
}

func outlineMasterTemplate(meta Meta) string {
	synopsis := meta.Synopsis
	if synopsis == "" {
		synopsis = "（待补充）"
	}
	return fmt.Sprintf(`# %s - 总纲

## 一句话梗概
%s

## 核心冲突

## 主线目标

## 创作目标
- 目标总字数：%s
- 单章字数：约 %d 字
- 写作风格：%s

## 分卷规划

| 卷 | 主题 | 核心事件 | 预计章数 |
|----|------|----------|----------|
| 1  |      |          | 30-50    |

## 基调
%s
`, meta.Title, synopsis, formatWordCount(meta.TargetWords), meta.ChapterWords, meta.WritingStyle(), meta.WritingStyle())
}

func formatWordCount(n int) string {
	if n >= 10000 && n%10000 == 0 {
		return fmt.Sprintf("%d 万字", n/10000)
	}
	if n >= 10000 {
		return fmt.Sprintf("%.1f 万字", float64(n)/10000)
	}
	return fmt.Sprintf("%d 字", n)
}

func outlineCoolPointsTemplate(meta Meta) string {
	return `# 爽点规划

## 大爽点（卷级）

## 中爽点（10章周期）

## 微爽点（章级）

## 追读力设计
- 章末钩子
- 悬念债务
- 微兑现节奏
`
}

func SetCurrentProject(root string) error {
	layout := filepath.Join(os.Getenv("HOME"), ".config", "nova")
	if home := os.Getenv("NOVA_HOME"); home != "" {
		layout = filepath.Join(home, "config")
	}
	if err := os.MkdirAll(layout, 0o755); err != nil {
		return err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(layout, "current"), []byte(abs+"\n"), 0o644)
}

func CurrentProjectRoot() (string, error) {
	layout := filepath.Join(os.Getenv("HOME"), ".config", "nova", "current")
	if home := os.Getenv("NOVA_HOME"); home != "" {
		layout = filepath.Join(home, "config", "current")
	}
	data, err := os.ReadFile(layout)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func NewMemoryID() string {
	return uuid.New().String()
}

func Timestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}
