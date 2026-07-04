package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/chapterops"
	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/outline"
	"github.com/tanlian/agent_nova/internal/workflows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventBookReadStatus = "bookread:status"
	eventBookReadDone   = "bookread:done"
	eventBookReadError  = "bookread:error"
	eventPolishStatus   = "polish:status"
	eventPolishProgress = "polish:progress"
	eventPolishDone     = "polish:done"
	eventPolishError    = "polish:error"
)

// OutlineChapterRowDTO 对照矩阵行。
type OutlineChapterRowDTO struct {
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

// OutlineMatrixSummaryDTO 对照汇总。
type OutlineMatrixSummaryDTO struct {
	TotalInOutline int `json:"total_in_outline"`
	Written        int `json:"written"`
	Unwritten      int `json:"unwritten"`
	Deviated       int `json:"deviated"`
	Abandoned      int `json:"abandoned"`
	Orphan         int `json:"orphan,omitempty"`
}

// OutlineMatrixDTO 对照矩阵。
type OutlineMatrixDTO struct {
	Volume  int                      `json:"volume"`
	Rows    []OutlineChapterRowDTO   `json:"rows"`
	Summary OutlineMatrixSummaryDTO  `json:"summary"`
}

// CascadeImpactDTO 级联影响。
type CascadeImpactDTO struct {
	ChaptersShifted       int `json:"chapters_shifted"`
	MemoriesAffected      int `json:"memories_affected"`
	ForeshadowsAffected   int `json:"foreshadows_affected"`
	EntitiesAffected      int `json:"entities_affected"`
	EntityHistoryAffected int `json:"entity_history_affected"`
	ReviewsAffected       int `json:"reviews_affected"`
	CoolPointsAffected    int `json:"cool_points_affected"`
}

// ChapterStructurePreviewDTO 插删章预览。
type ChapterStructurePreviewDTO struct {
	Operation                 string           `json:"operation"`
	TargetChapter             int              `json:"target_chapter"`
	NewChapter                int              `json:"new_chapter,omitempty"`
	Title                     string           `json:"title,omitempty"`
	Impact                    CascadeImpactDTO `json:"impact"`
	DirsToRename              []string         `json:"dirs_to_rename"`
	OpenForeshadowsAtTarget   int              `json:"open_foreshadows_at_target"`
}

// InsertChapterInput 插章请求。
type InsertChapterInput struct {
	AfterChapter int    `json:"after_chapter"`
	Title        string `json:"title"`
}

// DeleteChapterInput 删章请求。
type DeleteChapterInput struct {
	Chapter int `json:"chapter"`
}

// StartBookReadInput 通读报告请求。
type StartBookReadInput struct {
	FromChapter int    `json:"from_chapter"`
	ToChapter   int    `json:"to_chapter"`
	Focus       string `json:"focus"`
}

// BookReadJobInfo 通读任务。
type BookReadJobInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// BookReadItemDTO 通读条目。
type BookReadItemDTO struct {
	Category   string `json:"category"`
	Severity   string `json:"severity"`
	Chapter    int    `json:"chapter"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt"`
	Suggestion string `json:"suggestion"`
}

// BookReadReportDTO 通读报告。
type BookReadReportDTO struct {
	Summary string            `json:"summary"`
	Items   []BookReadItemDTO `json:"items"`
}

// StartBatchPolishInput 批量润色请求。
type StartBatchPolishInput struct {
	Chapters []int  `json:"chapters"`
	Rule     string `json:"rule"`
}

// BatchPolishJobInfo 润色任务。
type BatchPolishJobInfo struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// BatchPolishChapterResultDTO 单章润色结果。
type BatchPolishChapterResultDTO struct {
	Chapter  int    `json:"chapter"`
	Title    string `json:"title"`
	Original string `json:"original"`
	Polished string `json:"polished"`
	Error    string `json:"error,omitempty"`
}

// BatchPolishReportDTO 批量润色报告。
type BatchPolishReportDTO struct {
	Rule    string                        `json:"rule"`
	Results []BatchPolishChapterResultDTO `json:"results"`
}

type bookEditJob struct {
	kind   string
	status string
	result any
	errMsg string
	cancel context.CancelFunc
}

type bookEditManager struct {
	app  *App
	mu   sync.Mutex
	jobs map[string]*bookEditJob
}

func newBookEditManager(app *App) *bookEditManager {
	return &bookEditManager{app: app, jobs: map[string]*bookEditJob{}}
}

func toMatrixDTO(m outline.Matrix) OutlineMatrixDTO {
	rows := make([]OutlineChapterRowDTO, len(m.Rows))
	for i, r := range m.Rows {
		rows[i] = OutlineChapterRowDTO{
			Volume: r.Volume, Chapter: r.Chapter, Title: r.Title,
			OutlinePreview: r.OutlinePreview, PlanStatus: r.PlanStatus,
			PlanStatusNote: r.PlanStatusNote, MatchStatus: r.MatchStatus,
			HasBody: r.HasBody, WordCount: r.WordCount, BodyStatus: r.BodyStatus,
			InOutline: r.InOutline,
		}
	}
	return OutlineMatrixDTO{
		Volume: m.Volume, Rows: rows,
		Summary: OutlineMatrixSummaryDTO{
			TotalInOutline: m.Summary.TotalInOutline, Written: m.Summary.Written,
			Unwritten: m.Summary.Unwritten, Deviated: m.Summary.Deviated,
			Abandoned: m.Summary.Abandoned,
		},
	}
}

func toPreviewDTO(p chapterops.CascadePreview) ChapterStructurePreviewDTO {
	return ChapterStructurePreviewDTO{
		Operation: p.Operation, TargetChapter: p.TargetChapter, NewChapter: p.NewChapter,
		Title: p.Title,
		Impact: CascadeImpactDTO{
			ChaptersShifted: p.Impact.ChaptersShifted, MemoriesAffected: p.Impact.MemoriesAffected,
			ForeshadowsAffected: p.Impact.ForeshadowsAffected, EntitiesAffected: p.Impact.EntitiesAffected,
			EntityHistoryAffected: p.Impact.EntityHistoryAffected, ReviewsAffected: p.Impact.ReviewsAffected,
			CoolPointsAffected: p.Impact.CoolPointsAffected,
		},
		DirsToRename: p.DirsToRename, OpenForeshadowsAtTarget: p.OpenForeshadowsAtTarget,
	}
}

// ListOutlineVolumes 列出有大纲文件的卷号。
func (a *App) ListOutlineVolumes() ([]int, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	var out []int
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		vols, err := outline.ListVolumeNumbers(actx.Project)
		if err != nil {
			return err
		}
		out = vols
		return nil
	})
	return out, err
}

