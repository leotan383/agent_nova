package store

import "testing"

func TestCanonicalEntityName(t *testing.T) {
	cases := map[string]string{
		"母亲":         "母亲",
		"母亲（未具名）":   "母亲",
		"母亲（影像）":    "母亲",
		"母亲(未具名)":   "母亲",
		"林枫":         "林枫",
		"未知人物（窥伺者）": "未知人物",
	}
	for in, want := range cases {
		if got := CanonicalEntityName(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}

func TestEntityIDUsesCanonicalName(t *testing.T) {
	a := EntityID("character", "母亲（未具名）")
	b := EntityID("character", "母亲")
	if a != b {
		t.Fatalf("%q != %q", a, b)
	}
}

func TestMergeDuplicateEntities(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir + "/nova.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	_ = st.UpsertEntity(Entity{ID: "character:母亲（未具名）", Type: "character", Name: "母亲（未具名）", LastChapter: 3})
	_ = st.UpsertEntity(Entity{ID: "character:母亲（影像）", Type: "character", Name: "母亲（影像）", LastChapter: 5})
	_ = st.UpsertEntity(Entity{ID: "character:母亲", Type: "character", Name: "母亲", LastChapter: 7})

	n, err := st.MergeDuplicateEntities()
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("merged %d, want 2", n)
	}
	all, _ := st.ListEntities("character", 100)
	if len(all) != 1 {
		t.Fatalf("entities left: %d", len(all))
	}
	if all[0].Name != "母亲" || all[0].LastChapter != 7 {
		t.Fatalf("got %+v", all[0])
	}
}
