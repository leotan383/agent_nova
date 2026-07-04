package store

import "testing"

func TestEntityStateHistory(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir + "/nova.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	id := EntityID("character", "林枫")
	s1 := `{"处境":"初入宗门"}`
	s2 := `{"处境":"陷入绝境","玉佩":"光芒黯淡"}`

	if err := st.RecordEntityStateHistory(id, 3, s1); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordEntityStateHistory(id, 8, s2); err != nil {
		t.Fatal(err)
	}
	// 同章重提取应覆盖
	if err := st.RecordEntityStateHistory(id, 3, `{"处境":"修正后"}`); err != nil {
		t.Fatal(err)
	}

	hist, err := st.ListEntityStateHistory(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 {
		t.Fatalf("snapshots=%d", len(hist))
	}
	if hist[0].Chapter != 3 || hist[0].StateJSON != `{"处境":"修正后"}` {
		t.Fatalf("ch3=%+v", hist[0])
	}
	if hist[1].Chapter != 8 {
		t.Fatalf("ch8=%+v", hist[1])
	}
}

func TestReassignEntityHistoryOnMerge(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir + "/nova.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	target := EntityID("character", "母亲")
	alt := "character:母亲（影像）"
	_ = st.RecordEntityStateHistory(alt, 2, `{"身份":"影像"}`)
	_ = st.RecordEntityStateHistory(target, 5, `{"身份":"本人"}`)

	if err := st.reassignEntityHistory(alt, target); err != nil {
		t.Fatal(err)
	}
	hist, _ := st.ListEntityStateHistory(target)
	if len(hist) != 2 {
		t.Fatalf("merged history=%d", len(hist))
	}
	if _, err := st.ListEntityStateHistory(alt); err != nil {
		t.Fatal(err)
	}
	altHist, _ := st.ListEntityStateHistory(alt)
	if len(altHist) != 0 {
		t.Fatalf("alt history left=%d", len(altHist))
	}
}
