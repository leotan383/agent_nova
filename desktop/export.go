package main

import (
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/export"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportInput 导出请求。
type ExportInput struct {
	Format      string `json:"format"`
	OutPath     string `json:"out_path"`
	FromChapter int    `json:"from_chapter"`
	ToChapter   int    `json:"to_chapter"`
}

// ExportResultDTO 导出结果。
type ExportResultDTO struct {
	Path         string `json:"path"`
	Format       string `json:"format"`
	ChapterCount int    `json:"chapter_count"`
	WordCount    int    `json:"word_count"`
}

// DefaultExportFilename 根据当前书与格式生成默认文件名。
func (a *App) DefaultExportFilename(format string) (string, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return "", err
	}
	var name string
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		base := actx.Project.Meta.Title
		if base == "" {
			base = "novel"
		}
		base = sanitizeExportName(base)
		switch strings.ToLower(format) {
		case "epub":
			name = base + ".epub"
		case "txt":
			name = base + ".txt"
		default:
			name = base + ".md"
		}
		return nil
	})
	return name, err
}

// PickExportPath 选择导出保存路径。
func (a *App) PickExportPath(format, defaultName string) (string, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if defaultName == "" {
		var err error
		defaultName, err = a.DefaultExportFilename(format)
		if err != nil {
			return "", err
		}
	}
	filter := runtime.FileFilter{DisplayName: "Markdown", Pattern: "*.md"}
	switch format {
	case "epub":
		filter = runtime.FileFilter{DisplayName: "EPUB 电子书", Pattern: "*.epub"}
	case "txt":
		filter = runtime.FileFilter{DisplayName: "纯文本", Pattern: "*.txt"}
	}
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "导出小说",
		DefaultFilename: defaultName,
		Filters:         []runtime.FileFilter{filter},
	})
}

// ExportProject 导出当前小说正文。
func (a *App) ExportProject(in ExportInput) (ExportResultDTO, error) {
	format := strings.ToLower(strings.TrimSpace(in.Format))
	if format == "" {
		format = "markdown"
	}
	if in.OutPath == "" {
		return ExportResultDTO{}, fmt.Errorf("请选择导出保存路径")
	}

	reg, err := a.loadRegistry()
	if err != nil {
		return ExportResultDTO{}, err
	}

	var result ExportResultDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		opts := export.Options{FromChapter: in.FromChapter, ToChapter: in.ToChapter}
		chs, err := export.ListChapters(actx.Project, opts)
		if err != nil {
			return err
		}
		outPath := in.OutPath
		switch format {
		case "epub":
			err = export.WriteEPUB(actx.Project, outPath, opts)
		case "txt":
			err = export.WriteTXT(actx.Project, outPath, opts)
		case "markdown", "md":
			err = export.WriteMarkdown(actx.Project, outPath, opts)
		default:
			return fmt.Errorf("不支持的格式：%s（可选 markdown/epub/txt）", format)
		}
		if err != nil {
			return err
		}
		words := 0
		for _, ch := range chs {
			words += export.CountWords(ch.Body)
		}
		result = ExportResultDTO{
			Path: outPath, Format: format,
			ChapterCount: len(chs), WordCount: words,
		}
		return nil
	})
	return result, err
}

func sanitizeExportName(s string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	s = strings.TrimSpace(replacer.Replace(s))
	if s == "" {
		return "novel"
	}
	return s
}
