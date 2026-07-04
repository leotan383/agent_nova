package export

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/tanlian/agent_nova/internal/project"
)

const (
	pdfFontName   = "nova-body"
	pdfMarginMM   = 18.0
	pdfLineHeight = 7.0
)

// WritePDF 导出章节为 PDF（需系统 CJK 字体或 NOVA_PDF_FONT）。
func WritePDF(p *project.Project, outPath string, opts Options) error {
	chs, err := ListChapters(p, opts)
	if err != nil {
		return err
	}
	fontPath, err := resolvePDFFont()
	if err != nil {
		return err
	}
	if outPath == "" {
		outPath = filepath.Join(p.Root, sanitizeFilename(p.Meta.Title)+".pdf")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMarginMM, pdfMarginMM, pdfMarginMM)
	pdf.SetAutoPageBreak(true, pdfMarginMM)
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return fmt.Errorf("读取字体: %w", err)
	}
	pdf.AddUTF8FontFromBytes(pdfFontName, "", fontBytes)
	pdf.AddUTF8FontFromBytes(pdfFontName, "B", fontBytes)

	// 封面
	pdf.AddPage()
	writePDFTitle(pdf, p.Meta.Title)
	if sub := pdfSubtitle(p.Meta); sub != "" {
		writePDFParagraph(pdf, sub, 11, false, 6)
	}

	for _, ch := range chs {
		pdf.AddPage()
		label := chapterLabel(ch)
		pdf.Bookmark(label, 0, -1)
		writePDFHeading(pdf, label, 16)
		writePDFMarkdownBody(pdf, ch.Body)
	}

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return err
	}
	return nil
}

func pdfSubtitle(meta project.Meta) string {
	parts := []string{}
	if meta.Genre != "" {
		parts = append(parts, meta.Genre)
	}
	if style := meta.WritingStyle(); style != "" {
		parts = append(parts, style)
	}
	return strings.Join(parts, " · ")
}

func writePDFTitle(pdf *fpdf.Fpdf, title string) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "未命名"
	}
	pdf.SetFont(pdfFontName, "B", 22)
	pdf.Ln(40)
	pdf.MultiCell(0, 12, title, "", "C", false)
	pdf.Ln(8)
}

func writePDFHeading(pdf *fpdf.Fpdf, text string, size float64) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	pdf.SetFont(pdfFontName, "B", size)
	pdf.MultiCell(0, pdfLineHeight+1, text, "", "L", false)
	pdf.Ln(3)
}

func writePDFParagraph(pdf *fpdf.Fpdf, text string, size float64, bold bool, lineHeight float64) {
	text = strings.TrimSpace(text)
	if text == "" {
		pdf.Ln(lineHeight / 2)
		return
	}
	style := ""
	if bold {
		style = "B"
	}
	pdf.SetFont(pdfFontName, style, size)
	pdf.MultiCell(0, lineHeight, text, "", "L", false)
	pdf.Ln(2)
}

func writePDFMarkdownBody(pdf *fpdf.Fpdf, md string) {
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			pdf.Ln(3)
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "# "):
			writePDFHeading(pdf, strings.TrimPrefix(trimmed, "# "), 15)
		case strings.HasPrefix(trimmed, "## "):
			writePDFHeading(pdf, strings.TrimPrefix(trimmed, "## "), 13)
		case strings.HasPrefix(trimmed, "### "):
			writePDFParagraph(pdf, strings.TrimPrefix(trimmed, "### "), 12, true, pdfLineHeight)
		default:
			writePDFParagraph(pdf, trimmed, 12, false, pdfLineHeight)
		}
	}
}

func resolvePDFFont() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("NOVA_PDF_FONT")); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
		return "", fmt.Errorf("NOVA_PDF_FONT 指向的文件不存在: %s", custom)
	}

	candidates := pdfFontCandidates()
	for _, path := range candidates {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}
	return "", fmt.Errorf("未找到可用于 PDF 的中文字体（.ttf），请安装 Arial Unicode / 微软雅黑，或设置 NOVA_PDF_FONT 指向字体文件")
}

func pdfFontCandidates() []string {
	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = append(paths,
			"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
			"/Library/Fonts/Arial Unicode.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
		)
	case "windows":
		paths = append(paths,
			`C:\Windows\Fonts\msyh.ttf`,
			`C:\Windows\Fonts\msyhbd.ttf`,
			`C:\Windows\Fonts\simhei.ttf`,
			`C:\Windows\Fonts\simsun.ttf`,
			`C:\Windows\Fonts\arialuni.ttf`,
		)
	default:
		paths = append(paths,
			"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		)
	}
	// 通用候选
	paths = append(paths,
		"/usr/share/fonts/truetype/arphic/ukai.ttc",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	)
	return paths
}
