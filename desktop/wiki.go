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

// CreateWikiSettingInput 新建设定文档。
type CreateWikiSettingInput struct {
	Category     string `json:"category"`      // 角色|背景|势力|地点|物品|其他
	Title        string `json:"title"`         // 文件名（不含 .md）
	TemplateKind string `json:"template_kind"` // character|villain|blank
}

// CreateWikiSetting 在设定集对应子目录创建 Markdown 并返回新条目。
func (a *App) CreateWikiSetting(in CreateWikiSettingInput) (WikiContentDTO, error) {
	subdir, ok := categoryToSubdir[in.Category]
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

var categoryToSubdir = map[string]string{
	"角色": project.SettingsSubCharacter,
	"背景": project.SettingsSubWorld,
	"势力": project.SettingsSubFaction,
	"地点": project.SettingsSubLocation,
	"物品": project.SettingsSubItem,
	"其他": project.SettingsSubOther,
}
