package outline

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

const (
	MatchUnwritten  = "unwritten"
	MatchMatched    = "matched"
	MatchDeviated   = "deviated"
	MatchAbandoned  = "abandoned"
)

// Row 大纲与正文对照行。
type Row struct {
	Volume           int    `json:"volume"`
	Chapter          int    `json:"chapter"`
	Title            string `json:"title"`
	OutlinePreview   string `json:"outline_preview"`
	PlanStatus       string `json:"plan_status"`
	PlanStatusNote   string `json:"plan_status_note,omitempty"`
	MatchStatus      string `json:"match_status"`
	HasBody          bool   `json:"has_body"`
	WordCount        int    `json:"word_count"`
	BodyStatus       string `json:"body_status,omitempty"`
	InOutline        bool   `json:"in_outline"`
}

// Summary 对照矩阵汇总。
type Summary struct {
	TotalInOutline int `json:"total_in_outline"`
	Written        int `json:"written"`
	Unwritten      int `json:"unwritten"`
	Deviated       int `json:"deviated"`
	Abandoned      int `json:"abandoned"`
}

// Matrix 卷纲 ↔ 正文对照结果。
type Matrix struct {
	Volume int     `json:"volume"`
	Rows   []Row   `json:"rows"`
	Summary Summary `json:"summary"`
}

// ListVolumeNumbers 扫描大纲目录中的卷号。
func ListVolumeNumbers(p *project.Project) ([]int, error) {
	entries, err := os.ReadDir(p.OutlineDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []int{1}, nil
		}
		return nil, err
	}
	seen := map[int]struct{}{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		var vol int
		if _, err := fmt.Sscanf(strings.TrimSuffix(e.Name(), ".md"), "第%d卷", &vol); err == nil && vol > 0 {
			seen[vol] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return []int{1}, nil
	}
	out := make([]int, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Ints(out)
	return out, nil
}

// BuildMatrix 构建指定卷的对照矩阵。
func BuildMatrix(p *project.Project, st *store.Store, volume int) (Matrix, error) {
	if volume <= 0 {
		volume = 1
	}
	path := p.VolumeOutlinePath(volume)
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return Matrix{}, err
	}
	entries := ParseVolumeOutline(volume, string(data))
	entryByChapter := map[int]Entry{}
	for _, e := range entries {
		entryByChapter[e.Chapter] = e
	}

	chMap := map[int]store.Chapter{}
	if st != nil {
		chs, err := st.ListChapters()
		if err != nil {
			return Matrix{}, err
		}
		for _, ch := range chs {
			chMap[ch.Number] = ch
		}
	}
	bodyChapters, err := p.ListChapterNumbers()
	if err != nil {
		return Matrix{}, err
	}
	bodySet := map[int]struct{}{}
	for _, n := range bodyChapters {
		bodySet[n] = struct{}{}
	}

	var rows []Row
	var summary Summary

	addRow := func(row Row) {
		rows = append(rows, row)
		switch row.MatchStatus {
		case MatchMatched:
			summary.Written++
		case MatchUnwritten:
			summary.Unwritten++
		case MatchDeviated:
			summary.Deviated++
		case MatchAbandoned:
			summary.Abandoned++
		}
		if row.InOutline {
			summary.TotalInOutline++
		}
	}

	for _, e := range entries {
		_, hasBody := bodySet[e.Chapter]
		row := rowFromEntry(e, chMap[e.Chapter], hasBody)
		addRow(row)
	}

	if len(entries) == 0 {
		vols, mins, maxs, err := loadVolumeOutlineBounds(p)
		if err != nil {
			return Matrix{}, err
		}
		for _, row := range synthesizeRowsForEmptyOutline(volume, vols, mins, maxs, bodySet, chMap) {
			addRow(row)
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Chapter < rows[j].Chapter })
	return Matrix{Volume: volume, Rows: rows, Summary: summary}, nil
}

func rowFromEntry(e Entry, ch store.Chapter, hasBody bool) Row {
	match := MatchUnwritten
	switch e.PlanStatus {
	case "abandoned":
		match = MatchAbandoned
	case "deviated":
		match = MatchDeviated
	default:
		if hasBody {
			match = MatchMatched
		} else {
			match = MatchUnwritten
		}
	}
	if hasBody && e.PlanStatus == "done" {
		match = MatchMatched
	}
	title := e.Title
	if title == "" {
		title = ch.Title
	}
	return Row{
		Volume: e.Volume, Chapter: e.Chapter, Title: title,
		OutlinePreview: e.Preview, PlanStatus: e.PlanStatus,
		PlanStatusNote: e.PlanStatusNote, MatchStatus: match,
		HasBody: hasBody, WordCount: ch.WordCount, BodyStatus: ch.Status,
		InOutline: true,
	}
}
