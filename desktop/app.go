package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	goruntime "runtime"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/index"
	"github.com/tanlian/agent_nova/internal/library"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/status"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/version"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var errNoActiveProject = fmt.Errorf("请先在书库中选择或创建一本小说")

// App Wails 绑定层，供前端调用。
type App struct {
	ctx       context.Context
	write     writeManager
	coach     coachManager
	revise    reviseManager
	selection selectionManager
	discover  discoverManager
	session   projectSession
}

func NewApp() *App {
	a := &App{}
	a.write = *newWriteManager(a)
	a.coach = *newCoachManager(a)
	a.revise = *newReviseManager(a)
	a.selection = *newSelectionManager(a)
	a.discover = *newDiscoverManager(a)
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ListNovels 返回书库卡片；includeArchived 是否包含归档。
func (a *App) ListNovels(includeArchived bool) ([]library.NovelCard, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	return reg.ListCards(includeArchived), nil
}

// GetActiveNovel 当前激活小说 ID 与路径。
func (a *App) GetActiveNovel() (map[string]string, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	if reg.ActiveID == "" {
		return map[string]string{}, nil
	}
	return map[string]string{
		"id":   reg.ActiveID,
		"path": reg.ActivePath(),
	}, nil
}

// SwitchNovel 切换当前工作小说。
func (a *App) SwitchNovel(id string) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	if err := reg.SetActive(id); err != nil {
		// 前端可能持有已删除或过期 ID，自愈后重试一次
		_ = reg.Repair()
		if err2 := reg.SetActive(id); err2 != nil {
			return fmt.Errorf("%w（请返回书库刷新列表）", err)
		}
	}
	a.session.invalidate()
	return nil
}

// RegisterNovel 打开已有 nova 项目目录并加入书库。
func (a *App) RegisterNovel(path string) (library.NovelCard, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return library.NovelCard{}, err
	}
	e, err := reg.Register(path)
	if err != nil {
		return library.NovelCard{}, err
	}
	return library.BuildCard(*e), nil
}

// PickNovelDirectory 打开系统文件夹选择对话框。
func (a *App) PickNovelDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择 nova 小说项目目录（含 nova.yaml）",
	})
}

// RemoveFromLibrary 从书库移除（不删文件）。
func (a *App) RemoveFromLibrary(id string) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return reg.Remove(id)
}

// SetNovelArchived 归档或取消归档。
func (a *App) SetNovelArchived(id string, archived bool) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return reg.SetArchived(id, archived)
}

// SetNovelPinned 置顶或取消置顶。
func (a *App) SetNovelPinned(id string, pinned bool) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return reg.SetPinned(id, pinned)
}

// CreateNovelInput 快速新建小说（非 discover）。
type CreateNovelInput struct {
	Dir          string `json:"dir"`
	Title        string `json:"title"`
	Genre        string `json:"genre"`
	Style        string `json:"style"`
	TargetWords  int    `json:"target_words"`
	ChapterWords int    `json:"chapter_words"`
	Synopsis     string `json:"synopsis"`
	Tone         string `json:"tone"`
	Protagonist  string `json:"protagonist"`
	Cheat        string `json:"cheat"`
}

// CreateNovel 初始化新书并加入书库。
func (a *App) CreateNovel(in CreateNovelInput) (library.NovelCard, error) {
	if in.Dir == "" {
		return library.NovelCard{}, fmt.Errorf("请指定项目目录")
	}
	if strings.TrimSpace(in.Title) == "" {
		return library.NovelCard{}, fmt.Errorf("书名不能为空")
	}
	if in.Genre == "" {
		in.Genre = "玄幻"
	}
	if in.Style == "" && in.Tone != "" {
		in.Style = in.Tone
	}
	if in.TargetWords <= 0 {
		in.TargetWords = project.DefaultTargetWords
	}
	if in.ChapterWords <= 0 {
		in.ChapterWords = project.DefaultChapterWords
	}
	res, err := project.InitProject(project.InitInput{
		Dir: in.Dir, Title: strings.TrimSpace(in.Title), Genre: in.Genre,
		Style: in.Style, TargetWords: in.TargetWords, ChapterWords: in.ChapterWords,
		Synopsis: in.Synopsis, Tone: in.Tone, Protagonist: in.Protagonist, Cheat: in.Cheat,
	})
	if err != nil {
		return library.NovelCard{}, err
	}
	st, err := store.Open(filepath.Join(res.Root, ".nova", "nova.db"))
	if err == nil {
		_ = st.InitProject(res.Root, res.Meta)
		_ = st.Close()
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return library.NovelCard{}, err
	}
	e, err := reg.Register(res.Root)
	if err != nil {
		return library.NovelCard{}, err
	}
	a.session.invalidate()
	return library.BuildCard(*e), nil
}

// PickCreateDirectory 选择新书保存目录。
func (a *App) PickCreateDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择新书项目保存位置",
	})
}

// RevealInFolder 在系统文件管理器中显示路径。
func (a *App) RevealInFolder(path string) error {
	if path == "" {
		return fmt.Errorf("path required")
	}
	switch goruntime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// GetStatus 当前小说创作状态报告。
func (a *App) GetStatus() (rep status.Report, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return status.Report{}, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := a.syncChaptersFromDisk(actx); err != nil {
			return err
		}
		rep = status.Build(actx.Project, actx.Store, "all")
		return nil
	})
	return rep, err
}