// GetOutlineChapterMatrix 卷纲 ↔ 正文对照矩阵。
func (a *App) GetOutlineChapterMatrix(volume int) (OutlineMatrixDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return OutlineMatrixDTO{}, err
	}
	var out OutlineMatrixDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		m, err := outline.BuildMatrix(actx.Project, actx.Store, volume)
		if err != nil {
			return err
		}
		out = toMatrixDTO(m)
		return nil
	})
	return out, err
}

// PreviewInsertChapter 预览插章影响。
func (a *App) PreviewInsertChapter(in InsertChapterInput) (ChapterStructurePreviewDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return ChapterStructurePreviewDTO{}, err
	}
	var out ChapterStructurePreviewDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		preview, err := chapterops.PreviewInsertAfter(actx.Project, actx.Store, in.AfterChapter, in.Title)
		if err != nil {
			return err
		}
		out = toPreviewDTO(preview)
		return nil
	})
	return out, err
}

// PreviewDeleteChapter 预览删章影响。
func (a *App) PreviewDeleteChapter(chapter int) (ChapterStructurePreviewDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return ChapterStructurePreviewDTO{}, err
	}
	var out ChapterStructurePreviewDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		preview, err := chapterops.PreviewDelete(actx.Project, actx.Store, chapter)
		if err != nil {
			return err
		}
		out = toPreviewDTO(preview)
		return nil
	})
	return out, err
}

// InsertChapterAfter 在指定章后插入新章。
func (a *App) InsertChapterAfter(in InsertChapterInput) (int, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return 0, err
	}
	var newNum int
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if a.IsWriteRunning() || a.IsPlanRunning() {
			return fmt.Errorf("写章或卷纲任务进行中，请稍后再调整结构")
		}
		num, err := chapterops.InsertAfter(actx.Project, actx.Store, in.AfterChapter, in.Title)
		if err != nil {
			return err
		}
		newNum = num
		return nil
	})
	if err == nil {
		a.session.invalidate()
	}
	return newNum, err
}

// DeleteChapter 删除章节并顺延。
func (a *App) DeleteChapter(in DeleteChapterInput) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		if a.IsWriteRunning() || a.IsPlanRunning() {
			return fmt.Errorf("写章或卷纲任务进行中，请稍后再调整结构")
		}
		return chapterops.DeleteChapter(actx.Project, actx.Store, in.Chapter)
	})
	if err == nil {
		a.session.invalidate()
	}
	return err
}

