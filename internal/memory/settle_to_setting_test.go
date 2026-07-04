package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestSettleCharacterMemoryToSetting(t *testing.T) {
	dir := t.TempDir()
	p := &project.Project{
		Root: dir,
		Meta: project.Meta{Title: "测试", Genre: "玄幻", Style: "热血"},
	}
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	id := "mem-test"
	if err := st.InsertMemory(store.Memory{
		ID: id, Category: "character", Subject: "林枫", Content: "性格偏隐忍",
		SourceChapter: 5, Status: "active", CreatedAt: project.Timestamp(),
	}); err != nil {
		t.Fatal(err)
	}

	res, err := SettleCharacterMemoryToSetting(p, st, id)
	if err != nil {
		t.Fatal(err)
	}
	if res.RelPath != "角色/林枫.md" {
		t.Fatalf("rel=%q", res.RelPath)
	}

	data, err := os.ReadFile(filepath.Join(p.SettingsDir(), "角色", "林枫.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "性格偏隐忍") {
		t.Fatalf("missing content: %s", body)
	}
	if !strings.Contains(body, "第 5 章沉淀") {
		t.Fatalf("missing chapter note: %s", body)
	}

	m, err := st.GetMemory(id)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != "archived" {
		t.Fatalf("status=%q want archived", m.Status)
	}
}

func TestSettleCharacterMemoryAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	p := &project.Project{Root: dir, Meta: project.Meta{Title: "测试", Genre: "玄幻"}}
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	path := filepath.Join(p.SettingsDir(), "角色", "林枫.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# 林枫\n\n## 性格\n\n- 已有条目\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	id := "mem-2"
	_ = st.InsertMemory(store.Memory{
		ID: id, Category: "character", Subject: "林枫", Content: "遇事冷静",
		Status: "active", CreatedAt: project.Timestamp(),
	})

	_, err = SettleCharacterMemoryToSetting(p, st, id)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "已有条目") || !strings.Contains(string(data), "遇事冷静") {
		t.Fatalf("append failed: %s", data)
	}
}

func TestSettleCharacterMemoryRejectsNonCharacter(t *testing.T) {
	dir := t.TempDir()
	p := &project.Project{Root: dir, Meta: project.Meta{Title: "测试", Genre: "玄幻"}}
	st, err := store.Open(filepath.Join(dir, ".nova", "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	id := "mem-plot"
	_ = st.InsertMemory(store.Memory{
		ID: id, Category: "plot", Subject: "主线", Content: "复仇",
		Status: "active", CreatedAt: project.Timestamp(),
	})
	_, err = SettleCharacterMemoryToSetting(p, st, id)
	if err == nil {
		t.Fatal("expected error for non-character memory")
	}
}
