package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListAndCreateSettingCategories(t *testing.T) {
	root := t.TempDir()
	p := &Project{Root: root}
	if err := os.MkdirAll(p.SettingsDir(), 0o755); err != nil {
		t.Fatal(err)
	}

	cats, err := p.ListSettingCategories()
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 5 {
		t.Fatalf("want 5 visible builtin categories, got %d", len(cats))
	}

	created, err := p.CreateSettingCategory("功法")
	if err != nil {
		t.Fatal(err)
	}
	if created.Subdir != "功法" || created.Builtin {
		t.Fatalf("unexpected created: %+v", created)
	}

	cats, err = p.ListSettingCategories()
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 6 {
		t.Fatalf("want 6 categories (5 visible + 1 custom), got %d", len(cats))
	}

	if _, err := p.CreateSettingCategory("世界观"); err == nil {
		t.Fatal("expected duplicate builtin error")
	}
	if _, err := p.CreateSettingCategory("世界"); err == nil {
		t.Fatal("expected reserved subdir error")
	}

	if _, err := p.RenameSettingCategory("角色", "新角色"); err == nil {
		t.Fatal("expected builtin rename error")
	}

	renamed, err := p.RenameSettingCategory("功法", "武学")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != "武学" {
		t.Fatalf("renamed id = %q", renamed.ID)
	}
	path := filepath.Join(p.SettingsDir(), "武学")
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		t.Fatalf("renamed dir missing: %v", err)
	}

	deleted, err := p.DeleteSettingCategory("武学")
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 0 {
		t.Fatalf("expected 0 deleted files, got %d", len(deleted))
	}

	sub, ok := ResolveCategorySubdir("世界观")
	if !ok || sub != SettingsSubWorld {
		t.Fatalf("ResolveCategorySubdir 世界观 = %q ok=%v", sub, ok)
	}
	if got := SubdirToCategoryID(SettingsSubWorld); got != "世界观" {
		t.Fatalf("SubdirToCategoryID world = %q", got)
	}
	if got := SubdirToCategoryID("武学"); got != "武学" {
		t.Fatalf("SubdirToCategoryID custom = %q", got)
	}
}
