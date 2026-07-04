package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	PhaseEmpty    = "empty"
	PhaseInitDone = "init_done"
	PhasePlanning = "planning"
	PhaseWriting  = "writing"
	PhasePaused   = "paused"

	MetaFile = "nova.yaml"
	NovaDir  = ".nova"
)

type Meta struct {
	Title          string `yaml:"title"`           // 书名
	Genre          string `yaml:"genre"`           // 题材（玄幻/都市/科幻等）
	Phase          string `yaml:"phase"`           // 创作阶段：empty|init_done|planning|writing|paused
	CurrentVolume  int    `yaml:"current_volume"`  // 当前进行到的卷号
	CurrentChapter int    `yaml:"current_chapter"` // 当前已写到的章号
	Tone           string `yaml:"tone,omitempty"`  // 叙事基调（兼容旧项目）
	Style          string `yaml:"style,omitempty"` // 写作风格（热血/爽文等）
	TargetWords    int    `yaml:"target_words,omitempty"`  // 目标总字数
	ChapterWords   int    `yaml:"chapter_words,omitempty"` // 每章目标字数
	Synopsis       string `yaml:"synopsis,omitempty"`      // 故事简介
	Protagonist    string `yaml:"protagonist,omitempty"` // 主角名或简介
	Cheat          string `yaml:"cheat,omitempty"`       // 金手指设定摘要
	DailyWords     int    `yaml:"daily_words,omitempty"`     // 每日字数目标（0=按 daily_chapters×chapter_words）
	DailyChapters  int    `yaml:"daily_chapters,omitempty"`  // 每日章数目标
	BufferTarget   int    `yaml:"buffer_target,omitempty"`   // 存稿缓冲目标章数
}

// DailyWordsGoal 返回每日字数目标。
func (m Meta) DailyWordsGoal() int {
	if m.DailyWords > 0 {
		return m.DailyWords
	}
	ch := m.DailyChapters
	if ch <= 0 {
		ch = 1
	}
	cw := m.ChapterWords
	if cw <= 0 {
		cw = DefaultChapterWords
	}
	return ch * cw
}

// DailyChaptersGoal 返回每日章数目标。
func (m Meta) DailyChaptersGoal() int {
	if m.DailyChapters > 0 {
		return m.DailyChapters
	}
	return 1
}

// BufferTargetChapters 返回存稿缓冲目标。
func (m Meta) BufferTargetChapters() int {
	if m.BufferTarget > 0 {
		return m.BufferTarget
	}
	return 7
}

// WritingStyle 返回写作风格，兼容旧字段 tone。
func (m Meta) WritingStyle() string {
	if m.Style != "" {
		return m.Style
	}
	return m.Tone
}

type Project struct {
	Root string
	Meta Meta
}

func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, MetaFile)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("未找到 nova.yaml，请先 nova init 或在项目目录内运行")
		}
		dir = parent
	}
}

func Load(root string) (*Project, error) {
	data, err := os.ReadFile(filepath.Join(root, MetaFile))
	if err != nil {
		return nil, err
	}
	var meta Meta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &Project{Root: root, Meta: meta}, nil
}

func (p *Project) Save() error {
	data, err := yaml.Marshal(p.Meta)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.Root, MetaFile), data, 0o644)
}

func (p *Project) NovaDir() string     { return filepath.Join(p.Root, NovaDir) }
func (p *Project) DBPath() string      { return filepath.Join(p.NovaDir(), "nova.db") }
func (p *Project) BackupDir() string   { return filepath.Join(p.NovaDir(), "backups") }
func (p *Project) RunLedgerPath() string { return filepath.Join(p.NovaDir(), "run_ledger.json") }

func (p *Project) SettingsDir() string { return filepath.Join(p.Root, "设定集") }
func (p *Project) OutlineDir() string  { return filepath.Join(p.Root, "大纲") }
func (p *Project) ChaptersDir() string { return filepath.Join(p.Root, "正文") }
func (p *Project) ReviewsDir() string  { return filepath.Join(p.Root, "审查") }
func (p *Project) SummariesDir() string { return filepath.Join(p.Root, "摘要") }

func (p *Project) VolumeOutlinePath(vol int) string {
	return filepath.Join(p.OutlineDir(), fmt.Sprintf("第%02d卷.md", vol))
}

func ChapterFileName(num int, title string) string {
	if title == "" {
		return fmt.Sprintf("第%03d章.md", num)
	}
	return fmt.Sprintf("第%03d章-%s.md", num, sanitizeTitle(title))
}

func (p *Project) ChapterPath(num int, title string) string {
	return filepath.Join(p.ChaptersDir(), ChapterFileName(num, title))
}

func (p *Project) ReviewPath(num int) string {
	return filepath.Join(p.ReviewsDir(), fmt.Sprintf("第%03d章.review.md", num))
}

func (p *Project) SummaryPath(num int) string {
	return filepath.Join(p.SummariesDir(), fmt.Sprintf("第%03d章.summary.md", num))
}

func sanitizeTitle(s string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	return replacer.Replace(strings.TrimSpace(s))
}

// FindChapterFile 在正文目录中查找章节文件（同章多文件时取最新/最大）。
func (p *Project) FindChapterFile(number int) (path, title string, err error) {
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return "", "", err
	}
	prefix := fmt.Sprintf("第%03d", number)
	var bestPath string
	var bestSize int64
	var bestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(p.ChaptersDir(), e.Name())
		if bestPath == "" || info.ModTime().After(bestMod) ||
			(info.ModTime().Equal(bestMod) && info.Size() > bestSize) {
			bestPath = full
			bestSize = info.Size()
			bestMod = info.ModTime()
		}
	}
	if bestPath == "" {
		return "", "", fmt.Errorf("第 %d 章正文不存在", number)
	}
	base := filepath.Base(bestPath)
	base = strings.TrimSuffix(base, ".md")
	if i := strings.Index(base, "-"); i > 0 {
		title = base[i+1:]
	}
	return bestPath, title, nil
}

func ParseChapterRange(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("章节范围不能为空")
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("无效章节号: %s", s)
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("无效章节号: %s", s)
		}
		if start > end {
			start, end = end, start
		}
		var out []int
		for i := start; i <= end; i++ {
			out = append(out, i)
		}
		return out, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("无效章节号: %s", s)
	}
	return []int{n}, nil
}

func ParseVolumeRange(s string) ([]int, error) {
	return ParseChapterRange(s)
}

func ResolveProjectRoot(cwd string) (string, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return FindRoot(cwd)
}
