package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// DayActivity 单日写作记录。
type DayActivity struct {
	Words             int `json:"words"`
	ChaptersWritten   int `json:"chapters_written"`
	ChaptersPublished int `json:"chapters_published"`
}

// ActivityLog 项目写作活动日志（.nova/writing_activity.json）。
type ActivityLog struct {
	Days           map[string]DayActivity `json:"days"`
	CurrentStreak  int                    `json:"current_streak"`
	LongestStreak  int                    `json:"longest_streak"`
	LastActiveDate string                 `json:"last_active_date"`
}

func activityPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".nova", "writing_activity.json")
}

func todayKey() string {
	return time.Now().Format("2006-01-02")
}

// LoadActivity 读取活动日志。
func LoadActivity(projectRoot string) (ActivityLog, error) {
	data, err := os.ReadFile(activityPath(projectRoot))
	if err != nil {
		if os.IsNotExist(err) {
			return ActivityLog{Days: map[string]DayActivity{}}, nil
		}
		return ActivityLog{}, err
	}
	var log ActivityLog
	if err := json.Unmarshal(data, &log); err != nil {
		return ActivityLog{}, err
	}
	if log.Days == nil {
		log.Days = map[string]DayActivity{}
	}
	return log, nil
}

func saveActivity(projectRoot string, log ActivityLog) error {
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(projectRoot, ".nova")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(activityPath(projectRoot), data, 0o644)
}

// RecordChapterWritten 累加今日写作字数与章数。
func RecordChapterWritten(projectRoot string, wordCount int) error {
	log, err := LoadActivity(projectRoot)
	if err != nil {
		return err
	}
	key := todayKey()
	day := log.Days[key]
	day.Words += wordCount
	day.ChaptersWritten++
	log.Days[key] = day
	log.LastActiveDate = key
	return saveActivity(projectRoot, log)
}

// RecordChapterPublished 累加今日发布章数。
func RecordChapterPublished(projectRoot string) error {
	log, err := LoadActivity(projectRoot)
	if err != nil {
		return err
	}
	key := todayKey()
	day := log.Days[key]
	day.ChaptersPublished++
	log.Days[key] = day
	log.LastActiveDate = key
	return saveActivity(projectRoot, log)
}

// GoalMet 判断某日是否达成目标。
func GoalMet(day DayActivity, wordsGoal, chaptersGoal int) bool {
	if chaptersGoal > 0 && day.ChaptersWritten >= chaptersGoal {
		return true
	}
	if wordsGoal > 0 && day.Words >= wordsGoal {
		return true
	}
	return false
}

// ComputeStreak 根据当前目标重算连续写作天数。
func ComputeStreak(log ActivityLog, wordsGoal, chaptersGoal int) (current, longest int) {
	if len(log.Days) == 0 {
		return 0, log.LongestStreak
	}
	longest = 0
	run := 0
	// 从今天或昨天开始（今日进行中仍算 streak 延续）
	start := time.Now()
	today := start.Format("2006-01-02")
	if d, ok := log.Days[today]; !ok || !GoalMet(d, wordsGoal, chaptersGoal) {
		start = start.AddDate(0, 0, -1)
	}
	for i := 0; i < 366; i++ {
		key := start.AddDate(0, 0, -i).Format("2006-01-02")
		d, ok := log.Days[key]
		if !ok || !GoalMet(d, wordsGoal, chaptersGoal) {
			break
		}
		run++
	}
	current = run
	// longest: scan all days
	dates := sortedDates(log.Days)
	run = 0
	for i, key := range dates {
		d := log.Days[key]
		if GoalMet(d, wordsGoal, chaptersGoal) {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
		_ = i
	}
	if longest < current {
		longest = current
	}
	return current, longest
}

func sortedDates(days map[string]DayActivity) []string {
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
