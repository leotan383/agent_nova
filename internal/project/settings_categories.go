package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SettingCategoryInfo 设定集分类（内置或用户自定义）。
type SettingCategoryInfo struct {
	ID      string // 稳定标识，用于 API / UI
	Label   string // 展示名
	Subdir  string // 设定集子目录名
	Builtin bool
}

var builtinSettingCategories = []SettingCategoryInfo{
	{ID: SettingsSubCharacter, Label: SettingsSubCharacter, Subdir: SettingsSubCharacter, Builtin: true},
	{ID: "世界观", Label: "世界观", Subdir: SettingsSubWorld, Builtin: true},
	{ID: SettingsSubFaction, Label: SettingsSubFaction, Subdir: SettingsSubFaction, Builtin: true},
	{ID: SettingsSubLocation, Label: SettingsSubLocation, Subdir: SettingsSubLocation, Builtin: true},
	{ID: SettingsSubItem, Label: SettingsSubItem, Subdir: SettingsSubItem, Builtin: true},
}

var sidebarHiddenSettingCategoryIDs = map[string]struct{}{
	SettingsSubOther: {},
}

// SidebarHiddenSettingCategory 判断分类是否不在侧边栏展示。
func SidebarHiddenSettingCategory(categoryID string) bool {
	_, ok := sidebarHiddenSettingCategoryIDs[categoryID]
	return ok
}

// SanitizeCategoryName 清理用户新建的分类名（同时作为目录名与展示名）。
func SanitizeCategoryName(name string) string {
	name = SanitizeSettingTitle(name)
	return strings.TrimSpace(name)
}

// ResolveCategorySubdir 将 UI/API 分类 id 映射为磁盘子目录。
func ResolveCategorySubdir(categoryID string) (string, bool) {
	categoryID = strings.TrimSpace(categoryID)
	for _, c := range builtinSettingCategories {
		if c.ID == categoryID {
			return c.Subdir, true
		}
	}
	if categoryID == "" {
		return "", false
	}
	if _, reserved := sidebarHiddenSettingCategoryIDs[categoryID]; reserved {
		return "", false
	}
	for _, c := range builtinSettingCategories {
		if c.Subdir == categoryID {
			return "", false
		}
	}
	return categoryID, true
}

func resolveBuiltinByID(categoryID string) (SettingCategoryInfo, bool) {
	for _, c := range builtinSettingCategories {
		if c.ID == categoryID {
			return c, true
		}
	}
	return SettingCategoryInfo{}, false
}

// SubdirToCategoryID 从设定集子目录反查分类 id。
func SubdirToCategoryID(subdir string) string {
	subdir = strings.TrimSpace(subdir)
	for _, c := range builtinSettingCategories {
		if c.Subdir == subdir {
			return c.ID
		}
	}
	if subdir == SettingsSubOther {
		return SettingsSubOther
	}
	if subdir != "" {
		return subdir
	}
	return ""
}

// ListSettingCategories 返回侧边栏可见的设定分类（不含「其他」）。
func (p *Project) ListSettingCategories() ([]SettingCategoryInfo, error) {
	all, err := p.collectSettingCategories()
	if err != nil {
		return nil, err
	}
	order, legacyCustomOnly, err := p.LoadCategoryOrder()
	if err != nil {
		return nil, err
	}
	if legacyCustomOnly {
		return applyLegacyCustomCategoryOrder(all, order), nil
	}
	if len(order) > 0 {
		return applyCategoryOrder(all, order), nil
	}
	return all, nil
}

func (p *Project) collectSettingCategories() ([]SettingCategoryInfo, error) {
	if err := p.EnsureSettingsSubdirs(); err != nil {
		return nil, err
	}
	var out []SettingCategoryInfo
	for _, c := range builtinSettingCategories {
		if SidebarHiddenSettingCategory(c.ID) {
			continue
		}
		out = append(out, c)
	}

	seen := map[string]struct{}{}
	for _, c := range builtinSettingCategories {
		seen[c.Subdir] = struct{}{}
	}
	for id := range sidebarHiddenSettingCategoryIDs {
		if id == SettingsSubOther {
			seen[SettingsSubOther] = struct{}{}
		}
		if c, ok := resolveBuiltinByID(id); ok {
			seen[c.Subdir] = struct{}{}
		}
	}

	entries, err := os.ReadDir(p.SettingsDir())
	if err != nil {
		return nil, err
	}
	var custom []SettingCategoryInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		custom = append(custom, SettingCategoryInfo{
			ID: name, Label: name, Subdir: name, Builtin: false,
		})
	}
	sort.Slice(custom, func(i, j int) bool {
		return custom[i].Label < custom[j].Label
	})
	out = append(out, custom...)
	return out, nil
}

// SaveSettingCategoryOrderValidated 校验并保存设定分类排序（内置 + 自定义）。
func (p *Project) SaveSettingCategoryOrderValidated(order []string) error {
	all, err := p.collectSettingCategories()
	if err != nil {
		return err
	}
	validIDs := map[string]struct{}{}
	for _, c := range all {
		validIDs[c.ID] = struct{}{}
	}
	clean := make([]string, 0, len(order))
	used := map[string]struct{}{}
	for _, id := range order {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := validIDs[id]; !ok {
			return fmt.Errorf("无效分类: %s", id)
		}
		if _, dup := used[id]; dup {
			continue
		}
		used[id] = struct{}{}
		clean = append(clean, id)
	}
	for _, c := range all {
		if _, ok := used[c.ID]; !ok {
			clean = append(clean, c.ID)
		}
	}
	return p.SaveCategoryOrder(clean)
}

