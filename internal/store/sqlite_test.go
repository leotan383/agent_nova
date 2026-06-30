package store_test

import (
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestMemoryCRUD(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "nova.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	m := store.Memory{
		ID: project.NewMemoryID(), Category: "style", Subject: "钩子",
		Content: "章末悬念", SourceChapter: 1, Status: "active", CreatedAt: project.Timestamp(),
	}
	if err := st.InsertMemory(m); err != nil {
		t.Fatal(err)
	}
	items, err := st.QueryMemories("style", "钩", 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(items)
	if len(items) != 1 {
		t.Fatalf("want 1 got %d", len(items))
	}
}