// StartBookReadReport 异步生成全书通读报告。
func (a *App) StartBookReadReport(in StartBookReadInput) (BookReadJobInfo, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return BookReadJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return BookReadJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return BookReadJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return BookReadJobInfo{}, err
	}
	a.bookEdit.mu.Lock()
	for id, j := range a.bookEdit.jobs {
		if j.kind == "bookread" && (j.status == "running" || j.status == "pending") {
			a.bookEdit.mu.Unlock()
			return BookReadJobInfo{}, fmt.Errorf("已有通读任务进行中: %s", id)
		}
	}
	id := fmt.Sprintf("bookread-%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	a.bookEdit.jobs[id] = &bookEditJob{kind: "bookread", status: "pending", cancel: cancel}
	a.bookEdit.mu.Unlock()
	a.emitBookReadStatus(id, "pending", "")

	go func(projectRoot string, input StartBookReadInput) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failBookReadJob(id, err.Error())
			return
		}
		defer actx.Close()
		a.emitBookReadStatus(id, "running", "正在通读全书…")
		wf := workflows.NewBookReadWorkflow(actx.Config, actx.Project, actx.Store)
		report, err := wf.Run(ctx, actx.Project, actx.Store, workflows.BookReadOptions{
			FromChapter: input.FromChapter, ToChapter: input.ToChapter, Focus: input.Focus,
		})
		a.bookEdit.mu.Lock()
		j := a.bookEdit.jobs[id]
		if err != nil {
			if j != nil {
				j.status = "failed"
				j.errMsg = err.Error()
			}
		} else if j != nil {
			j.status = "done"
			j.result = report
		}
		a.bookEdit.mu.Unlock()
		if err != nil {
			a.emitBookReadError(id, err.Error())
			return
		}
		a.emitBookReadDone(id, report)
	}(root, in)
	return BookReadJobInfo{ID: id, Status: "pending"}, nil
}

// GetBookReadReport 获取通读报告结果。
func (a *App) GetBookReadReport(jobID string) (BookReadReportDTO, error) {
	a.bookEdit.mu.Lock()
	j, ok := a.bookEdit.jobs[jobID]
	a.bookEdit.mu.Unlock()
	if !ok {
		return BookReadReportDTO{}, fmt.Errorf("任务不存在")
	}
	if j.status != "done" {
		return BookReadReportDTO{}, fmt.Errorf("任务未完成: %s", j.status)
	}
	rep, ok := j.result.(*workflows.BookReadReport)
	if !ok || rep == nil {
		return BookReadReportDTO{}, fmt.Errorf("报告不可用")
	}
	items := make([]BookReadItemDTO, len(rep.Items))
	for i, it := range rep.Items {
		items[i] = BookReadItemDTO{
			Category: it.Category, Severity: it.Severity, Chapter: it.Chapter,
			Title: it.Title, Excerpt: it.Excerpt, Suggestion: it.Suggestion,
		}
	}
	return BookReadReportDTO{Summary: rep.Summary, Items: items}, nil
}

// StartBatchPolish 异步批量润色。
func (a *App) StartBatchPolish(in StartBatchPolishInput) (BatchPolishJobInfo, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return BatchPolishJobInfo{}, err
	}
	root := reg.ActivePath()
	if root == "" {
		return BatchPolishJobInfo{}, errNoActiveProject
	}
	cfg, err := config.Load()
	if err != nil {
		return BatchPolishJobInfo{}, err
	}
	if err := app.RequireAPIKey(cfg); err != nil {
		return BatchPolishJobInfo{}, err
	}
	if len(in.Chapters) == 0 {
		return BatchPolishJobInfo{}, fmt.Errorf("请选择至少一章")
	}
	a.bookEdit.mu.Lock()
	for id, j := range a.bookEdit.jobs {
		if j.kind == "polish" && (j.status == "running" || j.status == "pending") {
			a.bookEdit.mu.Unlock()
			return BatchPolishJobInfo{}, fmt.Errorf("已有润色任务进行中: %s", id)
		}
	}
	id := fmt.Sprintf("polish-%d", time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	a.bookEdit.jobs[id] = &bookEditJob{kind: "polish", status: "pending", cancel: cancel}
	a.bookEdit.mu.Unlock()
	a.emitPolishStatus(id, "pending", "")

	go func(projectRoot string, input StartBatchPolishInput) {
		defer cancel()
		actx, err := app.LoadContext(projectRoot)
		if err != nil {
			a.failPolishJob(id, err.Error())
			return
		}
		defer actx.Close()
		a.emitPolishStatus(id, "running", "正在批量润色…")
		wf := workflows.NewBatchPolishWorkflow(actx.Config, actx.Project, actx.Store)
		report, err := wf.Run(ctx, actx.Project, actx.Store, workflows.BatchPolishOptions{
			Chapters: input.Chapters, Rule: input.Rule,
		}, func(ch, total int) {
			a.emitPolishProgress(id, ch, total)
		})
		a.bookEdit.mu.Lock()
		j := a.bookEdit.jobs[id]
		if err != nil {
			if j != nil {
				j.status = "failed"
				j.errMsg = err.Error()
			}
		} else if j != nil {
			j.status = "done"
			j.result = report
		}
		a.bookEdit.mu.Unlock()
		if err != nil {
			a.emitPolishError(id, err.Error())
			return
		}
		a.emitPolishDone(id, report)
	}(root, in)
	return BatchPolishJobInfo{ID: id, Status: "pending"}, nil
}

