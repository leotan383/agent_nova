package main

import (
	"fmt"
	"strings"

	"github.com/tanlian/agent_nova/internal/inspiration"
	"github.com/tanlian/agent_nova/internal/library"
)

// InspirationListFilterDTO 灵感列表筛选。
type InspirationListFilterDTO struct {
	Query           string `json:"query"`
	Status          string `json:"status"`
	Genre           string `json:"genre"`
	Tag             string `json:"tag"`
	IncludeArchived bool   `json:"include_archived"`
}

// CreateInspirationInput 新建灵感。
type CreateInspirationInput struct {
	Spark       string   `json:"spark"`
	Title       string   `json:"title"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Synopsis    string   `json:"synopsis"`
	Protagonist string   `json:"protagonist"`
	Cheat       string   `json:"cheat"`
	Tags        []string `json:"tags"`
}

// UpdateInspirationInput 更新灵感。
type UpdateInspirationInput struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Spark       string   `json:"spark"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Synopsis    string   `json:"synopsis"`
	Protagonist string   `json:"protagonist"`
	Cheat       string   `json:"cheat"`
	Tags        []string `json:"tags"`
}

// InspirationDTO 灵感详情。
type InspirationDTO struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	DisplayTitle string  `json:"display_title"`
	Spark       string   `json:"spark"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Synopsis    string   `json:"synopsis"`
	Protagonist string   `json:"protagonist"`
	Cheat       string   `json:"cheat"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	Pinned      bool     `json:"pinned"`
	Archived    bool     `json:"archived"`
	NovelID     string   `json:"novel_id,omitempty"`
	NovelPath   string   `json:"novel_path,omitempty"`
	NovelTitle  string   `json:"novel_title,omitempty"`
	UsedAt      string   `json:"used_at,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// InspirationPrefillDTO 从灵感创书预填。
type InspirationPrefillDTO struct {
	InspirationID string `json:"inspiration_id"`
	Title         string `json:"title"`
	Genre         string `json:"genre"`
	Style         string `json:"style"`
	Synopsis      string `json:"synopsis"`
	Protagonist   string `json:"protagonist"`
	Cheat         string `json:"cheat"`
	SeedPrompt    string `json:"seed_prompt"`
}

func loadInspirationStore() (*inspiration.Store, error) {
	return inspiration.Load()
}

func toInspirationDTO(insp inspiration.Inspiration) InspirationDTO {
	dto := InspirationDTO{
		ID:           insp.ID,
		Title:        insp.Title,
		DisplayTitle: inspiration.DisplayTitle(insp),
		Spark:        insp.Spark,
		Genre:        insp.Genre,
		Style:        insp.Style,
		Synopsis:     insp.Synopsis,
		Protagonist:  insp.Protagonist,
		Cheat:        insp.Cheat,
		Tags:         insp.Tags,
		Status:       insp.Status,
		Pinned:       insp.Pinned,
		Archived:     insp.Archived,
		NovelID:      insp.NovelID,
		NovelPath:    insp.NovelPath,
		NovelTitle:   insp.NovelTitle,
		CreatedAt:    insp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:    insp.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if !insp.UsedAt.IsZero() {
		dto.UsedAt = insp.UsedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return dto
}

// ListInspirations 返回灵感卡片列表。
func (a *App) ListInspirations(filter InspirationListFilterDTO) ([]inspiration.Card, error) {
	store, err := loadInspirationStore()
	if err != nil {
		return nil, err
	}
	return store.ListCards(inspiration.ListFilter{
		Query:           filter.Query,
		Status:          filter.Status,
		Genre:           filter.Genre,
		Tag:             filter.Tag,
		IncludeArchived: filter.IncludeArchived,
	}), nil
}

// GetInspiration 获取灵感详情。
func (a *App) GetInspiration(id string) (InspirationDTO, error) {
	store, err := loadInspirationStore()
	if err != nil {
		return InspirationDTO{}, err
	}
	insp, err := store.Get(id)
	if err != nil {
		return InspirationDTO{}, err
	}
	return toInspirationDTO(insp), nil
}

// GetInspirationPrefill 获取创书预填字段。
func (a *App) GetInspirationPrefill(id string) (InspirationPrefillDTO, error) {
	store, err := loadInspirationStore()
	if err != nil {
		return InspirationPrefillDTO{}, err
	}
	insp, err := store.Get(id)
	if err != nil {
		return InspirationPrefillDTO{}, err
	}
	p := inspiration.ToPrefill(insp)
	return InspirationPrefillDTO{
		InspirationID: id,
		Title:         p.Title,
		Genre:         p.Genre,
		Style:         p.Style,
		Synopsis:      p.Synopsis,
		Protagonist:   p.Protagonist,
		Cheat:         p.Cheat,
		SeedPrompt:    p.SeedPrompt,
	}, nil
}

// CreateInspiration 新建灵感。
func (a *App) CreateInspiration(in CreateInspirationInput) (InspirationDTO, error) {
	store, err := loadInspirationStore()
	if err != nil {
		return InspirationDTO{}, err
	}
	insp, err := store.Create(inspiration.CreateInput{
		Spark: in.Spark, Title: in.Title, Genre: in.Genre, Style: in.Style,
		Synopsis: in.Synopsis, Protagonist: in.Protagonist, Cheat: in.Cheat, Tags: in.Tags,
	})
	if err != nil {
		return InspirationDTO{}, err
	}
	return toInspirationDTO(insp), nil
}

// UpdateInspiration 更新灵感。
func (a *App) UpdateInspiration(in UpdateInspirationInput) (InspirationDTO, error) {
	if strings.TrimSpace(in.ID) == "" {
		return InspirationDTO{}, fmt.Errorf("灵感 ID 不能为空")
	}
	store, err := loadInspirationStore()
	if err != nil {
		return InspirationDTO{}, err
	}
	insp, err := store.Update(in.ID, inspiration.UpdateInput{
		Title: in.Title, Spark: in.Spark, Genre: in.Genre, Style: in.Style,
		Synopsis: in.Synopsis, Protagonist: in.Protagonist, Cheat: in.Cheat, Tags: in.Tags,
	})
	if err != nil {
		return InspirationDTO{}, err
	}
	return toInspirationDTO(insp), nil
}

// DeleteInspiration 删除灵感。
func (a *App) DeleteInspiration(id string) error {
	store, err := loadInspirationStore()
	if err != nil {
		return err
	}
	return store.Delete(id)
}

// SetInspirationPinned 置顶或取消置顶。
func (a *App) SetInspirationPinned(id string, pinned bool) error {
	store, err := loadInspirationStore()
	if err != nil {
		return err
	}
	return store.SetPinned(id, pinned)
}

// SetInspirationArchived 归档或取消归档。
func (a *App) SetInspirationArchived(id string, archived bool) error {
	store, err := loadInspirationStore()
	if err != nil {
		return err
	}
	return store.SetArchived(id, archived)
}

func (a *App) markInspirationUsed(inspirationID string, card library.NovelCard) {
	if strings.TrimSpace(inspirationID) == "" {
		return
	}
	store, err := loadInspirationStore()
	if err != nil {
		return
	}
	_ = store.MarkUsed(inspirationID, card.ID, card.Path, card.Title)
}
