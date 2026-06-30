package wiki

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

func TestClassifySettingName(t *testing.T) {
	if classifySettingName("主角卡") != GroupCharacter {
		t.Fatal("主角卡 should be character group")
	}
	if classifySettingName("世界观") != GroupSetting {
		t.Fatal("世界观 should be setting group")
	}
}

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

	_ = os.WriteFile(filepath.Join(p.SettingsDir(), "配角-李四.md"), []byte("# 李四\n\n配角设定"), 0o644)
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
