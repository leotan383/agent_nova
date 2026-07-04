package wiki

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// CreateSetting 在设定集子目录新建 Markdown 并返回条目内容。
func CreateSetting(p *project.Project, st *store.Store, subdir, title, templateKind string) (Content, error) {
	title = project.SanitizeSettingTitle(title)
	if title == "" {
		return Content{}, fmt.Errorf("请填写名称")
	}
	subdir = strings.TrimSpace(subdir)
	if subdir == "" {
		return Content{}, fmt.Errorf("无效分类目录")
	}
	for _, sub := range project.DefaultSettingSubdirs {
		if sub == subdir {
			goto validSubdir
		}
	}
	return Content{}, fmt.Errorf("无效分类: %s", subdir)

validSubdir:
	rel := filepath.ToSlash(filepath.Join(subdir, title+".md"))
	if err := project.ValidateSettingRelPath(rel); err != nil {
		return Content{}, err
	}
	path := filepath.Join(p.SettingsDir(), filepath.FromSlash(rel))
	if _, err := os.Stat(path); err == nil {
		return Content{}, fmt.Errorf("「%s」已存在", title)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Content{}, err
	}
	body := project.SettingBodyTemplate(subdir, title, templateKind, p.Meta)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return Content{}, err
	}
	if st != nil {
		_ = st.IndexSettingFTS(rel, body)
	}
	return Get(p, st, settingID(rel))
}