// SaveCustomCategoryOrderValidated 兼容旧调用名。
func (p *Project) SaveCustomCategoryOrderValidated(order []string) error {
	return p.SaveSettingCategoryOrderValidated(order)
}

// CreateSettingCategory 新建用户自定义设定分类（创建子目录）。
func (p *Project) CreateSettingCategory(name string) (SettingCategoryInfo, error) {
	name = SanitizeCategoryName(name)
	if name == "" {
		return SettingCategoryInfo{}, fmt.Errorf("请填写分类名称")
	}
	if name == SettingsSubOther {
		return SettingCategoryInfo{}, fmt.Errorf("「其他」为系统保留分类")
	}
	for _, c := range builtinSettingCategories {
		if c.ID == name || c.Label == name || c.Subdir == name {
			return SettingCategoryInfo{}, fmt.Errorf("「%s」已存在", name)
		}
	}
	dir := p.SettingCategoryDir(name)
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return SettingCategoryInfo{}, fmt.Errorf("「%s」已存在", name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SettingCategoryInfo{}, err
	}
	_ = p.appendCategoryOrder(name)
	return SettingCategoryInfo{ID: name, Label: name, Subdir: name, Builtin: false}, nil
}

func (p *Project) customCategoryByID(categoryID string) (SettingCategoryInfo, error) {
	categoryID = strings.TrimSpace(categoryID)
	cats, err := p.ListSettingCategories()
	if err != nil {
		return SettingCategoryInfo{}, err
	}
	for _, c := range cats {
		if c.ID == categoryID {
			if c.Builtin {
				return SettingCategoryInfo{}, fmt.Errorf("内置分类不可修改")
			}
			return c, nil
		}
	}
	return SettingCategoryInfo{}, fmt.Errorf("分类不存在")
}

func validateNewCategoryName(p *Project, name, excludeID string) (string, error) {
	name = SanitizeCategoryName(name)
	if name == "" {
		return "", fmt.Errorf("请填写分类名称")
	}
	if name == excludeID {
		return name, nil
	}
	if name == SettingsSubOther {
		return "", fmt.Errorf("「其他」为系统保留分类")
	}
	for _, c := range builtinSettingCategories {
		if c.ID == name || c.Label == name || c.Subdir == name {
			return "", fmt.Errorf("「%s」已存在", name)
		}
	}
	cats, err := p.ListSettingCategories()
	if err != nil {
		return "", err
	}
	for _, c := range cats {
		if c.ID == name || c.Subdir == name {
			return "", fmt.Errorf("「%s」已存在", name)
		}
	}
	dir := p.SettingCategoryDir(name)
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return "", fmt.Errorf("「%s」已存在", name)
	}
	return name, nil
}

// WalkSettingMarkdownFiles 遍历某分类下全部设定 Markdown 相对路径。
func (p *Project) WalkSettingMarkdownFiles(subdir string, fn func(relPath, absPath string) error) error {
	dir := p.SettingCategoryDir(subdir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(subdir, e.Name()))
		abs := filepath.Join(dir, e.Name())
		if err := fn(rel, abs); err != nil {
			return err
		}
	}
	return nil
}

// RenameSettingCategory 重命名用户自定义分类目录。
func (p *Project) RenameSettingCategory(oldID, newName string) (SettingCategoryInfo, error) {
	cat, err := p.customCategoryByID(oldID)
	if err != nil {
		return SettingCategoryInfo{}, err
	}
	newName, err = validateNewCategoryName(p, newName, oldID)
	if err != nil {
		return SettingCategoryInfo{}, err
	}
	if newName == oldID {
		return cat, nil
	}
	oldDir := p.SettingCategoryDir(cat.Subdir)
	newDir := p.SettingCategoryDir(newName)
	if err := os.Rename(oldDir, newDir); err != nil {
		return SettingCategoryInfo{}, err
	}
	_ = p.renameCategoryOrder(oldID, newName)
	return SettingCategoryInfo{ID: newName, Label: newName, Subdir: newName, Builtin: false}, nil
}

// DeleteSettingCategory 删除用户自定义分类及其下全部设定文件，返回已删相对路径。
func (p *Project) DeleteSettingCategory(categoryID string) ([]string, error) {
	cat, err := p.customCategoryByID(categoryID)
	if err != nil {
		return nil, err
	}
	var deleted []string
	if err := p.WalkSettingMarkdownFiles(cat.Subdir, func(relPath, absPath string) error {
		if err := os.Remove(absPath); err != nil {
			return err
		}
		deleted = append(deleted, relPath)
		return nil
	}); err != nil {
		return deleted, err
	}
	dir := p.SettingCategoryDir(cat.Subdir)
	if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
		return deleted, err
	}
	_ = p.removeCategoryOrder(categoryID)
	return deleted, nil
}

// IsKnownSettingSubdir 判断子目录是否为有效设定分类目录。
func (p *Project) IsKnownSettingSubdir(subdir string) bool {
	subdir = strings.TrimSpace(subdir)
	if subdir == "" {
		return false
	}
	if subdir == SettingsSubOther {
		return true
	}
	cats, err := p.ListSettingCategories()
	if err != nil {
		return false
	}
	for _, c := range cats {
		if c.Subdir == subdir {
			return true
		}
	}
	path := filepath.Join(p.SettingsDir(), subdir)
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
