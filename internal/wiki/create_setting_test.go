package wiki

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestCreateSetting(t *testing.T) {
	dir := t.TempDir()
	p := &project.Project{
		Root: dir,
		Meta: project.Meta{Title: "测试书", Genre: "玄幻", Style: "热血"},
	}
	if err := os.MkdirAll(filepath.Join(dir, "设定集", project.SettingsSubCharacter), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	c, err := CreateSetting(p, st, project.SettingsSubCharacter, "配角-李师兄", "character")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "配角-李师兄" {
		t.Fatalf("title %q", c.Title)
	}
	path := filepath.Join(dir, "设定集", "角色", "配角-李师兄.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	_, err = CreateSetting(p, st, project.SettingsSubCharacter, "配角-李师兄", "character")
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}
