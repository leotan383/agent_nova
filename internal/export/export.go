package export

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tanlian/agent_nova/internal/project"
)

// WriteEPUB exports chapters under 正文/ to a minimal EPUB3 file.
func WriteEPUB(p *project.Project, outPath string) error {
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("无章节可导出")
	}
	if outPath == "" {
		outPath = filepath.Join(p.Root, p.Meta.Title+".epub")
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
	for i, name := range files {
		id := fmt.Sprintf("ch%03d", i+1)
		data, err := os.ReadFile(filepath.Join(p.ChaptersDir(), name))
		if err != nil {
			return err
		}
		body := markdownToXHTML(string(data))
		xhtml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?><!DOCTYPE html><html xmlns="http://www.w3.org/1999/xhtml"><head><title>%s</title></head><body>%s</body></html>`, html.EscapeString(strings.TrimSuffix(name, ".md")), body)
		writeZipFile(zw, "OEBPS/"+id+".xhtml", xhtml, true)
		manifest.WriteString(fmt.Sprintf(`<item id="%s" href="%s.xhtml" media-type="application/xhtml+xml"/>`, id, id))
		spine.WriteString(fmt.Sprintf(`<itemref idref="%s"/>`, id))
		nav.WriteString(fmt.Sprintf(`<li><a href="%s.xhtml">%s</a></li>`, id, html.EscapeString(strings.TrimSuffix(name, ".md"))))
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
func WriteMarkdown(p *project.Project, outPath string) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s\n\n", p.Meta.Title))
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.ChaptersDir(), e.Name()))
		if err != nil {
			return err
		}
		buf.WriteString("\n\n---\n\n")
		buf.Write(data)
	}
	return os.WriteFile(outPath, buf.Bytes(), 0o644)
}
