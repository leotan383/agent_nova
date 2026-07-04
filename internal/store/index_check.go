package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CheckIndexStale compares 正文目录与 chapters 表、FTS 是否一致（支持按章子目录）。
func (s *Store) CheckIndexStale(chaptersDir string) IndexStaleReport {
	rep := IndexStaleReport{}
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		rep.Stale = true
		rep.Issues = append(rep.Issues, "无法读取正文目录")
		return rep
	}
	fileNums := map[int]struct{}{}
	rawFiles := 0
	for _, e := range entries {
		if e.IsDir() {
			num := parseChapterNumFromName(e.Name())
			if num <= 0 {
				continue
			}
			bodyPath := filepath.Join(chaptersDir, e.Name(), "正文.md")
			if _, err := os.Stat(bodyPath); err != nil {
				continue
			}
			rawFiles++
			fileNums[num] = struct{}{}
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		num := parseChapterNumFromName(e.Name())
		if num <= 0 {
			continue
		}
		rawFiles++
		fileNums[num] = struct{}{}
	}
	// 按章号去重（同章多文件如 第001章.md 与 第001章-标题.md 只算一章）
	rep.FileCount = len(fileNums)
	indexed, _ := s.ListChapters()
	rep.IndexCount = len(indexed)
	indexByNum := map[int]Chapter{}
	for _, ch := range indexed {
		indexByNum[ch.Number] = ch
	}
	for num := range fileNums {
		ch, ok := indexByNum[num]
		if !ok {
			rep.Stale = true
			rep.Issues = append(rep.Issues, fmt.Sprintf("第%d章未写入 chapters 索引", num))
			continue
		}
		if ch.Path == "" || ch.WordCount == 0 {
			rep.Stale = true
			rep.Issues = append(rep.Issues, fmt.Sprintf("第%d章索引不完整", num))
			continue
		}
		fi, err := os.Stat(ch.Path)
		if err != nil {
			rep.Stale = true
			rep.Issues = append(rep.Issues, fmt.Sprintf("第%d章索引路径无效", num))
			continue
		}
		updated, err := time.Parse(time.RFC3339, ch.UpdatedAt)
		if err != nil {
			rep.Stale = true
			rep.Issues = append(rep.Issues, fmt.Sprintf("第%d章索引时间戳无效", num))
			continue
		}
		fileMod := fi.ModTime().UTC().Truncate(time.Second)
		indexMod := updated.Truncate(time.Second)
		if fileMod.After(indexMod) {
			rep.Stale = true
			rep.Issues = append(rep.Issues, fmt.Sprintf("第%d章正文已修改，索引过期", num))
		}
	}
	cFTS, _, _ := s.FTSStats()
	rep.FTSCount = cFTS
	if rep.FileCount > 0 && cFTS < rep.FileCount {
		rep.Stale = true
		rep.Issues = append(rep.Issues, fmt.Sprintf("FTS 章节数(%d)少于已写章节数(%d)，请 nova index rebuild", cFTS, rep.FileCount))
	}
	if rawFiles > rep.FileCount {
		// 同章多文件仅提示，不单独阻断 gate（去重后应能通过）
		_ = rawFiles
	}
	return rep
}

func parseChapterNumFromName(name string) int {
	var n int
	for _, r := range name {
		if r >= '0' && r <= '9' {
			n = n*10 + int(r-'0')
		} else if n > 0 {
			break
		}
	}
	return n
}
