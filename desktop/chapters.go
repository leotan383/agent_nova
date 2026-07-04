package main

import (
	"fmt"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/chapterdocs"
)

// ChapterDocDTO 章节正文/审查/摘要。
type ChapterDocDTO struct {
	Kind    string `json:"kind"`
	Chapter int    `json:"chapter"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Exists  bool   `json:"exists"`
	Path    string `json:"path,omitempty"`
}

func toChapterDocDTO(d chapterdocs.Doc) ChapterDocDTO {
	return ChapterDocDTO{
		Kind: d.Kind, Chapter: d.Chapter, Title: d.Title,
		Body: d.Body, Exists: d.Exists, Path: d.Path,
	}
}

// GetChapterDocument 读取正文(body)、审查(review)、摘要(summary)或 AI味(ai_check)。
func (a *App) GetChapterDocument(chapter int, kind string) (ChapterDocDTO, error) {
	if chapter <= 0 {
		return ChapterDocDTO{}, fmt.Errorf("无效章号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return ChapterDocDTO{}, err
	}
	var result ChapterDocDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if kind == chapterdocs.KindBody {
			if err := a.syncChaptersFromDisk(actx); err != nil {
				return err
			}
		}
		doc, err := chapterdocs.Get(actx.Project, chapter, kind)
		if err != nil {
			return err
		}
		result = toChapterDocDTO(doc)
		return nil
	})
	return result, err
}

// SaveChapterDocument 保存正文/审查/摘要。
func (a *App) SaveChapterDocument(chapter int, kind, body string) error {
	if chapter <= 0 {
		return fmt.Errorf("无效章号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		_, err := chapterdocs.Save(actx.Project, actx.Store, chapter, kind, body)
		return err
	})
}
