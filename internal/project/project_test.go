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
	if got != "第001章-开端.md" {
		t.Fatalf("unexpected: %s", got)
	}
}
