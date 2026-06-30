package main

import (
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/pipeline"
	"github.com/tanlian/agent_nova/internal/version"
)

// VersionEntryDTO 章节历史版本。
type VersionEntryDTO struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Source    string `json:"source"`
	Label     string `json:"label"`
	WordCount int    `json:"word_count"`
	File      string `json:"file"`
}

// DiffLineDTO 单行 diff。
type DiffLineDTO struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// DiffResultDTO 两版对比结果。
type DiffResultDTO struct {
	FromID       string        `json:"from_id"`
	ToID         string        `json:"to_id"`
	FromLabel    string        `json:"from_label"`
	ToLabel      string        `json:"to_label"`
	Lines        []DiffLineDTO `json:"lines"`
	AddedWords   int           `json:"added_words"`
	RemovedWords int           `json:"removed_words"`
}

func toVersionDTOs(entries []version.Entry) []VersionEntryDTO {
	out := make([]VersionEntryDTO, len(entries))
	for i, e := range entries {
		out[i] = VersionEntryDTO{
			ID: e.ID, CreatedAt: e.CreatedAt, Source: e.Source,
			Label: e.Label, WordCount: e.WordCount, File: e.File,
		}
	}
	return out
}

func toDiffDTO(d version.DiffResult) DiffResultDTO {
	lines := make([]DiffLineDTO, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = DiffLineDTO{Type: l.Type, Text: l.Text}
	}
	return DiffResultDTO{
		FromID: d.FromID, ToID: d.ToID, FromLabel: d.FromLabel, ToLabel: d.ToLabel,
		Lines: lines, AddedWords: d.AddedWords, RemovedWords: d.RemovedWords,
	}
}

// ListChapterVersions 返回章节版本列表（新→旧）。
func (a *App) ListChapterVersions(chapter int) (out []VersionEntryDTO, err error) {
	if chapter <= 0 {
		return nil, fmt.Errorf("无效章号")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		entries, listErr := version.List(actx.Project, chapter)
		if listErr != nil {
			return listErr
		}
		out = toVersionDTOs(entries)
		return nil
	})
	return out, err
}

// PreviewChapterDiff 应用改稿前预览当前正文 vs 待写入内容。
func (a *App) PreviewChapterDiff(chapter int, newContent string) (DiffResultDTO, error) {
	newContent = strings.TrimSpace(newContent)
	if chapter <= 0 {
		return DiffResultDTO{}, fmt.Errorf("无效章号")
	}
	if newContent == "" {
		return DiffResultDTO{}, fmt.Errorf("正文不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return DiffResultDTO{}, err
	}
	var result DiffResultDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		diff, diffErr := version.DiffWithNew(actx.Project, chapter, newContent)
		if diffErr != nil {
			return diffErr
		}
		result = toDiffDTO(diff)
		return nil
	})
	return result, err
}

// DiffChapterVersions 对比两个版本；toID 可为 current。
func (a *App) DiffChapterVersions(chapter int, fromID, toID string) (DiffResultDTO, error) {
	if chapter <= 0 {
		return DiffResultDTO{}, fmt.Errorf("无效章号")
	}
	if fromID == "" {
		return DiffResultDTO{}, fmt.Errorf("请选择起始版本")
	}
	if toID == "" {
		toID = version.CurrentVersionID()
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return DiffResultDTO{}, err
	}
	var result DiffResultDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		diff, diffErr := version.Diff(actx.Project, chapter, fromID, toID)
		if diffErr != nil {
			return diffErr
		}
		result = toDiffDTO(diff)
		return nil
	})
	return result, err
}

// RestoreChapterVersion 将历史版本写回正文（会先快照当前版）。
func (a *App) RestoreChapterVersion(chapter int, versionID string) error {
	if chapter <= 0 {
		return fmt.Errorf("无效章号")
	}
	if versionID == "" {
		return fmt.Errorf("请选择版本")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		body, err := version.GetContent(actx.Project, chapter, versionID)
		if err != nil {
			return err
		}
		body = strings.TrimSpace(body)
		if body == "" {
			return fmt.Errorf("版本正文为空")
		}
		title := pipeline.ParseChapterTitle(body)
		if title == "" {
			if ch, chErr := actx.Store.GetChapter(chapter); chErr == nil {
				title = ch.Title
			}
		}
		label := fmt.Sprintf("恢复 %s", versionID)
		path, err := pipeline.SaveChapterWithVersion(actx.Project, chapter, title, body, version.SourceRestore, label)
		if err != nil {
			return err
		}
		return pipeline.PostWriteIndex(actx.Project, actx.Store, chapter, path)
	})
}
