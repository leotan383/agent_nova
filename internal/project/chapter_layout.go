package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ChapterBodyFile    = "正文.md"
	ChapterReviewFile  = "审查.md"
	ChapterSummaryFile = "摘要.md"
	ChapterAICheckFile = "AI味.md"
)

// ChapterDirName 章节目录名（正文/第001章-标题/）。
func ChapterDirName(num int, title string) string {
	if strings.TrimSpace(title) == "" {
		return fmt.Sprintf("第%03d章", num)
	}
	return fmt.Sprintf("第%03d章-%s", num, sanitizeTitle(title))
}

// ParseChapterDirName 从目录名解析章号与标题。
func ParseChapterDirName(name string) (num int, title string) {
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "第") {
		return 0, ""
	}
	after := strings.TrimPrefix(name, "第")
	i := strings.Index(after, "章")
	if i < 0 {
		return 0, ""
	}
	fmt.Sscanf(after[:i], "%d", &num)
	title = strings.TrimPrefix(after[i+len("章"):], "-")
	return num, title
}

// ParseChapterFileName 从旧式平铺文件名解析（第001章-标题.md）。
func ParseChapterFileName(name string) (num int, title string) {
	base := strings.TrimSuffix(name, ".md")
	return ParseChapterDirName(base)
}

// ChapterDirFor 返回章节目录路径；若已存在同章号目录则复用。
func (p *Project) ChapterDirFor(num int, title string) string {
	if dir, _, err := p.findChapterDirRaw(num); err == nil && dir != "" {
		return dir
	}
	return filepath.Join(p.ChaptersDir(), ChapterDirName(num, title))
}

// ChapterPath 正文文件路径（正文/第NNN章-标题/正文.md）。
func (p *Project) ChapterPath(num int, title string) string {
	return filepath.Join(p.ChapterDirFor(num, title), ChapterBodyFile)
}

// ReviewPath 审查文件路径。
func (p *Project) ReviewPath(num int) string {
	dir, _, err := p.findChapterDirRaw(num)
	if err != nil || dir == "" {
		dir = filepath.Join(p.ChaptersDir(), fmt.Sprintf("第%03d章", num))
	}
	return filepath.Join(dir, ChapterReviewFile)
}

// SummaryPath 摘要文件路径。
func (p *Project) SummaryPath(num int) string {
	dir, _, err := p.findChapterDirRaw(num)
	if err != nil || dir == "" {
		dir = filepath.Join(p.ChaptersDir(), fmt.Sprintf("第%03d章", num))
	}
	return filepath.Join(dir, ChapterSummaryFile)
}

// AICheckPath AI 味检测报告路径。
func (p *Project) AICheckPath(num int) string {
	dir, _, err := p.findChapterDirRaw(num)
	if err != nil || dir == "" {
		dir = filepath.Join(p.ChaptersDir(), fmt.Sprintf("第%03d章", num))
	}
	return filepath.Join(dir, ChapterAICheckFile)
}

// ReviewsDir 旧版顶层审查目录（仅迁移用）。
func (p *Project) ReviewsDir() string { return filepath.Join(p.Root, "审查") }

// SummariesDir 旧版顶层摘要目录（仅迁移用）。
func (p *Project) SummariesDir() string { return filepath.Join(p.Root, "摘要") }

// FindChapterDir 查找章节目录（第 N 章）。
func (p *Project) FindChapterDir(number int) (dirPath, title string, err error) {
	return p.findChapterDirRaw(number)
}

func (p *Project) findChapterDirRaw(number int) (dirPath, title string, err error) {
	prefix := fmt.Sprintf("第%03d章", number)
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return "", "", err
	}
	var bestDir string
	var bestTitle string
	var bestMod time.Time
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(p.ChaptersDir(), e.Name())
		if bestDir == "" || info.ModTime().After(bestMod) {
			bestDir = full
			_, bestTitle = ParseChapterDirName(e.Name())
			bestMod = info.ModTime()
		}
	}
	if bestDir != "" {
		return bestDir, bestTitle, nil
	}
	return "", "", nil
}

