package main

import (
	"fmt"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/wiki"
)

// WikiEntryDTO 百科目录条目。
type WikiEntryDTO struct {
	ID       string `json:"id"`
	Group    string `json:"group"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Kind     string `json:"kind"`
	Path     string `json:"path,omitempty"`
}

// WikiContentDTO 百科正文。
type WikiContentDTO struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Body     string `json:"body"`
	Path     string `json:"path,omitempty"`
	CanOpen  bool   `json:"can_open"`
	Editable bool   `json:"editable"`
}

func toWikiEntryDTOs(entries []wiki.Entry) []WikiEntryDTO {
	out := make([]WikiEntryDTO, len(entries))
	for i, e := range entries {
		out[i] = WikiEntryDTO{
			ID: e.ID, Group: e.Group, Title: e.Title,
			Subtitle: e.Subtitle, Kind: e.Kind, Path: e.Path,
		}
	}
	return out
}

func toWikiContentDTO(c wiki.Content) WikiContentDTO {
	return WikiContentDTO{
		ID: c.ID, Title: c.Title, Group: c.Group, Kind: c.Kind,
		Body: c.Body, Path: c.Path, CanOpen: c.CanOpen, Editable: c.Editable,
	}
}

// ListWikiEntries 返回人物/设定/大纲百科目录。
func (a *App) ListWikiEntries() (out []WikiEntryDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		entries, listErr := wiki.List(actx.Project, actx.Store)
		if listErr != nil {
			return listErr
		}
		out = toWikiEntryDTOs(entries)
		return nil
	})
	return out, err
}

// GetWikiContent 读取百科条目正文。
func (a *App) GetWikiContent(id string) (WikiContentDTO, error) {
	if id == "" {
		return WikiContentDTO{}, fmt.Errorf("请选择条目")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return WikiContentDTO{}, err
	}
	var result WikiContentDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		content, getErr := wiki.Get(actx.Project, actx.Store, id)
		if getErr != nil {
			return getErr
		}
		result = toWikiContentDTO(content)
		return nil
	})
	return result, err
}

// SaveWikiContent 保存百科条目（设定集/大纲/记忆）。
func (a *App) SaveWikiContent(id, body string) error {
	if id == "" {
		return fmt.Errorf("请选择条目")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return wiki.Save(actx.Project, actx.Store, id, body)
	})
}

// SettingCategoryDTO 设定集分类。
type SettingCategoryDTO struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Subdir  string `json:"subdir"`
	Builtin bool   `json:"builtin"`
}

func toSettingCategoryDTOs(cats []project.SettingCategoryInfo) []SettingCategoryDTO {
	out := make([]SettingCategoryDTO, len(cats))
	for i, c := range cats {
		out[i] = SettingCategoryDTO{
			ID: c.ID, Label: c.Label, Subdir: c.Subdir, Builtin: c.Builtin,
		}
	}
	return out
}

// ListSettingCategories 返回当前项目的设定分类（内置 + 用户自定义，不含「其他」）。
func (a *App) ListSettingCategories() (out []SettingCategoryDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		cats, listErr := actx.Project.ListSettingCategories()
		if listErr != nil {
			return listErr
		}
		out = toSettingCategoryDTOs(cats)
		return nil
	})
	return out, err
}

// CreateSettingCategory 新建用户自定义设定分类。
func (a *App) CreateSettingCategory(name string) (SettingCategoryDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return SettingCategoryDTO{}, err
	}
	var result SettingCategoryDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		cat, createErr := actx.Project.CreateSettingCategory(name)
		if createErr != nil {
			return createErr
		}
		result = SettingCategoryDTO{
			ID: cat.ID, Label: cat.Label, Subdir: cat.Subdir, Builtin: cat.Builtin,
		}
		return nil
	})
	return result, err
}

// RenameSettingCategory 重命名用户自定义设定分类。
func (a *App) RenameSettingCategory(categoryID, newName string) (SettingCategoryDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return SettingCategoryDTO{}, err
	}
	var result SettingCategoryDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		cat, renameErr := wiki.RenameSettingCategory(actx.Project, actx.Store, categoryID, newName)
		if renameErr != nil {
			return renameErr
		}
		result = SettingCategoryDTO{
			ID: cat.ID, Label: cat.Label, Subdir: cat.Subdir, Builtin: cat.Builtin,
		}
		return nil
	})
	return result, err
}

// DeleteSettingCategory 删除用户自定义设定分类及其下全部设定。
func (a *App) DeleteSettingCategory(categoryID string) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return wiki.DeleteSettingCategory(actx.Project, actx.Store, categoryID)
	})
}

// SettingChecklistItemDTO 题材模板 checklist 条目。
type SettingChecklistItemDTO struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	CategoryID   string `json:"category_id"`
	TemplateKind string `json:"template_kind"`
	Done         bool   `json:"done"`
	SettingRel   string `json:"setting_rel,omitempty"`
}

// SettingChecklistDTO 题材设定模板完成度。
type SettingChecklistDTO struct {
	Genre     string                    `json:"genre"`
	Items     []SettingChecklistItemDTO `json:"items"`
	DoneCount int                       `json:"done_count"`
	Total     int                       `json:"total"`
}

// GetSettingChecklist 返回当前题材建议的设定模板 checklist。
func (a *App) GetSettingChecklist() (SettingChecklistDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return SettingChecklistDTO{}, err
	}
	var result SettingChecklistDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		items, genre, listErr := actx.Project.SettingChecklist()
		if listErr != nil {
			return listErr
		}
		result.Genre = genre
		result.Total = len(items)
		for _, it := range items {
			if it.Done {
				result.DoneCount++
			}
			result.Items = append(result.Items, SettingChecklistItemDTO{
				ID: it.ID, Title: it.Title, CategoryID: it.CategoryID,
				TemplateKind: it.TemplateKind, Done: it.Done, SettingRel: it.SettingRel,
			})
		}
		return nil
	})
	return result, err
}

// SaveSettingCategoryOrder 保存设定分类排序（内置 + 自定义）。
func (a *App) SaveSettingCategoryOrder(order []string) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return actx.Project.SaveSettingCategoryOrderValidated(order)
	})
}

// CreateWikiSettingInput 新建设定文档。
type CreateWikiSettingInput struct {
	Category     string `json:"category"`      // 分类 id（内置或自定义）
	Title        string `json:"title"`         // 文件名（不含 .md）
	TemplateKind string `json:"template_kind"` // character|villain|blank
}

// CreateWikiSetting 在设定集对应子目录创建 Markdown 并返回新条目。
func (a *App) CreateWikiSetting(in CreateWikiSettingInput) (WikiContentDTO, error) {
	subdir, ok := resolveCategorySubdir(in.Category)
	if !ok || in.Category == "" {
		return WikiContentDTO{}, fmt.Errorf("无效设定分类")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return WikiContentDTO{}, err
	}
	var result WikiContentDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		content, createErr := wiki.CreateSetting(actx.Project, actx.Store, subdir, in.Title, in.TemplateKind)
		if createErr != nil {
			return createErr
		}
		result = toWikiContentDTO(content)
		return nil
	})
	return result, err
}

func resolveCategorySubdir(categoryID string) (string, bool) {
	if subdir, ok := project.ResolveCategorySubdir(categoryID); ok {
		return subdir, true
	}
	if categoryID == project.SettingsSubOther {
		return project.SettingsSubOther, true
	}
	return "", false
}
