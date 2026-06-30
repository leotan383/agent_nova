package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tanlian/agent_nova/internal/project"
)

func TestRegisterAndList(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("NOVA_HOME", home)

	in := project.InitInput{Dir: dir, Title: "测试书", Genre: "玄幻"}
	res, err := project.InitProject(in)
	if err != nil {
		t.Fatal(err)
	}

	reg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	e, err := reg.Register(res.Root)
	if err != nil {
		t.Fatal(err)
	}
	if e.ID == "" {
		t.Fatal("expected id")
	}
	if reg.ActiveID != e.ID {
		t.Fatal("active id mismatch")
	}

	cards := reg.ListCards(true)
	if len(cards) != 1 {
		t.Fatalf("want 1 card, got %d", len(cards))
	}
	if cards[0].Title != "测试书" {
		t.Fatalf("title=%q", cards[0].Title)
	}
	if cards[0].Missing {
		t.Fatal("should not be missing")
	}

	if err := reg.SetActive(e.ID); err != nil {
		t.Fatal(err)
	}
	if err := reg.Remove(e.ID); err != nil {
		t.Fatal(err)
	}
	if len(reg.Novels) != 0 {
		t.Fatal("expected empty after remove")
	}

	// verify library file written under NOVA_HOME
	libPath := filepath.Join(home, "config", fileName)
	if _, err := os.Stat(libPath); err != nil {
		t.Fatal(err)
	}
}

func TestRepairStaleActive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("NOVA_HOME", home)

	reg := &Registry{
		ActiveID: "gone-id",
		Novels: []Entry{{
			ID: "keep-id", Path: "/tmp/novel-a",
			LastOpenedAt: time.Now().UTC(),
		}},
	}
	if err := reg.Repair(); err != nil {
		t.Fatal(err)
	}
	if reg.ActiveID != "keep-id" {
		t.Fatalf("active_id=%q want keep-id", reg.ActiveID)
	}
}
