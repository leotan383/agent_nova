package project_test

import (
	"testing"

	"github.com/tanlian/agent_nova/internal/project"
)

func TestParseChapterRange(t *testing.T) {
	cases := []struct {
		in   string
		want []int
	}{
		{"1", []int{1}},
		{"1-3", []int{1, 2, 3}},
		{"3-1", []int{1, 2, 3}},
	}
	for _, c := range cases {
		got, err := project.ParseChapterRange(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%q: got %v want %v", c.in, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%q: got %v want %v", c.in, got, c.want)
			}
		}
	}
}

func TestChapterFileName(t *testing.T) {
	got := project.ChapterFileName(1, "开端")
	if got != "第001章-开端" {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestSettingFileSubdir(t *testing.T) {
	if project.SettingFileSubdir("主角卡.md") != project.SettingsSubCharacter {
		t.Fatal("主角卡 should be in 角色")
	}
	if project.SettingFileSubdir("世界观.md") != project.SettingsSubWorld {
		t.Fatal("世界观 should be in 世界")
	}
}

func TestClassifySettingRel(t *testing.T) {
	if project.ClassifySettingRel("角色/主角卡.md", "主角卡") != "人物" {
		t.Fatal("角色 subdir should classify as 人物")
	}
	if project.ClassifySettingRel("世界/力量体系.md", "力量体系") != "设定" {
		t.Fatal("世界 subdir should classify as 设定")
	}
}