// FindChapterFile 查找章节正文路径。
func (p *Project) FindChapterFile(number int) (path, title string, err error) {
	if err := p.MigrateChapterLayout(); err != nil {
		return "", "", err
	}
	dir, title, err := p.findChapterDirRaw(number)
	if err != nil {
		return "", "", err
	}
	if dir != "" {
		body := filepath.Join(dir, ChapterBodyFile)
		if _, err := os.Stat(body); err == nil {
			return body, title, nil
		}
	}
	prefix := fmt.Sprintf("第%03d", number)
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return "", "", err
	}
	var bestPath string
	var bestTitle string
	var bestSize int64
	var bestMod time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		full := filepath.Join(p.ChaptersDir(), e.Name())
		if bestPath == "" || info.ModTime().After(bestMod) ||
			(info.ModTime().Equal(bestMod) && info.Size() > bestSize) {
			bestPath = full
			_, bestTitle = ParseChapterFileName(e.Name())
			bestSize = info.Size()
			bestMod = info.ModTime()
		}
	}
	if bestPath == "" {
		return "", "", fmt.Errorf("第 %d 章正文不存在", number)
	}
	return bestPath, bestTitle, nil
}

// ListChapterNumbers 列出磁盘上所有章号。
func (p *Project) ListChapterNumbers() ([]int, error) {
	if err := p.MigrateChapterLayout(); err != nil {
		return nil, err
	}
	seen := map[int]struct{}{}
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			num, _ := ParseChapterDirName(e.Name())
			if num > 0 {
				seen[num] = struct{}{}
			}
			continue
		}
		if strings.HasSuffix(e.Name(), ".md") {
			num, _ := ParseChapterFileName(e.Name())
			if num > 0 {
				seen[num] = struct{}{}
			}
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// MigrateChapterLayout 将平铺正文与顶层审查/摘要迁入按章子目录（幂等）。
func (p *Project) MigrateChapterLayout() error {
	if err := os.MkdirAll(p.ChaptersDir(), 0o755); err != nil {
		return err
	}
	if err := p.migrateFlatBodyFiles(); err != nil {
		return err
	}
	if err := p.migrateLegacyReviewFiles(); err != nil {
		return err
	}
	return p.migrateLegacySummaryFiles()
}

func (p *Project) migrateFlatBodyFiles() error {
	entries, err := os.ReadDir(p.ChaptersDir())
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		num, title := ParseChapterFileName(e.Name())
		if num <= 0 {
			continue
		}
		dir := filepath.Join(p.ChaptersDir(), ChapterDirName(num, title))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(dir, ChapterBodyFile)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		src := filepath.Join(p.ChaptersDir(), e.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func (p *Project) migrateLegacyReviewFiles() error {
	legacyDir := p.ReviewsDir()
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".review.md") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".review.md")
		num, title := ParseChapterDirName(base)
		if num <= 0 {
			continue
		}
		dir, _, _ := p.findChapterDirRaw(num)
		if dir == "" {
			dir = filepath.Join(p.ChaptersDir(), ChapterDirName(num, title))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		dst := filepath.Join(dir, ChapterReviewFile)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		src := filepath.Join(legacyDir, e.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return removeDirIfEmpty(legacyDir)
}

func (p *Project) migrateLegacySummaryFiles() error {
	legacyDir := p.SummariesDir()
	entries, err := os.ReadDir(legacyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".summary.md") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".summary.md")
		num, title := ParseChapterDirName(base)
		if num <= 0 {
			continue
		}
		dir, _, _ := p.findChapterDirRaw(num)
		if dir == "" {
			dir = filepath.Join(p.ChaptersDir(), ChapterDirName(num, title))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
		}
		dst := filepath.Join(dir, ChapterSummaryFile)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		src := filepath.Join(legacyDir, e.Name())
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return removeDirIfEmpty(legacyDir)
}

func removeDirIfEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}
