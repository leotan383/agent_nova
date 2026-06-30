package version

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/project"
)

const (
	SourceWriteDraft  = "write_draft"
	SourceWriteReview = "write_review"
	SourceCoachApply  = "coach_apply"
	SourceRestore     = "restore"
	SourceCoachDraft  = "coach_draft"
	SourceManualEdit  = "manual_edit"

	maxVersionsPerChapter = 50
	currentVersionID      = "current"
)

// Entry 章节历史版本。
type Entry struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	Source    string `json:"source"`
	Label     string `json:"label"`
	WordCount int    `json:"word_count"`
	File      string `json:"file"`
}

type manifest struct {
	Chapter  int     `json:"chapter"`
	Versions []Entry `json:"versions"`
}

// DiffLine 单行 diff。
type DiffLine struct {
	Type string `json:"type"` // add, del, same
	Text string `json:"text"`
}

// DiffResult 两版对比结果。
type DiffResult struct {
	FromID       string     `json:"from_id"`
	ToID         string     `json:"to_id"`
	FromLabel    string     `json:"from_label"`
	ToLabel      string     `json:"to_label"`
	Lines        []DiffLine `json:"lines"`
	AddedWords   int        `json:"added_words"`
	RemovedWords int        `json:"removed_words"`
}

func chapterVersionsDir(p *project.Project, chapter int) string {
	return filepath.Join(p.NovaDir(), "versions", fmt.Sprintf("chapter-%03d", chapter))
}

func manifestPath(p *project.Project, chapter int) string {
	return filepath.Join(chapterVersionsDir(p, chapter), "manifest.json")
}

func loadManifest(p *project.Project, chapter int) (manifest, error) {
	m := manifest{Chapter: chapter, Versions: []Entry{}}
	data, err := os.ReadFile(manifestPath(p, chapter))
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	if m.Versions == nil {
		m.Versions = []Entry{}
	}
	return m, nil
}

func saveManifest(p *project.Project, chapter int, m manifest) error {
	dir := chapterVersionsDir(p, chapter)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath(p, chapter), data, 0o644)
}

