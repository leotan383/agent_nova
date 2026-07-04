package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tanlian/agent_nova/internal/index"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/version"
)

type RunLedger struct {
	Chapter int               `json:"chapter"`
	Steps   []RunStep         `json:"steps"`
	Updated time.Time         `json:"updated"`
}

type RunStep struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Started   time.Time `json:"started"`
	Finished  time.Time `json:"finished,omitempty"`
	Message   string    `json:"message,omitempty"`
}

func LoadLedger(path string) (*RunLedger, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &RunLedger{}, nil
		}
		return nil, err
	}
	var l RunLedger
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

func (l *RunLedger) Save(path string) error {
	l.Updated = time.Now().UTC()
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (l *RunLedger) Record(name, status, msg string) {
	l.Steps = append(l.Steps, RunStep{
		Name: name, Status: status, Started: time.Now().UTC(),
		Finished: time.Now().UTC(), Message: msg,
	})
}

func (l *RunLedger) ResumeStep() string {
	if len(l.Steps) == 0 {
		return "draft"
	}
	last := l.Steps[len(l.Steps)-1]
	if last.Status == "done" {
		switch last.Name {
		case "draft":
			return "review"
		case "review", "polish":
			return "summary"
		default:
			return "done"
		}
	}
	return last.Name
}

// IsResumable 是否存在未完成的写章流水线（可 --resume）。
func (l *RunLedger) IsResumable() bool {
	if l.Chapter <= 0 || len(l.Steps) == 0 {
		return false
	}
	last := l.Steps[len(l.Steps)-1]
	return !(last.Name == "commit" && last.Status == "done")
}

type GateStage string

const (
	GatePrewrite   GateStage = "prewrite"
	GatePrecommit  GateStage = "precommit"
	GatePostcommit GateStage = "postcommit"
)

type GateResult struct {
	OK      bool     `json:"ok"`
	Stage   string   `json:"stage"`
	Chapter int      `json:"chapter"`
	Issues  []string `json:"issues,omitempty"`
}

func RunGate(p *project.Project, st *store.Store, chapter int, stage GateStage) GateResult {
	res := GateResult{Stage: string(stage), Chapter: chapter, OK: true}
	switch stage {
	case GatePrewrite:
		if p.Meta.Phase != project.PhaseWriting && p.Meta.Phase != project.PhasePlanning {
			res.Issues = append(res.Issues, "项目 phase 未进入 writing/planning")
		}
		vol := p.Meta.CurrentVolume
		if vol <= 0 {
			vol = 1
		}
		if _, err := os.Stat(p.VolumeOutlinePath(vol)); err != nil {
			res.Issues = append(res.Issues, fmt.Sprintf("卷纲不存在: 第%02d卷", vol))
		}
		if chapter > 1 {
			if _, err := os.Stat(p.SummaryPath(chapter - 1)); err != nil {
				res.Issues = append(res.Issues, fmt.Sprintf("上一章摘要缺失: 第%d章", chapter-1))
			}
		}
		if st != nil {
			stale := st.CheckIndexStale(p.ChaptersDir())
			if stale.Stale {
				idx := index.New(p, st)
				if err := idx.RebuildChapters(0); err == nil {
					stale = st.CheckIndexStale(p.ChaptersDir())
				}
			}
			if stale.Stale {
				res.Issues = append(res.Issues, stale.Issues...)
			}
		}
	case GatePrecommit:
		matches, _ := filepath.Glob(filepath.Join(p.ChaptersDir(), fmt.Sprintf("第%03d章*.md", chapter)))
		if len(matches) == 0 {
			res.Issues = append(res.Issues, "正文文件不存在")
		}
	case GatePostcommit:
		if _, err := os.Stat(p.SummaryPath(chapter)); err != nil {
			res.Issues = append(res.Issues, "摘要未生成")
		}
		if st != nil {
			if _, err := st.GetChapter(chapter); err != nil {
				res.Issues = append(res.Issues, "章节未写入索引")
			}
		}
	}
	if len(res.Issues) > 0 {
		res.OK = false
	}
	return res
}

func SaveChapterWithVersion(p *project.Project, chapter int, title, content, source, label string) (string, error) {
	if err := version.BeforeSave(p, chapter, content, source, label); err != nil {
		return "", err
	}
	return SaveChapter(p, chapter, title, content)
}

func SaveChapter(p *project.Project, chapter int, title, content string) (string, error) {
	path := p.ChapterPath(chapter, title)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func SaveSummary(p *project.Project, chapter int, summary string) error {
	path := p.SummaryPath(chapter)
	return os.WriteFile(path, []byte(summary), 0o644)
}

func UpdateProjectProgress(p *project.Project, chapter int) error {
	if chapter > p.Meta.CurrentChapter {
		p.Meta.CurrentChapter = chapter
	}
	if p.Meta.Phase == project.PhasePlanning || p.Meta.Phase == project.PhaseInitDone {
		p.Meta.Phase = project.PhaseWriting
	}
	return p.Save()
}

// HasReviewReport 该章磁盘上是否已有非空审查报告。
func HasReviewReport(p *project.Project, chapter int) bool {
	data, err := os.ReadFile(p.ReviewPath(chapter))
	return err == nil && strings.TrimSpace(string(data)) != ""
}

// InferChapterStatus 根据审查报告等推断章节状态（draft / reviewed / published / scheduled）。
func InferChapterStatus(p *project.Project, st *store.Store, chapter int) string {
	if ch, err := st.GetChapter(chapter); err == nil {
		s := strings.ToLower(strings.TrimSpace(ch.Status))
		if s == "published" || s == "scheduled" {
			return s
		}
	}
	if HasReviewReport(p, chapter) {
		return "reviewed"
	}
	return "draft"
}

// RefreshChapterStatuses 将 DB 中章节状态与磁盘审查报告对齐。
func RefreshChapterStatuses(p *project.Project, st *store.Store) error {
	chs, err := st.ListChapters()
	if err != nil {
		return err
	}
	for _, ch := range chs {
		inferred := InferChapterStatus(p, st, ch.Number)
		if inferred == ch.Status {
			continue
		}
		ch.Status = inferred
		if err := st.UpsertChapter(ch); err != nil {
			return err
		}
	}
	return nil
}

func PostWriteIndex(p *project.Project, st *store.Store, chapter int, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	title := extractChapterTitle(filepath.Base(path))
	wordCount := utf8.RuneCountInString(string(data))
	summaryPath := p.SummaryPath(chapter)
	status := InferChapterStatus(p, st, chapter)
	_ = st.UpsertChapter(store.Chapter{
		Number: chapter, Title: title, WordCount: wordCount, Path: path,
		SummaryPath: summaryPath, Status: status, UpdatedAt: project.Timestamp(),
	})
	idx := index.New(p, st)
	return idx.RebuildChapters(chapter)
}

func extractChapterTitle(base string) string {
	base = strings.TrimSuffix(base, ".md")
	if i := strings.Index(base, "-"); i > 0 {
		return base[i+1:]
	}
	return ""
}

var chapterHeading = regexp.MustCompile(`^#\s*第?\s*(\d+)\s*章`)

func ParseChapterTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			return strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "#")), " ")
		}
		if line != "" {
			break
		}
	}
	return ""
}

func BackupChapter(p *project.Project, chapter int) error {
	_, err := version.Snapshot(p, chapter, "manual", "手动备份")
	return err
}

