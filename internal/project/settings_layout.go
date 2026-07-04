package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 设定集子目录，与 Studio 侧边栏分类对齐。
const (
	SettingsSubCharacter = "角色"
	SettingsSubWorld     = "世界"
	SettingsSubFaction   = "势力"
	SettingsSubLocation  = "地点"
	SettingsSubItem      = "物品"
	SettingsSubOther     = "其他"
)

// DefaultSettingSubdirs 新建项目时创建的设定集子目录。
var DefaultSettingSubdirs = []string{
	SettingsSubCharacter,
	SettingsSubWorld,
	SettingsSubFaction,
	SettingsSubLocation,
	SettingsSubItem,
	SettingsSubOther,
}

// SettingFileSubdir 返回 Init 模板文件应放置的子目录。
func SettingFileSubdir(filename string) string {
	switch filename {
	case "主角卡.md", "反派设计.md":
		return SettingsSubCharacter
	case "世界观.md", "力量体系.md", "科技体系.md", "金手指.md":
		return SettingsSubWorld
	case "势力关系.md":
		return SettingsSubFaction
	default:
		return SettingsSubOther
	}
}

// ClassifySettingRel 根据相对路径（如 角色/主角卡.md）推断分组；无子目录时回退文件名规则。
func ClassifySettingRel(relPath, title string) string {
	sub := SettingSubdirName(relPath)
	if sub != "" {
		switch sub {
		case SettingsSubCharacter:
			return "人物"
		case SettingsSubWorld, SettingsSubFaction, SettingsSubLocation, SettingsSubItem, SettingsSubOther:
			return "设定"
		}
	}
	return classifySettingByTitle(title)
}

func classifySettingByTitle(name string) string {
	switch name {
	case "主角卡", "反派设计":
		return "人物"
	}
	for _, kw := range []string{"主角", "角色", "人物", "反派", "配角"} {
		if strings.Contains(name, kw) {
			return "人物"
		}
	}
	return "设定"
}

// SettingSubdirName 从 setting 相对路径提取一级子目录名。
func SettingSubdirName(relPath string) string {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	if relPath == "" || relPath == "." {
		return ""
	}
	parts := strings.Split(relPath, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// SettingSubtitle 列表副标题，展示子目录分类。
func SettingSubtitle(relPath string) string {
	sub := SettingSubdirName(relPath)
	if sub != "" {
		return sub
	}
	return "设定集"
}

func (p *Project) SettingCategoryDir(category string) string {
	return filepath.Join(p.SettingsDir(), category)
}

// EnsureSettingsSubdirs 创建全部设定集子目录（幂等）。
func (p *Project) EnsureSettingsSubdirs() error {
	for _, sub := range DefaultSettingSubdirs {
		if err := os.MkdirAll(p.SettingCategoryDir(sub), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// MigrateFlatSettings 将设定集根目录下的 .md 移入对应子目录（幂等，仅处理根层文件）。
func (p *Project) MigrateFlatSettings() (int, error) {
	if err := p.EnsureSettingsSubdirs(); err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(p.SettingsDir())
	if err != nil {
		return 0, err
	}
	moved := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		sub := SettingFileSubdir(e.Name())
		if sub == "" {
			sub = SettingsSubOther
		}
		src := filepath.Join(p.SettingsDir(), e.Name())
		dst := filepath.Join(p.SettingsDir(), sub, e.Name())
		if _, err := os.Stat(dst); err == nil {
			return moved, fmt.Errorf("无法迁移 %s：目标已存在 %s", e.Name(), dst)
		}
		if err := os.Rename(src, dst); err != nil {
			return moved, err
		}
		moved++
	}
	return moved, nil
}

// ValidateSettingRelPath 校验 setting 相对路径（允许 角色/主角卡.md）。
func ValidateSettingRelPath(rel string) error {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.Contains(rel, "..") {
		return fmt.Errorf("无效路径")
	}
	if !strings.HasSuffix(rel, ".md") {
		return fmt.Errorf("仅支持 Markdown 文件")
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("无效路径")
		}
	}
	return nil
}
