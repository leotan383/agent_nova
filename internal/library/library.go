package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/tanlian/agent_nova/internal/paths"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/status"
	"github.com/tanlian/agent_nova/internal/store"
)

const fileName = "library.json"

// Entry 书库索引项（真源仍在项目目录 nova.yaml）。
type Entry struct {
	ID           string    `json:"id"`
	Path         string    `json:"path"`
	Pinned       bool      `json:"pinned"`
	Archived     bool      `json:"archived"`
	LastOpenedAt time.Time `json:"last_opened_at"`
}

// Registry 多小说书库。
type Registry struct {
	ActiveID string  `json:"active_id"`
	Novels   []Entry `json:"novels"`
}

// NovelCard 供 UI 展示的书籍卡片。
type NovelCard struct {
	ID             string `json:"id"`
	Path           string `json:"path"`
	Title          string `json:"title"`
	Genre          string `json:"genre"`
	Phase          string `json:"phase"`
	CurrentVolume  int    `json:"current_volume"`
	CurrentChapter int    `json:"current_chapter"`
	ChapterCount     int     `json:"chapter_count"`
	WrittenWords     int     `json:"written_words"`
	TargetWords      int     `json:"target_words"`
	ProgressPercent  float64 `json:"progress_percent"`
	Pinned           bool    `json:"pinned"`
	Archived       bool   `json:"archived"`
	LastOpenedAt   string `json:"last_opened_at"`
	Missing        bool   `json:"missing"`
}

func registryPath() string {
	return filepath.Join(paths.Global().ConfigDir, fileName)
}

// Load 读取书库；不存在则返回空书库。
func Load() (*Registry, error) {
	path := registryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, err
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse library: %w", err)
	}
	return &reg, nil
}

// Save 持久化书库。
func (r *Registry) Save() error {
	dir := filepath.Dir(registryPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(registryPath(), data, 0o644)
}

func (r *Registry) find(id string) (*Entry, int) {
	for i := range r.Novels {
		if r.Novels[i].ID == id {
			return &r.Novels[i], i
		}
	}
	return nil, -1
}

func (r *Registry) findByPath(abs string) (*Entry, int) {
	norm, err := filepath.Abs(abs)
	if err != nil {
		norm = abs
	}
	for i := range r.Novels {
		p, err := filepath.Abs(r.Novels[i].Path)
		if err != nil {
			p = r.Novels[i].Path
		}
		if p == norm {
			return &r.Novels[i], i
		}
	}
	return nil, -1
}

// Repair 规范化路径、去重，并修复失效的 active_id。
func (r *Registry) Repair() error {
	changed := false
	byPath := map[string]Entry{}
	for _, e := range r.Novels {
		abs, err := filepath.Abs(e.Path)
		if err != nil {
			abs = e.Path
		}
		e.Path = abs
		if prev, ok := byPath[abs]; !ok || e.LastOpenedAt.After(prev.LastOpenedAt) {
			byPath[abs] = e
		}
	}
	if len(byPath) != len(r.Novels) {
		changed = true
	}
	r.Novels = make([]Entry, 0, len(byPath))
	for _, e := range byPath {
		r.Novels = append(r.Novels, e)
	}
	if r.ActiveID != "" {
		if e, _ := r.find(r.ActiveID); e == nil {
			r.ActiveID = ""
			changed = true
		}
	}
	if r.ActiveID == "" && len(r.Novels) > 0 {
		sort.Slice(r.Novels, func(i, j int) bool {
			return r.Novels[i].LastOpenedAt.After(r.Novels[j].LastOpenedAt)
		})
		r.ActiveID = r.Novels[0].ID
		changed = true
	}
	if !changed {
		return nil
	}
	return r.Save()
}
func (r *Registry) Register(root string) (*Entry, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(abs, project.MetaFile)); err != nil {
		return nil, fmt.Errorf("未找到 nova.yaml: %s", abs)
	}
	if e, _ := r.findByPath(abs); e != nil {
		e.LastOpenedAt = time.Now().UTC()
		r.ActiveID = e.ID
		return e, r.Save()
	}
	e := Entry{
		ID:           uuid.New().String(),
		Path:         abs,
		LastOpenedAt: time.Now().UTC(),
	}
	r.Novels = append(r.Novels, e)
	r.ActiveID = e.ID
	if err := r.Save(); err != nil {
		return nil, err
	}
	_ = project.SetCurrentProject(abs)
	return &e, nil
}

