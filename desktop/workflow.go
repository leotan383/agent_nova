package main

import (
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/workflow"
)

// DailyWorkflowDTO 日更工作流报告。
type DailyWorkflowDTO struct {
	TodayWords        int               `json:"today_words"`
	TodayWordsGoal    int               `json:"today_words_goal"`
	TodayChapters     int               `json:"today_chapters"`
	TodayChaptersGoal int               `json:"today_chapters_goal"`
	GoalMetToday      bool              `json:"goal_met_today"`
	CurrentStreak     int               `json:"current_streak"`
	LongestStreak     int               `json:"longest_streak"`
	BufferCount       int               `json:"buffer_count"`
	BufferReady       int               `json:"buffer_ready"`
	BufferTarget      int               `json:"buffer_target"`
	BufferOK          bool              `json:"buffer_ok"`
	Calendar          []CalendarDayDTO  `json:"calendar"`
	Suggestions       []TodoItemDTO     `json:"suggestions"`
	Settings          WorkflowSettingsDTO `json:"settings"`
}

// CalendarDayDTO 日历单日。
type CalendarDayDTO struct {
	Date     string `json:"date"`
	Words    int    `json:"words"`
	Chapters int    `json:"chapters"`
	GoalMet  bool   `json:"goal_met"`
}

// WorkflowSettingsDTO 日更配置。
type WorkflowSettingsDTO struct {
	DailyWords    int `json:"daily_words"`
	DailyChapters int `json:"daily_chapters"`
	BufferTarget  int `json:"buffer_target"`
}

// UpdateWorkflowSettingsInput 更新日更目标。
type UpdateWorkflowSettingsInput struct {
	DailyWords    int `json:"daily_words"`
	DailyChapters int `json:"daily_chapters"`
	BufferTarget  int `json:"buffer_target"`
}

// SetChapterStatusInput 设置章节发布状态。
type SetChapterStatusInput struct {
	Chapter int    `json:"chapter"`
	Status  string `json:"status"`
}

var allowedChapterStatuses = map[string]bool{
	"draft": true, "reviewed": true, "published": true, "scheduled": true,
}

func toDailyWorkflowDTO(r workflow.DailyReport) DailyWorkflowDTO {
	cal := make([]CalendarDayDTO, len(r.Calendar))
	for i, d := range r.Calendar {
		cal[i] = CalendarDayDTO{
			Date: d.Date, Words: d.Words, Chapters: d.Chapters, GoalMet: d.GoalMet,
		}
	}
	sugs := make([]TodoItemDTO, len(r.Suggestions))
	for i, t := range r.Suggestions {
		sugs[i] = TodoItemDTO{
			ID: t.ID, Label: t.Label, Detail: t.Detail,
			Severity: t.Severity, Action: t.Action, ActionParam: t.ActionParam,
		}
	}
	return DailyWorkflowDTO{
		TodayWords: r.TodayWords, TodayWordsGoal: r.TodayWordsGoal,
		TodayChapters: r.TodayChapters, TodayChaptersGoal: r.TodayChaptersGoal,
		GoalMetToday: r.GoalMetToday,
		CurrentStreak: r.CurrentStreak, LongestStreak: r.LongestStreak,
		BufferCount: r.BufferCount, BufferReady: r.BufferReady,
		BufferTarget: r.BufferTarget, BufferOK: r.BufferOK,
		Calendar: cal, Suggestions: sugs,
		Settings: WorkflowSettingsDTO{
			DailyWords: r.Settings.DailyWords, DailyChapters: r.Settings.DailyChapters,
			BufferTarget: r.Settings.BufferTarget,
		},
	}
}

// GetDailyWorkflow 返回日更工作流与存稿缓冲状态。
func (a *App) GetDailyWorkflow() (DailyWorkflowDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return DailyWorkflowDTO{}, err
	}
	var out DailyWorkflowDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := a.syncChaptersFromDisk(actx); err != nil {
			return err
		}
		rep, err := workflow.BuildDailyReport(actx.Project, actx.Store)
		if err != nil {
			return err
		}
		out = toDailyWorkflowDTO(rep)
		return nil
	})
	return out, err
}

// UpdateWorkflowSettings 更新项目日更目标（写入 nova.yaml）。
func (a *App) UpdateWorkflowSettings(in UpdateWorkflowSettingsInput) error {
	if in.DailyChapters < 0 || in.DailyWords < 0 || in.BufferTarget < 0 {
		return fmt.Errorf("目标值不能为负数")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		actx.Project.Meta.DailyWords = in.DailyWords
		actx.Project.Meta.DailyChapters = in.DailyChapters
		actx.Project.Meta.BufferTarget = in.BufferTarget
		return actx.Project.Save()
	})
}

// SetChapterStatus 设置章节发布状态。
func (a *App) SetChapterStatus(in SetChapterStatusInput) error {
	if in.Chapter <= 0 {
		return fmt.Errorf("无效章号")
	}
	status := strings.ToLower(strings.TrimSpace(in.Status))
	if !allowedChapterStatuses[status] {
		return fmt.Errorf("无效状态: %s", in.Status)
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if err := actx.Store.SetChapterStatus(in.Chapter, status); err != nil {
			return err
		}
		if status == "published" {
			_ = workflow.RecordChapterPublished(actx.Project.Root)
		}
		return nil
	})
}

// recordWriteActivity 写章成功后记录今日活动（由 write.go 直接调用 workflow 包）。
