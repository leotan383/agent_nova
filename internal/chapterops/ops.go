package chapterops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tanlian/agent_nova/internal/index"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/version"
)

// CascadePreview 结构变更影响预览。
type CascadePreview struct {
	Operation            string   `json:"operation"`
	TargetChapter        int      `json:"target_chapter"`
	NewChapter           int      `json:"new_chapter,omitempty"`
	Title                string   `json:"title,omitempty"`
	Impact               store.CascadeImpact `json:"impact"`
	DirsToRename         []string `json:"dirs_to_rename"`
	OpenForeshadowsAtTarget int   `json:"open_foreshadows_at_target"`
}

// InsertAfter 在指定章后插入新章（后续章号 +1）。
func InsertAfter(p *project.Project, st *store.Store, after int, title string) (int, error) {
	if after < 0 {
		return 0, fmt.Errorf("无效章号")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "新章"
	}
	insertNum := after + 1
	if err := p.MigrateChapterLayout(); err != nil {
		return 0, err
	}
	nums, err := p.ListChapterNumbers()
	if err != nil {
		return 0, err
	}
	from := insertNum
	if len(nums) > 0 {
		maxNum := nums[len(nums)-1]
		if from <= maxNum {
			if err := renameChapterDirs(p, from, +1); err != nil {
				return 0, err
			}
			if err := st.ShiftChapterReferences(from, +1); err != nil {
				return 0, err
			}
		}
	}
	dir := filepath.Join(p.ChaptersDir(), project.ChapterDirName(insertNum, title))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	bodyPath := filepath.Join(dir, project.ChapterBodyFile)
	heading := fmt.Sprintf("# 第%d章 %s\n\n", insertNum, title)
	if err := os.WriteFile(bodyPath, []byte(heading), 0o644); err != nil {
		return 0, err
	}
	if err := syncMetaCurrentChapter(p, st); err != nil {
		return 0, err
	}
	idx := index.New(p, st)
	if err := idx.RebuildChapters(0); err != nil {
		return 0, err
	}
	return insertNum, nil
}

// DeleteChapter 删除指定章并顺延后续章号。
func DeleteChapter(p *project.Project, st *store.Store, chapter int) error {
	if chapter <= 0 {
		return fmt.Errorf("无效章号")
	}
	if err := p.MigrateChapterLayout(); err != nil {
		return err
	}
	if _, _, err := p.FindChapterFile(chapter); err != nil {
		return fmt.Errorf("第 %d 章不存在", chapter)
	}
	dir, _, err := p.FindChapterDir(chapter)
	if err != nil {
		return err
	}
	if dir != "" {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	if err := st.DeleteChapterReferences(chapter); err != nil {
		return err
	}
	nums, err := p.ListChapterNumbers()
	if err != nil {
		return err
	}
	hasLater := false
	for _, n := range nums {
		if n > chapter {
			hasLater = true
			break
		}
	}
	if hasLater {
		from := chapter + 1
		if err := renameChapterDirs(p, from, -1); err != nil {
			return err
		}
		if err := st.ShiftChapterReferences(from, -1); err != nil {
			return err
		}
	}
	if err := syncMetaCurrentChapter(p, st); err != nil {
		return err
	}
	idx := index.New(p, st)
	return idx.RebuildChapters(0)
}

// PreviewInsertAfter 预览插章影响。
func PreviewInsertAfter(p *project.Project, st *store.Store, after int, title string) (CascadePreview, error) {
	insertNum := after + 1
	from := insertNum
	impact, err := st.PreviewChapterShift(from, +1)
	if err != nil {
		return CascadePreview{}, err
	}
	dirs, _ := listDirsToShift(p, from, +1)
	return CascadePreview{
		Operation: "insert", TargetChapter: after, NewChapter: insertNum,
		Title: strings.TrimSpace(title), Impact: impact, DirsToRename: dirs,
	}, nil
}

// PreviewDelete 预览删章影响。
func PreviewDelete(p *project.Project, st *store.Store, chapter int) (CascadePreview, error) {
	from := chapter + 1
	impact, err := st.PreviewChapterShift(from, -1)
	if err != nil {
		return CascadePreview{}, err
	}
	dirs, _ := listDirsToShift(p, from, -1)
	openAt := 0
	fs, _ := st.ListForeshadows("open")
	for _, f := range fs {
		if f.PlantedChapter == chapter {
			openAt++
		}
	}
	return CascadePreview{
		Operation: "delete", TargetChapter: chapter, Impact: impact,
		DirsToRename: dirs, OpenForeshadowsAtTarget: openAt,
	}, nil
}

func listDirsToShift(p *project.Project, from int, delta int) ([]string, error) {
	nums, err := p.ListChapterNumbers()
	if err != nil {
		return nil, err
	}
	var affected []int
	for _, n := range nums {
		if n >= from {
			affected = append(affected, n)
		}
	}
	if delta > 0 {
		sort.Slice(affected, func(i, j int) bool { return affected[i] > affected[j] })
	} else {
		sort.Ints(affected)
	}
	var dirs []string
	for _, n := range affected {
		dir, title, err := p.FindChapterDir(n)
		if err != nil || dir == "" {
			continue
		}
		newNum := n + delta
		dirs = append(dirs, fmt.Sprintf("%s → %s", filepath.Base(dir), project.ChapterDirName(newNum, title)))
	}
	return dirs, nil
}

func renameChapterDirs(p *project.Project, from int, delta int) error {
	nums, err := p.ListChapterNumbers()
	if err != nil {
		return err
	}
	var affected []int
	for _, n := range nums {
		if n >= from {
			affected = append(affected, n)
		}
	}
	if len(affected) == 0 {
		return nil
	}
	if delta > 0 {
		sort.Slice(affected, func(i, j int) bool { return affected[i] > affected[j] })
	} else {
		sort.Ints(affected)
	}
	for _, n := range affected {
		dir, title, err := p.FindChapterDir(n)
		if err != nil || dir == "" {
			continue
		}
		newNum := n + delta
		newDir := filepath.Join(p.ChaptersDir(), project.ChapterDirName(newNum, title))
		if err := os.Rename(dir, newDir); err != nil {
			return fmt.Errorf("重命名 %s: %w", filepath.Base(dir), err)
		}
		_ = renameVersionDir(p, n, newNum)
	}
	return nil
}

func renameVersionDir(p *project.Project, oldNum, newNum int) error {
	oldPath := filepath.Join(p.NovaDir(), "versions", fmt.Sprintf("chapter-%03d", oldNum))
	newPath := filepath.Join(p.NovaDir(), "versions", fmt.Sprintf("chapter-%03d", newNum))
	if _, err := os.Stat(oldPath); err != nil {
		return nil
	}
	return os.Rename(oldPath, newPath)
}

func syncMetaCurrentChapter(p *project.Project, st *store.Store) error {
	nums, err := p.ListChapterNumbers()
	if err != nil {
		return err
	}
	max := 0
	for _, n := range nums {
		if n > max {
			max = n
		}
	}
	p.Meta.CurrentChapter = max
	if err := p.Save(); err != nil {
		return err
	}
	_ = st.InitProject(p.Root, p.Meta)
	if max > 0 {
		_, _ = version.Snapshot(p, max, version.SourceManualEdit, "结构变更")
	}
	return nil
}
