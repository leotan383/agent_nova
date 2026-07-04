package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/status"
	"github.com/tanlian/agent_nova/internal/store"
)

// CalendarDay 日历单日摘要。
type CalendarDay struct {
	Date      string `json:"date"`
	Words     int    `json:"words"`
	Chapters  int    `json:"chapters"`
	GoalMet   bool   `json:"goal_met"`
}

// DailyReport 日更工作流汇总。
type DailyReport struct {
	TodayWords        int                `json:"today_words"`
	TodayWordsGoal    int                `json:"today_words_goal"`
	TodayChapters     int                `json:"today_chapters"`
	TodayChaptersGoal int                `json:"today_chapters_goal"`
	GoalMetToday      bool               `json:"goal_met_today"`
	CurrentStreak     int                `json:"current_streak"`
	LongestStreak     int                `json:"longest_streak"`
	BufferCount       int                `json:"buffer_count"`
	BufferReady       int                `json:"buffer_ready"`
	BufferTarget      int                `json:"buffer_target"`
	BufferOK          bool               `json:"buffer_ok"`
	Calendar          []CalendarDay      `json:"calendar"`
	Suggestions       []status.TodoItem  `json:"suggestions"`
	Settings          DailySettings      `json:"settings"`
}

// DailySettings 当前项目日更配置。
type DailySettings struct {
	DailyWords    int `json:"daily_words"`
	DailyChapters int `json:"daily_chapters"`
	BufferTarget  int `json:"buffer_target"`
}

// BuildDailyReport 生成日更工作流报告。
func BuildDailyReport(p *project.Project, st *store.Store) (DailyReport, error) {
	meta := p.Meta
	wordsGoal := meta.DailyWordsGoal()
	chaptersGoal := meta.DailyChaptersGoal()
	bufferTarget := meta.BufferTargetChapters()

	log, err := LoadActivity(p.Root)
	if err != nil {
		return DailyReport{}, err
	}

	today := todayKey()
	day := log.Days[today]
	goalMet := GoalMet(day, wordsGoal, chaptersGoal)
	currentStreak, longestStreak := ComputeStreak(log, wordsGoal, chaptersGoal)

	bufferCount := 0
	bufferReady := 0
	if st != nil {
		chs, err := st.ListChapters()
		if err != nil {
			return DailyReport{}, err
		}
		for _, ch := range chs {
			s := strings.ToLower(strings.TrimSpace(ch.Status))
			if s != "published" {
				bufferCount++
			}
			if s == "reviewed" {
				bufferReady++
			}
		}
	}

	rep := status.Build(p, st, "all")
	nextCh := rep.CurrentChapter + 1
	if nextCh <= 0 {
		nextCh = 1
	}

	var suggestions []status.TodoItem
	if !goalMet {
		suggestions = append(suggestions, status.TodoItem{
			ID:       "daily_write",
			Label:    fmt.Sprintf("写第 %d 章", nextCh),
			Detail:   fmt.Sprintf("今日目标：%d 章 / %d 字", chaptersGoal, wordsGoal),
			Severity: "info",
			Action:   "open_write",
		})
	} else {
		suggestions = append(suggestions, status.TodoItem{
			ID:       "daily_goal_done",
			Label:    "今日目标已达成",
			Detail:   "继续保持连载节奏",
			Severity: "info",
			Action:   "open_write",
		})
	}

	if rep.CurrentChapter > 0 {
		reviewCh := rep.CurrentChapter
		if ch, err := st.GetChapter(reviewCh); err == nil && strings.EqualFold(ch.Status, "draft") {
			suggestions = append(suggestions, status.TodoItem{
				ID:          fmt.Sprintf("daily_review_%d", reviewCh),
				Label:       fmt.Sprintf("审查第 %d 章", reviewCh),
				Detail:      "存稿入库前建议完成审查",
				Severity:    "warn",
				Action:      "review_chapter",
				ActionParam: fmt.Sprint(reviewCh),
			})
		}
	}

	if bufferReady > 0 && bufferReady >= bufferTarget {
		suggestions = append(suggestions, status.TodoItem{
			ID:       "buffer_publish",
			Label:    fmt.Sprintf("存稿充足（%d 章已审）", bufferReady),
			Detail:   "可将已审章节标记为「发布」",
			Severity: "info",
			Action:   "open_chapters",
		})
	} else if bufferReady < bufferTarget && bufferCount > 0 {
		need := bufferTarget - bufferReady
		suggestions = append(suggestions, status.TodoItem{
			ID:       "buffer_build",
			Label:    fmt.Sprintf("存稿缓冲 %d/%d 章", bufferReady, bufferTarget),
			Detail:   fmt.Sprintf("建议再备 %d 章已审存稿", need),
			Severity: "warn",
			Action:   "open_write",
		})
	}

	calendar := buildCalendar(log, wordsGoal, chaptersGoal, 14)

	return DailyReport{
		TodayWords: day.Words, TodayWordsGoal: wordsGoal,
		TodayChapters: day.ChaptersWritten, TodayChaptersGoal: chaptersGoal,
		GoalMetToday: goalMet,
		CurrentStreak: currentStreak, LongestStreak: longestStreak,
		BufferCount: bufferCount, BufferReady: bufferReady,
		BufferTarget: bufferTarget, BufferOK: bufferReady >= bufferTarget,
		Calendar: calendar, Suggestions: suggestions,
		Settings: DailySettings{
			DailyWords: meta.DailyWords, DailyChapters: meta.DailyChapters,
			BufferTarget: meta.BufferTarget,
		},
	}, nil
}

func buildCalendar(log ActivityLog, wordsGoal, chaptersGoal, days int) []CalendarDay {
	out := make([]CalendarDay, 0, days)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		key := now.AddDate(0, 0, -i).Format("2006-01-02")
		d := log.Days[key]
		out = append(out, CalendarDay{
			Date: key, Words: d.Words, Chapters: d.ChaptersWritten,
			GoalMet: GoalMet(d, wordsGoal, chaptersGoal),
		})
	}
	return out
}
