package inspiration

import (
	"strings"
	"testing"
)

func TestCreateAndList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("NOVA_HOME", home)

	store, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	insp, err := store.Create(CreateInput{
		Spark: "废土世界，主角能回溯 10 秒",
		Genre: "科幻",
	})
	if err != nil {
		t.Fatal(err)
	}
	if insp.ID == "" {
		t.Fatal("expected id")
	}
	if insp.Status != StatusReady {
		t.Fatalf("status=%q want ready", insp.Status)
	}

	store2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	cards := store2.ListCards(ListFilter{})
	if len(cards) != 1 {
		t.Fatalf("want 1 card, got %d", len(cards))
	}
	if cards[0].Title != "废土世界，主角能回溯 10 秒" {
		t.Fatalf("title=%q", cards[0].Title)
	}
}

func TestComputeStatusReady(t *testing.T) {
	insp := Inspiration{
		Spark:       "一个想法",
		Protagonist: "林风",
	}
	if ComputeStatus(insp) != StatusReady {
		t.Fatalf("got %s", ComputeStatus(insp))
	}
}

func TestDiscoverSeedPrompt(t *testing.T) {
	p := DiscoverSeedPrompt(Inspiration{
		Title: "时间回溯者",
		Genre: "科幻",
		Spark: "废土+回溯",
	})
	for _, part := range []string{"时间回溯者", "废土+回溯", "科幻"} {
		if !strings.Contains(p, part) {
			t.Fatalf("seed prompt missing %q: %q", part, p)
		}
	}
}

func TestMarkUsed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("NOVA_HOME", home)

	store, _ := Load()
	insp, _ := store.Create(CreateInput{Spark: "test"})
	if err := store.MarkUsed(insp.ID, "novel-1", "/tmp/book", "测试书"); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(insp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusUsed || got.NovelTitle != "测试书" {
		t.Fatalf("got %+v", got)
	}
}
