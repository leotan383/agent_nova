package agent

import "testing"

func TestThinkingStreamParser(t *testing.T) {
	var thinking, content string
	p := newThinkingStreamParser()
	onThink := func(d string) error { thinking += d; return nil }
	onContent := func(d string) error { content += d; return nil }

	chunks := []string{"[think]", "分析节奏", "[/think]", "建议压缩描写"}
	for _, c := range chunks {
		if err := p.Feed(c, onThink, onContent); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Flush(onThink, onContent); err != nil {
		t.Fatal(err)
	}
	if thinking != "分析节奏" {
		t.Fatalf("thinking=%q", thinking)
	}
	if content != "建议压缩描写" {
		t.Fatalf("content=%q", content)
	}
}

func TestThinkingStreamParserNoTags(t *testing.T) {
	var content string
	p := newThinkingStreamParser()
	err := p.Feed("直接回复", nil, func(d string) error {
		content += d
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if content != "直接回复" {
		t.Fatalf("content=%q", content)
	}
}
