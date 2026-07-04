package wiki

import (
	"fmt"
	"os"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func reindexSettingSubdir(p *project.Project, st *store.Store, subdir string) error {
	if st == nil {
		return nil
	}
	return p.WalkSettingMarkdownFiles(subdir, func(relPath, absPath string) error {
		data, err := os.ReadFile(absPath)
		if err != nil {
			return err
		}
		return st.IndexSettingFTS(relPath, string(data))
	})
}

func customSubdir(p *project.Project, categoryID string) (string, error) {
	cats, err := p.ListSettingCategories()
	if err != nil {
		return "", err
	}
	for _, c := range cats {
		if c.ID == categoryID {
			if c.Builtin {
				return "", fmt.Errorf("内置分类不可修改")
			}
			return c.Subdir, nil
		}
	}
	return "", fmt.Errorf("分类不存在")
}

// RenameSettingCategory 重命名用户自定义分类并更新检索索引。
func RenameSettingCategory(p *project.Project, st *store.Store, oldID, newName string) (project.SettingCategoryInfo, error) {
	oldSubdir, err := customSubdir(p, oldID)
	if err != nil {
		return project.SettingCategoryInfo{}, err
	}
	if st != nil {
		_ = p.WalkSettingMarkdownFiles(oldSubdir, func(relPath, _ string) error {
			return st.DeleteSettingFTS(relPath)
		})
	}
	renamed, err := p.RenameSettingCategory(oldID, newName)
	if err != nil {
		return project.SettingCategoryInfo{}, err
	}
	if err := reindexSettingSubdir(p, st, renamed.Subdir); err != nil {
		return project.SettingCategoryInfo{}, err
	}
	return renamed, nil
}

// DeleteSettingCategory 删除用户自定义分类及其下全部设定文件。
func DeleteSettingCategory(p *project.Project, st *store.Store, categoryID string) error {
	if _, err := customSubdir(p, categoryID); err != nil {
		return err
	}
	deleted, err := p.DeleteSettingCategory(categoryID)
	if err != nil {
		return err
	}
	if st == nil {
		return nil
	}
	for _, rel := range deleted {
		if err := st.DeleteSettingFTS(rel); err != nil {
			return err
		}
	}
	return nil
}