// ChapterDTO 章节摘要。
type ChapterDTO struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	WordCount int    `json:"word_count"`
	Status    string `json:"status"`
}

// ListChapters 列出当前小说章节（自动与正文目录同步）。
func (a *App) ListChapters() (out []ChapterDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := a.syncChaptersFromDisk(actx); err != nil {
			return err
		}
		chs, err := actx.Store.ListChapters()
		if err != nil {
			return err
		}
		out = make([]ChapterDTO, len(chs))
		for i, c := range chs {
			out[i] = ChapterDTO{
				Number: c.Number, Title: c.Title, WordCount: c.WordCount, Status: c.Status,
			}
		}
		return nil
	})
	return out, err
}

// GetChapterContent 读取章节正文。
func (a *App) GetChapterContent(number int) (content string, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return "", err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := a.syncChaptersFromDisk(actx); err != nil {
			return err
		}
		// 始终以正文目录最新文件为准，避免 DB 中 path 过期导致读到旧/不完整内容
		path, _, err := actx.Project.FindChapterFile(number)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content = string(data)
		return nil
	})
	return content, err
}

// SaveChapterContent 保存章节正文（手动编辑，自动留版本快照）。
func (a *App) SaveChapterContent(number int, content string) error {
	if number <= 0 {
		return fmt.Errorf("无效章号")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("正文不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		title := pipeline.ParseChapterTitle(content)
		if title == "" {
			if ch, err := actx.Store.GetChapter(number); err == nil {
				title = ch.Title
			}
		}
		path, err := pipeline.SaveChapterWithVersion(
			actx.Project, number, title, content, version.SourceManualEdit, "手动编辑",
		)
		if err != nil {
			return err
		}
		return pipeline.PostWriteIndex(actx.Project, actx.Store, number, path)
	})
}

func (a *App) syncChaptersFromDisk(actx *app.Context) error {
	chs, err := actx.Store.ListChapters()
	if err != nil {
		return err
	}
	needsSync := len(chs) == 0
	for _, c := range chs {
		if c.Path == "" || c.WordCount == 0 {
			needsSync = true
			break
		}
	}
	if !needsSync {
		entries, err := os.ReadDir(actx.Project.ChaptersDir())
		if err == nil {
			onDisk := 0
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					onDisk++
				}
			}
			if onDisk > len(chs) {
				needsSync = true
			}
		}
	}
	if !needsSync {
		return nil
	}
	idx := index.New(actx.Project, actx.Store)
	return idx.RebuildChapters(0)
}

// MemoryDTO 长期记忆条目。
type MemoryDTO struct {
	ID            string `json:"id"`
	Category      string `json:"category"`
	Subject       string `json:"subject"`
	Content       string `json:"content"`
	SourceChapter int    `json:"source_chapter"`
	Status        string `json:"status"`
}

// ForeshadowDTO 伏笔条目。
type ForeshadowDTO struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	PlantedChapter  int    `json:"planted_chapter"`
	ResolvedChapter int    `json:"resolved_chapter"`
	Status          string `json:"status"`
}

// ListMemories 列出长期记忆。
func (a *App) ListMemories() (out []MemoryDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		items, err := actx.Store.QueryMemories("", "", 200)
		if err != nil {
			return err
		}
		out = make([]MemoryDTO, len(items))
		for i, m := range items {
			out[i] = MemoryDTO{
				ID: m.ID, Category: m.Category, Subject: m.Subject, Content: m.Content,
				SourceChapter: m.SourceChapter, Status: m.Status,
			}
		}
		return nil
	})
	return out, err
}

// ListForeshadows 列出伏笔；status 传 open/resolved 或空表示全部。
func (a *App) ListForeshadows(status string) (out []ForeshadowDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		items, err := actx.Store.ListForeshadows(status)
		if err != nil {
			return err
		}
		out = make([]ForeshadowDTO, len(items))
		for i, f := range items {
			out[i] = ForeshadowDTO{
				ID: f.ID, Description: f.Description, PlantedChapter: f.PlantedChapter,
				ResolvedChapter: f.ResolvedChapter, Status: f.Status,
			}
		}
		return nil
	})
	return out, err
}

// HasAPIKey 是否已配置 LLM API。
func (a *App) HasAPIKey() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	return cfg.OpenAIAPIKey != ""
}

// AppInfo 应用信息。
func (a *App) AppInfo() map[string]string {
	return map[string]string{
		"name":    "Nova Studio",
		"version": "0.1.0",
	}
}

func (a *App) loadRegistry() (*library.Registry, error) {
	reg, err := library.Load()
	if err != nil {
		return nil, err
	}
	_ = library.SyncCurrentFromCLI(reg)
	_ = reg.Repair()
	return reg, nil
}

func (a *App) activeProjectRoot() (string, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return "", err
	}
	root := reg.ActivePath()
	if root == "" {
		return "", errNoActiveProject
	}
	return root, nil
}
