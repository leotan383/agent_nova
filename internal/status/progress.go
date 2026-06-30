package status

import (
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// Progress 创作进度与目标追踪。
type Progress struct {
	WrittenWords           int     `json:"written_words"`
	TargetWords            int     `json:"target_words"`
	ChapterWordsGoal       int     `json:"chapter_words_goal"`
	ProgressPercent        float64 `json:"progress_percent"`
	RemainingWords         int     `json:"remaining_words"`
	EstimatedTotalChapters int     `json:"estimated_total_chapters"`
	RemainingChapters      int     `json:"remaining_chapters"`
	AvgWordsPerChapter     int     `json:"avg_words_per_chapter"`
	Style                  string  `json:"style,omitempty"`
}

func ComputeProgress(meta project.Meta, chapters []store.Chapter) Progress {
	target := meta.TargetWords
	if target <= 0 {
		target = project.DefaultTargetWords
	}
	chapterGoal := meta.ChapterWords
	if chapterGoal <= 0 {
		chapterGoal = project.DefaultChapterWords
	}

	written := 0
	for _, c := range chapters {
		written += c.WordCount
	}

	avg := 0
	if len(chapters) > 0 && written > 0 {
		avg = written / len(chapters)
	}

	remaining := target - written
	if remaining < 0 {
		remaining = 0
	}

	percent := 0.0
	if target > 0 {
		percent = float64(written) / float64(target) * 100
		if percent > 100 {
			percent = 100
		}
	}

	estTotal := (target + chapterGoal - 1) / chapterGoal
	if estTotal < 1 {
		estTotal = 1
	}

	remainingChapters := 0
	if remaining > 0 {
		perChapter := chapterGoal
		if avg > 0 {
			perChapter = avg
		}
		remainingChapters = (remaining + perChapter - 1) / perChapter
	}

	return Progress{
		WrittenWords:           written,
		TargetWords:            target,
		ChapterWordsGoal:       chapterGoal,
		ProgressPercent:        percent,
		RemainingWords:         remaining,
		EstimatedTotalChapters: estTotal,
		RemainingChapters:      remainingChapters,
		AvgWordsPerChapter:     avg,
		Style:                  meta.WritingStyle(),
	}
}