// SetActive 切换当前小说；已是当前书则直接成功。
func (r *Registry) SetActive(id string) error {
	if id == "" {
		return fmt.Errorf("小说 ID 不能为空")
	}
	if r.ActiveID == id {
		if e, _ := r.find(id); e != nil {
			return nil
		}
	}
	e, _ := r.find(id)
	if e == nil {
		return fmt.Errorf("书库中未找到小说: %s", id)
	}
	if _, err := os.Stat(filepath.Join(e.Path, project.MetaFile)); err != nil {
		return fmt.Errorf("项目路径无效: %s", e.Path)
	}
	r.ActiveID = id
	e.LastOpenedAt = time.Now().UTC()
	if err := r.Save(); err != nil {
		return err
	}
	return project.SetCurrentProject(e.Path)
}

// Remove 从书库移除（不删除磁盘文件）。
func (r *Registry) Remove(id string) error {
	idx := -1
	for i := range r.Novels {
		if r.Novels[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("书库中未找到小说: %s", id)
	}
	r.Novels = append(r.Novels[:idx], r.Novels[idx+1:]...)
	if r.ActiveID == id {
		r.ActiveID = ""
		if len(r.Novels) > 0 {
			r.ActiveID = r.Novels[0].ID
		}
	}
	return r.Save()
}

// SetPinned 置顶。
func (r *Registry) SetPinned(id string, pinned bool) error {
	e, _ := r.find(id)
	if e == nil {
		return fmt.Errorf("书库中未找到小说: %s", id)
	}
	e.Pinned = pinned
	return r.Save()
}

// SetArchived 归档。
func (r *Registry) SetArchived(id string, archived bool) error {
	e, _ := r.find(id)
	if e == nil {
		return fmt.Errorf("书库中未找到小说: %s", id)
	}
	e.Archived = archived
	return r.Save()
}

// ActivePath 当前激活小说路径。
func (r *Registry) ActivePath() string {
	if r.ActiveID == "" {
		return ""
	}
	e, _ := r.find(r.ActiveID)
	if e == nil {
		return ""
	}
	return e.Path
}

// ListCards 列出书库卡片（含元数据与章节数）。
func (r *Registry) ListCards(includeArchived bool) []NovelCard {
	out := make([]NovelCard, 0, len(r.Novels))
	for _, e := range r.Novels {
		if e.Archived && !includeArchived {
			continue
		}
		out = append(out, BuildCard(e))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		return out[i].LastOpenedAt > out[j].LastOpenedAt
	})
	return out
}

func BuildCard(e Entry) NovelCard {
	card := NovelCard{
		ID:           e.ID,
		Path:         e.Path,
		Pinned:       e.Pinned,
		Archived:     e.Archived,
		LastOpenedAt: e.LastOpenedAt.Format(time.RFC3339),
	}
	if _, err := os.Stat(filepath.Join(e.Path, project.MetaFile)); err != nil {
		card.Missing = true
		card.Title = filepath.Base(e.Path)
		return card
	}
	p, err := project.Load(e.Path)
	if err != nil {
		card.Missing = true
		card.Title = filepath.Base(e.Path)
		return card
	}
	card.Title = p.Meta.Title
	card.Genre = p.Meta.Genre
	card.Phase = p.Meta.Phase
	card.CurrentVolume = p.Meta.CurrentVolume
	card.CurrentChapter = p.Meta.CurrentChapter
	if st, err := store.Open(p.DBPath()); err == nil {
		if chs, err := st.ListChapters(); err == nil {
			card.ChapterCount = len(chs)
			prog := status.ComputeProgress(p.Meta, chs)
			card.WrittenWords = prog.WrittenWords
			card.TargetWords = prog.TargetWords
			card.ProgressPercent = prog.ProgressPercent
		}
		_ = st.Close()
	}
	return card
}

// SyncCurrentFromCLI 若书库为空，尝试从 ~/.config/nova/current 导入。
func SyncCurrentFromCLI(reg *Registry) error {
	if len(reg.Novels) > 0 {
		return nil
	}
	cur, err := project.CurrentProjectRoot()
	if err != nil || cur == "" {
		return nil
	}
	_, err = reg.Register(cur)
	return err
}
