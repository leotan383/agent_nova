package project

import (
	"os"
	"path/filepath"
	"strings"
)

// SettingChecklistItem 题材模板建议条目（与 Init 设定模板对齐）。
type SettingChecklistItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	CategoryID   string `json:"category_id"`
	TemplateKind string `json:"template_kind"`
	Done         bool   `json:"done"`
	SettingRel   string `json:"setting_rel,omitempty"`
}

func settingTemplateMeta(filename string) (categoryID, templateKind, title string) {
	title = strings.TrimSuffix(filename, ".md")
	switch filename {
	case "主角卡.md":
		return SettingsSubCharacter, "character", title
	case "反派设计.md":
		return SettingsSubCharacter, "villain", title
	case "世界观.md", "力量体系.md", "科技体系.md", "金手指.md":
		return "世界观", "default", title
	case "势力关系.md":
		return SettingsSubFaction, "default", title
	default:
		sub := SettingFileSubdir(filename)
		cat := SubdirToCategoryID(sub)
		if cat == "" {
			cat = SettingsSubOther
		}
		return cat, "blank", title
	}
}

// SettingChecklist 返回当前题材建议的设定模板完成度。
func (p *Project) SettingChecklist() ([]SettingChecklistItem, string, error) {
	genre := strings.TrimSpace(p.Meta.Genre)
	if genre == "" {
		genre = "玄幻"
	}
	files := DefaultSettingFiles(genre)
	var items []SettingChecklistItem
	for _, filename := range files {
		categoryID, templateKind, title := settingTemplateMeta(filename)
		if SidebarHiddenSettingCategory(categoryID) {
			continue
		}
		subdir, ok := ResolveCategorySubdir(categoryID)
		if !ok {
			subdir = SettingFileSubdir(filename)
		}
		rel := filepath.ToSlash(filepath.Join(subdir, filename))
		path := filepath.Join(p.SettingsDir(), filepath.FromSlash(rel))
		done := false
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			done = true
		}
		item := SettingChecklistItem{
			ID:           strings.TrimSuffix(filename, ".md"),
			Title:        title,
			CategoryID:   categoryID,
			TemplateKind: templateKind,
			Done:         done,
		}
		if done {
			item.SettingRel = rel
		}
		items = append(items, item)
	}
	return items, genre, nil
}