func readCurrentContent(p *project.Project, chapter int) (string, error) {
	path, _, err := p.FindChapterFile(chapter)
	if err != nil {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// Snapshot 保存当前正文为历史版本（覆盖前调用）。
func Snapshot(p *project.Project, chapter int, source, label string) (Entry, error) {
	body, err := readCurrentContent(p, chapter)
	if err != nil {
		return Entry{}, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return Entry{}, nil
	}
	m, err := loadManifest(p, chapter)
	if err != nil {
		return Entry{}, err
	}
	id := fmt.Sprintf("v%03d", nextVersionNum(m.Versions)+1)
	ts := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.md", id, ts)
	dir := chapterVersionsDir(p, chapter)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Entry{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(body), 0o644); err != nil {
		return Entry{}, err
	}
	entry := Entry{
		ID: id, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Source: source, Label: labelOrDefault(label, source),
		WordCount: utf8.RuneCountInString(body), File: filename,
	}
	m.Versions = append([]Entry{entry}, m.Versions...) // newest first
	if len(m.Versions) > maxVersionsPerChapter {
		for _, old := range m.Versions[maxVersionsPerChapter:] {
			_ = os.Remove(filepath.Join(dir, old.File))
		}
		m.Versions = m.Versions[:maxVersionsPerChapter]
	}
	if err := saveManifest(p, chapter, m); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func labelOrDefault(label, source string) string {
	if label != "" {
		return label
	}
	switch source {
	case SourceWriteDraft:
		return "写章起草"
	case SourceWriteReview:
		return "审查润色"
	case SourceCoachApply:
		return "改稿应用"
	case SourceRestore:
		return "版本恢复"
	case SourceCoachDraft:
		return "改稿草稿"
	case SourceManualEdit:
		return "手动编辑"
	default:
		return source
	}
}

// List 返回章节版本（新→旧）。
func List(p *project.Project, chapter int) ([]Entry, error) {
	m, err := loadManifest(p, chapter)
	if err != nil {
		return nil, err
	}
	return m.Versions, nil
}

// GetContent 读取指定版本正文；versionID 为 current 时读当前正文。
func GetContent(p *project.Project, chapter int, versionID string) (string, error) {
	if versionID == "" || versionID == currentVersionID {
		return readCurrentContent(p, chapter)
	}
	m, err := loadManifest(p, chapter)
	if err != nil {
		return "", err
	}
	for _, e := range m.Versions {
		if e.ID == versionID {
			data, err := os.ReadFile(filepath.Join(chapterVersionsDir(p, chapter), e.File))
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("版本不存在: %s", versionID)
}

// Diff 对比两个版本；toID 可为 current。
func Diff(p *project.Project, chapter int, fromID, toID string) (DiffResult, error) {
	from, err := GetContent(p, chapter, fromID)
	if err != nil {
		return DiffResult{}, err
	}
	to, err := GetContent(p, chapter, toID)
	if err != nil {
		return DiffResult{}, err
	}
	return DiffTexts(fromID, toID, labelForID(p, chapter, fromID), labelForID(p, chapter, toID), from, to), nil
}

// DiffWithNew 当前正文 vs 待应用内容（不落盘）。
func DiffWithNew(p *project.Project, chapter int, newContent string) (DiffResult, error) {
	current, err := readCurrentContent(p, chapter)
	if err != nil {
		return DiffResult{}, err
	}
	return DiffTexts(currentVersionID, "pending", "当前正文", "修改稿", current, newContent), nil
}

func labelForID(p *project.Project, chapter int, id string) string {
	if id == "" || id == currentVersionID {
		return "当前正文"
	}
	if id == "pending" {
		return "修改稿"
	}
	m, _ := loadManifest(p, chapter)
	for _, e := range m.Versions {
		if e.ID == id {
			return fmt.Sprintf("%s · %s", e.ID, e.Label)
		}
	}
	return id
}

func DiffTexts(fromID, toID, fromLabel, toLabel, from, to string) DiffResult {
	lines := lineDiff(from, to)
	added, removed := countWordDelta(from, to)
	return DiffResult{
		FromID: fromID, ToID: toID, FromLabel: fromLabel, ToLabel: toLabel,
		Lines: lines, AddedWords: added, RemovedWords: removed,
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

func lineDiff(old, new string) []DiffLine {
	a := splitLines(old)
	b := splitLines(new)
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []DiffLine
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			out = append(out, DiffLine{Type: "same", Text: a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			out = append(out, DiffLine{Type: "del", Text: a[i]})
			i++
		} else {
			out = append(out, DiffLine{Type: "add", Text: b[j]})
			j++
		}
	}
	for i < n {
		out = append(out, DiffLine{Type: "del", Text: a[i]})
		i++
	}
	for j < m {
		out = append(out, DiffLine{Type: "add", Text: b[j]})
		j++
	}
	return out
}

func nextVersionNum(versions []Entry) int {
	max := 0
	for _, v := range versions {
		var n int
		if _, err := fmt.Sscanf(v.ID, "v%d", &n); err == nil && n > max {
			max = n
		}
	}
	return max
}

func countWordDelta(old, new string) (added, removed int) {
	oldWords := utf8.RuneCountInString(old)
	newWords := utf8.RuneCountInString(new)
	if newWords > oldWords {
		added = newWords - oldWords
	} else {
		removed = oldWords - newWords
	}
	return added, removed
}

// BeforeSave 若正文将发生变化，先快照当前版本。
func BeforeSave(p *project.Project, chapter int, newContent, source, label string) error {
	current, err := readCurrentContent(p, chapter)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current) == "" {
		return nil
	}
	if strings.TrimSpace(current) == strings.TrimSpace(newContent) {
		return nil
	}
	_, err = Snapshot(p, chapter, source, label)
	return err
}

// CurrentVersionID 表示当前正文（非历史条目）。
func CurrentVersionID() string { return currentVersionID }
