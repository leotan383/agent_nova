package wiki

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestListAndGetSetting(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.Load(res.Root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(p.Root, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_ = st.InitProject(p.Root, p.Meta)

	entries, err := List(p, st)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected setting entries")
	}

	var protagonistID string
	for _, e := range entries {
		if e.Title == "主角卡" {
			protagonistID = e.ID
			if e.Group != GroupCharacter {
				t.Fatalf("group=%s", e.Group)
			}
			if e.Subtitle != project.SettingsSubCharacter {
				t.Fatalf("subtitle=%s", e.Subtitle)
			}
		}
	}
	if protagonistID == "" {
		t.Fatal("missing 主角卡 entry")
	}

	content, err := Get(p, st, protagonistID)
	if err != nil {
		t.Fatal(err)
	}
	if content.Body == "" {
		t.Fatal("empty body")
	}

	charDir := filepath.Join(p.SettingsDir(), project.SettingsSubCharacter)
	_ = os.WriteFile(filepath.Join(charDir, "配角-李四.md"), []byte("# 李四\n\n配角设定"), 0o644)
	entries2, _ := List(p, st)
	found := false
	for _, e := range entries2 {
		if e.Title == "配角-李四" && e.Group == GroupCharacter {
			found = true
		}
	}
	if !found {
		t.Fatal("expected 配角 in character group")
	}
}

func TestListDedupesEntityVariants(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := project.Load(res.Root)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(filepath.Join(p.Root, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_ = st.InitProject(p.Root, p.Meta)

	_ = st.UpsertEntity(store.Entity{ID: "character:母亲（未具名）", Type: "character", Name: "母亲（未具名）", LastChapter: 3})
	_ = st.UpsertEntity(store.Entity{ID: "character:母亲（影像）", Type: "character", Name: "母亲（影像）", LastChapter: 5})
	_ = st.UpsertEntity(store.Entity{ID: "character:母亲", Type: "character", Name: "母亲", LastChapter: 7})

	entries, err := List(p, st)
	if err != nil {
		t.Fatal(err)
	}
	var mother []Entry
	for _, e := range entries {
		if e.Kind == KindEntity && e.Title == "母亲" {
			mother = append(mother, e)
		}
	}
	if len(mother) != 1 {
		t.Fatalf("expected 1 母亲 entity entry, got %d: %+v", len(mother), mother)
	}
	if mother[0].ID != "entity:character:母亲" {
		t.Fatalf("id=%s", mother[0].ID)
	}
}

func TestMigrateFlatSettings(t *testing.T) {
	dir := t.TempDir()
	res, err := project.InitProject(project.InitInput{Dir: dir, Title: "测试", Genre: "玄幻"})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := project.Load(res.Root)
	// 模拟旧项目：根目录平铺文件
	_ = os.WriteFile(filepath.Join(p.SettingsDir(), "旧设定.md"), []byte("# old"), 0o644)
	moved, err := p.MigrateFlatSettings()
	if err != nil {
		t.Fatal(err)
	}
	if moved != 1 {
		t.Fatalf("moved=%d", moved)
	}
	if _, err := os.Stat(filepath.Join(p.SettingsDir(), project.SettingsSubOther, "旧设定.md")); err != nil {
		t.Fatal("expected migrated file in 其他/")
	}
}
