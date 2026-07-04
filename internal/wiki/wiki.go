package wiki

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

const (
	KindSetting = "setting"
	KindMemory  = "memory"
	KindEntity  = "entity"
	KindOutline = "outline"

	GroupCharacter = "人物"
	GroupSetting   = "设定"
	GroupOutline   = "大纲"
)

// Entry 百科目录条目。
type Entry struct {
	ID       string `json:"id"`
	Group    string `json:"group"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Kind     string `json:"kind"`
	Path     string `json:"path,omitempty"`
}

// Content 百科正文。
type Content struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Group    string `json:"group"`
	Kind     string `json:"kind"`
	Body     string `json:"body"`
	Path     string `json:"path,omitempty"`
	CanOpen  bool   `json:"can_open"`
	Editable bool   `json:"editable"`
}

// List 汇总设定集、大纲、实体与相关记忆。
func List(p *project.Project, st *store.Store) ([]Entry, error) {
	_, _ = p.MigrateFlatSettings()

	var entries []Entry

	settingsDir := p.SettingsDir()
	_ = filepath.Walk(settingsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, err := filepath.Rel(settingsDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		title := strings.TrimSuffix(filepath.Base(path), ".md")
		entries = append(entries, Entry{
			ID:       settingID(rel),
			Group:    project.ClassifySettingRel(rel, title),
			Title:    title,
			Subtitle: project.SettingSubtitle(rel),
			Kind:     KindSetting,
			Path:     path,
		})
		return nil
	})

	outlineDir := p.OutlineDir()
	if outlines, err := os.ReadDir(outlineDir); err == nil {
		for _, e := range outlines {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			title := strings.TrimSuffix(e.Name(), ".md")
			entries = append(entries, Entry{
				ID:       outlineID(e.Name()),
				Group:    GroupOutline,
				Title:    title,
				Subtitle: "大纲",
				Kind:     KindOutline,
				Path:     filepath.Join(outlineDir, e.Name()),
			})
		}
	}

	if st != nil {
		if _, err := st.MergeDuplicateEntities(); err != nil {
			return nil, err
		}
		entities, err := st.ListEntities("", 500)
		if err == nil {
			seen := map[string]struct{}{}
			for _, e := range entities {
				typ, canonical := store.EntityCanonicalKey(e)
				if canonical == "" {
					continue
				}
				dedupeKey := typ + "\x00" + canonical
				if _, ok := seen[dedupeKey]; ok {
					continue
				}
				seen[dedupeKey] = struct{}{}

				group := GroupSetting
				subtitle := entityTypeLabel(typ)
				if typ == "character" {
					group = GroupCharacter
				}
				entries = append(entries, Entry{
					ID: settingEntityID(store.EntityID(typ, canonical)), Group: group, Title: canonical,
					Subtitle: subtitle, Kind: KindEntity,
				})
			}
		}

		memories, err := st.QueryMemories("", "", 200)
		if err == nil {
			for _, m := range memories {
				if m.Status != "" && m.Status != "active" {
					continue
				}
				group := memoryGroup(m.Category)
				if group == "" {
					continue
				}
				title := m.Subject
				if title == "" {
					title = memoryCategoryLabel(m.Category)
				}
				subtitle := memoryCategoryLabel(m.Category)
				if m.SourceChapter > 0 {
					subtitle = fmt.Sprintf("%s · 第%d章", subtitle, m.SourceChapter)
				}
				entries = append(entries, Entry{
					ID: memoryID(m.ID), Group: group, Title: title,
					Subtitle: subtitle, Kind: KindMemory,
				})
			}
		}
	}

	sortEntries(entries)
	return entries, nil
}

// Get 读取百科正文。
func Get(p *project.Project, st *store.Store, id string) (Content, error) {
	kind, key, err := parseID(id)
	if err != nil {
		return Content{}, err
	}

	switch kind {
	case KindSetting:
		if err := project.ValidateSettingRelPath(key); err != nil {
			return Content{}, err
		}
		path := filepath.Join(p.SettingsDir(), filepath.FromSlash(key))
		body, err := os.ReadFile(path)
		if err != nil {
			return Content{}, err
		}
		title := strings.TrimSuffix(filepath.Base(key), ".md")
		return Content{
			ID: id, Title: title, Group: project.ClassifySettingRel(key, title),
			Kind: KindSetting, Body: string(body), Path: path, CanOpen: true, Editable: true,
		}, nil
	case KindOutline:
		path := filepath.Join(p.OutlineDir(), key)
		body, err := os.ReadFile(path)
		if err != nil {
			return Content{}, err
		}
		title := strings.TrimSuffix(key, ".md")
		return Content{
			ID: id, Title: title, Group: GroupOutline,
			Kind: KindOutline, Body: string(body), Path: path, CanOpen: true, Editable: true,
		}, nil
	case KindMemory:
		if st == nil {
			return Content{}, fmt.Errorf("记忆不可用")
		}
		items, err := st.QueryMemories("", "", 1000)
		if err != nil {
			return Content{}, err
		}
		for _, m := range items {
			if m.ID != key {
				continue
			}
			title := m.Subject
			if title == "" {
				title = memoryCategoryLabel(m.Category)
			}
			return Content{
				ID: id, Title: title, Group: memoryGroup(m.Category),
				Kind: KindMemory, Body: m.Content, Editable: true,
			}, nil
		}
		return Content{}, fmt.Errorf("记忆不存在")
	case KindEntity:
		if st == nil {
			return Content{}, fmt.Errorf("实体不可用")
		}
		e, err := st.FindEntity(key)
		if err != nil {
			return Content{}, err
		}
		typ, canonical := store.EntityCanonicalKey(*e)
		group := GroupSetting
		if typ == "character" {
			group = GroupCharacter
		}
		return Content{
			ID: id, Title: canonical, Group: group, Kind: KindEntity,
			Body: formatEntityBody(*e), Editable: false,
		}, nil
	default:
		return Content{}, fmt.Errorf("未知条目: %s", id)
	}
}

func settingID(relPath string) string { return KindSetting + ":" + filepath.ToSlash(relPath) }
func outlineID(filename string) string { return KindOutline + ":" + filename }
func memoryID(id string) string        { return KindMemory + ":" + id }
func settingEntityID(id string) string { return KindEntity + ":" + id }

func parseID(id string) (kind, key string, err error) {
	i := strings.Index(id, ":")
	if i <= 0 {
		return "", "", fmt.Errorf("无效条目 ID: %s", id)
	}
	return id[:i], id[i+1:], nil
}

func memoryGroup(category string) string {
	switch category {
	case "character":
		return GroupCharacter
	case "world", "plot", "style":
		return GroupSetting
	default:
		return ""
	}
}

func memoryCategoryLabel(category string) string {
	switch category {
	case "character":
		return "角色记忆"
	case "world":
		return "世界观"
	case "plot":
		return "剧情记忆"
	case "style":
		return "写法记忆"
	default:
		return category
	}
}

func entityTypeLabel(t string) string {
	switch t {
	case "character":
		return "AI提取"
	case "location":
		return "AI提取"
	case "item":
		return "AI提取"
	default:
		return t
	}
}

// Save 保存可编辑百科条目。
func Save(p *project.Project, st *store.Store, id, body string) error {
	kind, key, err := parseID(id)
	if err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("内容不能为空")
	}
	switch kind {
	case KindSetting:
		if err := project.ValidateSettingRelPath(key); err != nil {
			return err
		}
		path := filepath.Join(p.SettingsDir(), filepath.FromSlash(key))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(body), 0o644)
	case KindOutline:
		if err := validateMarkdownFilename(key); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(p.OutlineDir(), key), []byte(body), 0o644)
	case KindMemory:
		if st == nil {
			return fmt.Errorf("记忆不可用")
		}
		return st.UpdateMemoryContent(key, body)
	default:
		return fmt.Errorf("该条目不可编辑")
	}
}

func validateMarkdownFilename(name string) error {
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("无效文件名")
	}
	if !strings.HasSuffix(name, ".md") {
		return fmt.Errorf("仅支持 Markdown 文件")
	}
	return nil
}

func formatEntityBody(e store.Entity) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", e.Name))
	b.WriteString(fmt.Sprintf("类型：%s\n", entityTypeLabel(e.Type)))
	if e.LastChapter > 0 {
		b.WriteString(fmt.Sprintf("最近更新：第 %d 章\n\n", e.LastChapter))
	} else {
		b.WriteString("\n")
	}
	if e.StateJSON == "" || e.StateJSON == "{}" {
		b.WriteString("暂无结构化状态。")
		return b.String()
	}
	var state any
	if err := json.Unmarshal([]byte(e.StateJSON), &state); err != nil {
		b.WriteString(e.StateJSON)
		return b.String()
	}
	pretty, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		b.WriteString(e.StateJSON)
		return b.String()
	}
	b.WriteString("## 状态\n\n")
	b.Write(pretty)
	return b.String()
}

func sortEntries(entries []Entry) {
	groupOrder := map[string]int{
		GroupCharacter: 0,
		GroupSetting:   1,
		GroupOutline:   2,
	}
	kindOrder := map[string]int{
		KindSetting: 0,
		KindOutline: 1,
		KindEntity:  2,
		KindMemory:  3,
	}
	sort.Slice(entries, func(i, j int) bool {
		gi, gj := groupOrder[entries[i].Group], groupOrder[entries[j].Group]
		if gi != gj {
			return gi < gj
		}
		ki, kj := kindOrder[entries[i].Kind], kindOrder[entries[j].Kind]
		if ki != kj {
			return ki < kj
		}
		return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
	})
}

// Excerpt 取正文摘要（用于列表预览，可选）。
func Excerpt(body string, maxRunes int) string {
	body = strings.TrimSpace(body)
	if maxRunes <= 0 || utf8.RuneCountInString(body) <= maxRunes {
		return body
	}
	runes := []rune(body)
	return string(runes[:maxRunes]) + "…"
}