// GetBatchPolishReport 获取批量润色结果。
func (a *App) GetBatchPolishReport(jobID string) (BatchPolishReportDTO, error) {
	a.bookEdit.mu.Lock()
	j, ok := a.bookEdit.jobs[jobID]
	a.bookEdit.mu.Unlock()
	if !ok {
		return BatchPolishReportDTO{}, fmt.Errorf("任务不存在")
	}
	if j.status != "done" {
		return BatchPolishReportDTO{}, fmt.Errorf("任务未完成: %s", j.status)
	}
	rep, ok := j.result.(*workflows.BatchPolishReport)
	if !ok || rep == nil {
		return BatchPolishReportDTO{}, fmt.Errorf("报告不可用")
	}
	results := make([]BatchPolishChapterResultDTO, len(rep.Results))
	for i, r := range rep.Results {
		results[i] = BatchPolishChapterResultDTO{
			Chapter: r.Chapter, Title: r.Title, Original: r.Original,
			Polished: r.Polished, Error: r.Error,
		}
	}
	return BatchPolishReportDTO{Rule: rep.Rule, Results: results}, nil
}

// ApplyBatchPolishChapter 应用单章润色结果。
func (a *App) ApplyBatchPolishChapter(chapter int, content string) error {
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return workflows.ApplyPolishChapter(actx.Project, actx.Store, chapter, content)
	})
	if err == nil {
		a.session.invalidate()
	}
	return err
}

// PreviewBatchPolishDiff 预览单章润色 diff。
func (a *App) PreviewBatchPolishDiff(chapter int, original, polished string) (DiffResultDTO, error) {
	diff := workflows.PreviewPolishDiff(chapter, original, polished)
	return toDiffDTO(diff), nil
}

func (a *App) failBookReadJob(id, msg string) {
	a.bookEdit.mu.Lock()
	if j, ok := a.bookEdit.jobs[id]; ok {
		j.status = "failed"
		j.errMsg = msg
	}
	a.bookEdit.mu.Unlock()
	a.emitBookReadError(id, msg)
}

func (a *App) failPolishJob(id, msg string) {
	a.bookEdit.mu.Lock()
	if j, ok := a.bookEdit.jobs[id]; ok {
		j.status = "failed"
		j.errMsg = msg
	}
	a.bookEdit.mu.Unlock()
	a.emitPolishError(id, msg)
}

func (a *App) emitBookReadStatus(jobID, status, message string) {
	runtime.EventsEmit(a.ctx, eventBookReadStatus, map[string]any{
		"job_id": jobID, "status": status, "message": message,
	})
}

func (a *App) emitBookReadError(jobID, errMsg string) {
	runtime.EventsEmit(a.ctx, eventBookReadError, map[string]any{"job_id": jobID, "error": errMsg})
}

func (a *App) emitBookReadDone(jobID string, rep *workflows.BookReadReport) {
	dto := BookReadReportDTO{Summary: rep.Summary}
	for _, it := range rep.Items {
		dto.Items = append(dto.Items, BookReadItemDTO{
			Category: it.Category, Severity: it.Severity, Chapter: it.Chapter,
			Title: it.Title, Excerpt: it.Excerpt, Suggestion: it.Suggestion,
		})
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventBookReadDone, map[string]any{"job_id": jobID, "report": string(raw)})
}

func (a *App) emitPolishStatus(jobID, status, message string) {
	runtime.EventsEmit(a.ctx, eventPolishStatus, map[string]any{
		"job_id": jobID, "status": status, "message": message,
	})
}

func (a *App) emitPolishProgress(jobID string, chapter, total int) {
	runtime.EventsEmit(a.ctx, eventPolishProgress, map[string]any{
		"job_id": jobID, "chapter": chapter, "total": total,
	})
}

func (a *App) emitPolishError(jobID, errMsg string) {
	runtime.EventsEmit(a.ctx, eventPolishError, map[string]any{"job_id": jobID, "error": errMsg})
}

func (a *App) emitPolishDone(jobID string, rep *workflows.BatchPolishReport) {
	dto := BatchPolishReportDTO{Rule: rep.Rule}
	for _, r := range rep.Results {
		dto.Results = append(dto.Results, BatchPolishChapterResultDTO{
			Chapter: r.Chapter, Title: r.Title, Original: r.Original,
			Polished: r.Polished, Error: r.Error,
		})
	}
	raw, _ := json.Marshal(dto)
	runtime.EventsEmit(a.ctx, eventPolishDone, map[string]any{"job_id": jobID, "report": string(raw)})
}
