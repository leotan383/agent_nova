package project

import (
	"strings"
	"testing"
)

func TestSettingPrimaryValue(t *testing.T) {
	meta := Meta{Protagonist: "林枫"}
	cases := []struct {
		title string
		want  string
	}{
		{"王腾", "王腾"},
		{"配角-李师兄", "李师兄"},
		{"主角卡", "林枫"},
		{"反派设计", ""},
		{"  张三  ", "张三"},
	}
	for _, c := range cases {
		got := settingPrimaryValue(c.title, meta)
		if got != c.want {
			t.Fatalf("%q: got %q want %q", c.title, got, c.want)
		}
	}
}

func TestSettingBodyTemplateFillsCharacterName(t *testing.T) {
	body := SettingBodyTemplate(SettingsSubCharacter, "王腾", "character", Meta{Title: "测试", Genre: "玄幻"})
	if !strings.Contains(body, "## 姓名\n王腾") {
		t.Fatalf("body missing name: %q", body)
	}
}
