package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCustomCategoryOrder(t *testing.T) {
	root := t.TempDir()
	p := &Project{Root: root, Meta: Meta{Genre: "玄幻"}}
	if err := os.MkdirAll(p.SettingsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateSettingCategory("功法"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.CreateSettingCategory("种族"); err != nil {
		t.Fatal(err)
	}

	if err := p.SaveSettingCategoryOrderValidated([]string{"种族", "功法", "角色", "世界观", "势力", "地点", "物品"}); err != nil {
		t.Fatal(err)
	}
	cats, err := p.ListSettingCategories()
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) < 2 || cats[0].ID != "种族" || cats[1].ID != "功法" {
		t.Fatalf("order = %v", categoryIDs(cats))
	}
}

func TestFullCategoryOrder(t *testing.T) {
	root := t.TempDir()
	p := &Project{Root: root}
	if err := os.MkdirAll(p.SettingsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	order := []string{"物品", "地点", "势力", "世界观", "角色"}
	if err := p.SaveSettingCategoryOrderValidated(order); err != nil {
		t.Fatal(err)
	}
	cats, err := p.ListSettingCategories()
	if err != nil {
		t.Fatal(err)
	}
	got := categoryIDs(cats)
	if len(got) != 5 {
		t.Fatalf("got %d categories", len(got))
	}
	for i, id := range order {
		if got[i] != id {
			t.Fatalf("index %d: got %q want %q (full=%v)", i, got[i], id, got)
		}
	}
}

func TestSettingChecklist(t *testing.T) {
	root := t.TempDir()
	p := &Project{Root: root, Meta: Meta{Genre: "玄幻"}}
	if err := os.MkdirAll(filepath.Join(p.SettingsDir(), SettingsSubWorld), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.SettingsDir(), SettingsSubWorld, "世界观.md"), []byte("# 世界观"), 0o644); err != nil {
		t.Fatal(err)
	}
	items, genre, err := p.SettingChecklist()
	if err != nil {
		t.Fatal(err)
	}
	if genre != "玄幻" {
		t.Fatalf("genre = %q", genre)
	}
	done := 0
	for _, it := range items {
		if it.CategoryID == "世界观" && it.Title == "世界观" && it.Done {
			done++
		}
	}
	if done != 1 {
		t.Fatalf("expected 世界观 done, items=%+v", items)
	}
}
