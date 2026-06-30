package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/project"
)

// Options 导出范围与格式选项。
type Options struct {
	FromChapter int // 0 = 不限
	ToChapter   int // 0 = 不限
}

// ChapterFile 待导出章节。
type ChapterFile struct {
	Number int
	Path   string
	Name   string
	Body   string
}

var chapterNumRe = regexp.MustCompile(`^第(\d+)章`)

// ListChapters 按章号排序列出正文文件。
func ListChapters(p *project.Project, opts Options) ([]ChapterFile, error) {
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return nil, err
	}
	byNum := map[int]ChapterFile{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		m := chapterNumRe.FindStringSubmatch(e.Name())
		if len(m) < 2 {
			continue
		}
		var num int
		fmt.Sscanf(m[1], "%d", &num)
		if num <= 0 {
			continue
		}
		if opts.FromChapter > 0 && num < opts.FromChapter {
			continue
		}
		if opts.ToChapter > 0 && num > opts.ToChapter {
			continue
		}
		path := filepath.Join(p.ChaptersDir(), e.Name())
		prev, ok := byNum[num]
		if ok {
			info, err1 := e.Info()
			prevInfo, err2 := os.Stat(prev.Path)
			if err1 == nil && err2 == nil && !info.ModTime().After(prevInfo.ModTime()) {
				continue
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		byNum[num] = ChapterFile{Number: num, Path: path, Name: e.Name(), Body: string(data)}
	}
	if len(byNum) == 0 {
		return nil, fmt.Errorf("无章节可导出")
	}
	nums := make([]int, 0, len(byNum))
	for n := range byNum {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	out := make([]ChapterFile, len(nums))
	for i, n := range nums {
		out[i] = byNum[n]
	}
	return out, nil
}

// WriteEPUB exports chapters under 正文/ to a minimal EPUB3 file.
func WriteEPUB(p *project.Project, outPath string, opts Options) error {
	chs, err := ListChapters(p, opts)
	if err != nil {
		return err
	}
	if outPath == "" {
		outPath = filepath.Join(p.Root, sanitizeFilename(p.Meta.Title)+".epub")
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	writeZipFile(zw, "mimetype", "application/epub+zip", false)
	container := `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`
	writeZipFile(zw, "META-INF/container.xml", container, true)
	var manifest strings.Builder
	var spine strings.Builder
	var nav strings.Builder
	nav.WriteString(`<?xml version="1.0" encoding="UTF-8"?><html xmlns="http://www.w3.org/1999/xhtml"><head><title>目录</title></head><body><nav><ol>`)
	for i, ch := range chs {
		id := fmt.Sprintf("ch%03d", i+1)
		label := chapterLabel(ch)
		body := markdownToXHTML(ch.Body)
		xhtml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE html><html xmlns="http://www.w3.org/1999/xhtml"><head><title>%s</title></head><body>%s</body></html>`, html.EscapeString(label), body)
		writeZipFile(zw, "OEBPS/"+id+".xhtml", xhtml, true)
		manifest.WriteString(fmt.Sprintf(`<item id="%s" href="%s.xhtml" media-type="application/xhtml+xml"/>`, id, id))
		spine.WriteString(fmt.Sprintf(`<itemref idref="%s"/>`, id))
		nav.WriteString(fmt.Sprintf(`<li><a href="%s.xhtml">%s</a></li>`, id, html.EscapeString(label)))
	}
	nav.WriteString(`</ol></nav></body></html>`)
	writeZipFile(zw, "OEBPS/nav.xhtml", nav.String(), true)
	manifest.WriteString(`<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml"/>`)
	opf := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="uuid"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>%s</dc:title><dc:language>zh-CN</dc:language><dc:identifier id="uuid">urn:uuid:nova-%d</dc:identifier></metadata><manifest>%s</manifest><spine>%s</spine></package>`,
		html.EscapeString(p.Meta.Title), time.Now().Unix(), manifest.String(), spine.String())
	writeZipFile(zw, "OEBPS/content.opf", opf, true)
	return nil
}

func writeZipFile(zw *zip.Writer, name, content string, compress bool) {
	method := zip.Store
	if compress {
		method = zip.Deflate
	}
	w, _ := zw.CreateHeader(&zip.FileHeader{Name: name, Method: uint16(method)})
	_, _ = w.Write([]byte(content))
}

func markdownToXHTML(md string) string {
	var b strings.Builder
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			b.WriteString("<p></p>")
			continue
		}
		if strings.HasPrefix(line, "# ") {
			b.WriteString("<h1>" + html.EscapeString(strings.TrimPrefix(line, "# ")) + "</h1>")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			b.WriteString("<h2>" + html.EscapeString(strings.TrimPrefix(line, "## ")) + "</h2>")
			continue
		}
		b.WriteString("<p>" + html.EscapeString(line) + "</p>")
	}
	return b.String()
}

// WriteMarkdown merges chapters to a single markdown file.
func WriteMarkdown(p *project.Project, outPath string, opts Options) error {
	chs, err := ListChapters(p, opts)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n\n", p.Meta.Title))
	for i, ch := range chs {
		if i > 0 {
			buf.WriteString("\n\n---\n\n")
		}
		buf.WriteString(ch.Body)
	}
	if outPath == "" {
		outPath = filepath.Join(p.Root, "export.md")
	}
	return os.WriteFile(outPath, buf.Bytes(), 0o644)
}

// WriteTXT 导出为纯文本（去除 Markdown 标题符号，保留段落）。
func WriteTXT(p *project.Project, outPath string, opts Options) error {
	chs, err := ListChapters(p, opts)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(p.Meta.Title + "\n")
	buf.WriteString(strings.Repeat("=", utf8.RuneCountInString(p.Meta.Title)) + "\n\n")
	for i, ch := range chs {
		if i > 0 {
			buf.WriteString("\n\n" + strings.Repeat("-", 40) + "\n\n")
		}
		buf.WriteString(chapterLabel(ch) + "\n\n")
		buf.WriteString(markdownToPlain(ch.Body))
	}
	if outPath == "" {
		outPath = filepath.Join(p.Root, sanitizeFilename(p.Meta.Title)+".txt")
	}
	return os.WriteFile(outPath, buf.Bytes(), 0o644)
}

func markdownToPlain(md string) string {
	var lines []string
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			lines = append(lines, "")
			continue
		}
		for strings.HasPrefix(line, "#") {
			line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func chapterLabel(ch ChapterFile) string {
	title := strings.TrimSuffix(ch.Name, ".md")
	if title != "" {
		return title
	}
	return fmt.Sprintf("第%d章", ch.Number)
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "novel"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")
	return replacer.Replace(s)
}

// CountWords 统计 UTF-8 字数（不计空白）。
func CountWords(text string) int {
	n := 0
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		n++
	}
	return n
}
