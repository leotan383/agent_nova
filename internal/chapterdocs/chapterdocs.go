package chapterdocs

import (
	"fmt"
	"os"
	"strings"

	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/version"
)

const (
	KindBody    = "body"
	KindReview  = "review"
	KindSummary = "summary"
)

// Doc 章节关联文档。
type Doc struct {
	Kind    string `json:"kind"`
	Chapter int    `json:"chapter"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Exists  bool   `json:"exists"`
	Path    string `json:"path,omitempty"`
}

func pathFor(p *project.Project, chapter int, kind string) (string, error) {
	switch kind {
	case KindBody:
		path, _, err := p.FindChapterFile(chapter)
		return path, err
	case KindReview:
		return p.ReviewPath(chapter), nil
	case KindSummary:
		return p.SummaryPath(chapter), nil
	default:
		return "", fmt.Errorf("未知文档类型: %s", kind)
	}
}

func titleFor(kind string, chapter int) string {
	switch kind {
	case KindBody:
		return fmt.Sprintf("第%d章 · 正文", chapter)
	case KindReview:
		return fmt.Sprintf("第%d章 · 审查", chapter)
	case KindSummary:
		return fmt.Sprintf("第%d章 · 摘要", chapter)
	default:
		return fmt.Sprintf("第%d章", chapter)
	}
}

// Get 读取章节正文 / 审查 / 摘要。
func Get(p *project.Project, chapter int, kind string) (Doc, error) {
	if chapter <= 0 {
		return Doc{}, fmt.Errorf("无效章号")
	}
	path, err := pathFor(p, chapter, kind)
	if err != nil && kind == KindBody {
		return Doc{}, err
	}
	if kind != KindBody {
		path, _ = pathFor(p, chapter, kind)
	}
	doc := Doc{Kind: kind, Chapter: chapter, Title: titleFor(kind, chapter), Path: path}
	if path == "" {
		return doc, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return Doc{}, err
	}
	doc.Body = string(data)
	doc.Exists = strings.TrimSpace(doc.Body) != ""
	return doc, nil
}

// Save 保存章节文档；正文会留版本快照并重建索引。
func Save(p *project.Project, st *store.Store, chapter int, kind, body string) (string, error) {
	if chapter <= 0 {
		return "", fmt.Errorf("无效章号")
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return "", fmt.Errorf("内容不能为空")
	}
	switch kind {
	case KindBody:
		title := pipeline.ParseChapterTitle(body)
		if title == "" {
			if ch, err := st.GetChapter(chapter); err == nil {
				title = ch.Title
			}
		}
		path, err := pipeline.SaveChapterWithVersion(p, chapter, title, body, version.SourceManualEdit, "手动编辑")
		if err != nil {
			return "", err
		}
		if err := pipeline.PostWriteIndex(p, st, chapter, path); err != nil {
			return "", err
		}
		return path, nil
	case KindReview:
		path := p.ReviewPath(chapter)
		if err := os.MkdirAll(p.ReviewsDir(), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return "", err
		}
		return path, nil
	case KindSummary:
		path := p.SummaryPath(chapter)
		if err := os.MkdirAll(p.SummariesDir(), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", fmt.Errorf("未知文档类型: %s", kind)
	}
}
