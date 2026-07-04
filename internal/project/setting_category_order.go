package project

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type settingCategoryOrderFile struct {
	Order       []string `json:"order,omitempty"`
	CustomOrder []string `json:"custom_order,omitempty"` // legacy: 仅自定义分类排序
}

func (p *Project) settingCategoryOrderPath() string {
	return filepath.Join(p.NovaDir(), "setting_category_order.json")
}

func (p *Project) readCategoryOrderFile() (settingCategoryOrderFile, error) {
	data, err := os.ReadFile(p.settingCategoryOrderPath())
	if err != nil {
		if os.IsNotExist(err) {
			return settingCategoryOrderFile{}, nil
		}
		return settingCategoryOrderFile{}, err
	}
	var f settingCategoryOrderFile
	if err := json.Unmarshal(data, &f); err != nil {
		return settingCategoryOrderFile{}, err
	}
	return f, nil
}

// LoadCategoryOrder 读取分类排序。legacyCustomOnly 为 true 时表示旧版仅含自定义分类 id。
func (p *Project) LoadCategoryOrder() (order []string, legacyCustomOnly bool, err error) {
	f, err := p.readCategoryOrderFile()
	if err != nil {
		return nil, false, err
	}
	if len(f.Order) > 0 {
		return f.Order, false, nil
	}
	if len(f.CustomOrder) > 0 {
		return f.CustomOrder, true, nil
	}
	return nil, false, nil
}

// SaveCategoryOrder 保存完整分类排序（内置 + 自定义）。
func (p *Project) SaveCategoryOrder(order []string) error {
	if err := os.MkdirAll(p.NovaDir(), 0o755); err != nil {
		return err
	}
	f := settingCategoryOrderFile{Order: order}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.settingCategoryOrderPath(), data, 0o644)
}

func saveLegacyCustomCategoryOrder(path string, order []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f := settingCategoryOrderFile{CustomOrder: order}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func applyCategoryOrder(categories []SettingCategoryInfo, order []string) []SettingCategoryInfo {
	if len(categories) == 0 || len(order) == 0 {
		return categories
	}
	byID := map[string]SettingCategoryInfo{}
	for _, c := range categories {
		byID[c.ID] = c
	}
	seen := map[string]struct{}{}
	var out []SettingCategoryInfo
	for _, id := range order {
		if c, ok := byID[id]; ok {
			out = append(out, c)
			seen[id] = struct{}{}
		}
	}
	for _, c := range categories {
		if _, ok := seen[c.ID]; !ok {
			out = append(out, c)
		}
	}
	return out
}

func applyLegacyCustomCategoryOrder(all []SettingCategoryInfo, customOrder []string) []SettingCategoryInfo {
	var builtin, custom []SettingCategoryInfo
	for _, c := range all {
		if c.Builtin {
			builtin = append(builtin, c)
		} else {
			custom = append(custom, c)
		}
	}
	custom = applyCategoryOrder(custom, customOrder)
	return append(builtin, custom...)
}

func (p *Project) appendCategoryOrder(id string) error {
	order, legacy, err := p.LoadCategoryOrder()
	if err != nil {
		return err
	}
	if legacy {
		for _, existing := range order {
			if existing == id {
				return nil
			}
		}
		order = append(order, id)
		return saveLegacyCustomCategoryOrder(p.settingCategoryOrderPath(), order)
	}
	all, err := p.collectSettingCategories()
	if err != nil {
		return err
	}
	ids := categoryIDs(applyCategoryOrder(all, order))
	for _, existing := range ids {
		if existing == id {
			return nil
		}
	}
	ids = append(ids, id)
	return p.SaveCategoryOrder(ids)
}

func (p *Project) removeCategoryOrder(id string) error {
	order, legacy, err := p.LoadCategoryOrder()
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return nil
	}
	next := order[:0]
	for _, existing := range order {
		if existing != id {
			next = append(next, existing)
		}
	}
	if legacy {
		return saveLegacyCustomCategoryOrder(p.settingCategoryOrderPath(), next)
	}
	return p.SaveCategoryOrder(next)
}

func (p *Project) renameCategoryOrder(oldID, newID string) error {
	order, legacy, err := p.LoadCategoryOrder()
	if err != nil {
		return err
	}
	if len(order) == 0 {
		return nil
	}
	changed := false
	for i, existing := range order {
		if existing == oldID {
			order[i] = newID
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if legacy {
		return saveLegacyCustomCategoryOrder(p.settingCategoryOrderPath(), order)
	}
	return p.SaveCategoryOrder(order)
}

func categoryIDs(cats []SettingCategoryInfo) []string {
	ids := make([]string, len(cats))
	for i, c := range cats {
		ids[i] = c.ID
	}
	return ids
}
