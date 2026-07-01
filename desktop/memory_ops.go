package main

import (
	"fmt"

	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// UpdateMemoryInput 更新记忆。
type UpdateMemoryInput struct {
	ID            string `json:"id"`
	Category      string `json:"category"`
	Subject       string `json:"subject"`
	Content       string `json:"content"`
	SourceChapter int    `json:"source_chapter"`
}

// CreateMemoryInput 新建记忆。
type CreateMemoryInput struct {
	Category      string `json:"category"`
	Subject       string `json:"subject"`
	Content       string `json:"content"`
	SourceChapter int    `json:"source_chapter"`
}

// UpdateMemory 更新记忆条目。
func (a *App) UpdateMemory(in UpdateMemoryInput) error {
	if in.ID == "" {
		return fmt.Errorf("记忆 ID 不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return actx.Store.UpdateMemory(store.Memory{
			ID: in.ID, Category: in.Category, Subject: in.Subject, Content: in.Content,
			SourceChapter: in.SourceChapter, Status: "active",
		})
	})
}

// ArchiveMemory 归档记忆。
func (a *App) ArchiveMemory(id string) error {
	if id == "" {
		return fmt.Errorf("记忆 ID 不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return actx.Store.SetMemoryStatus(id, "archived")
	})
}

// CreateMemory 手动新增记忆。
func (a *App) CreateMemory(in CreateMemoryInput) (MemoryDTO, error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return MemoryDTO{}, err
	}
	var out MemoryDTO
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		id := project.NewMemoryID()
		m := store.Memory{
			ID: id, Category: in.Category, Subject: in.Subject, Content: in.Content,
			SourceChapter: in.SourceChapter, Status: "active", CreatedAt: project.Timestamp(),
		}
		if err := actx.Store.InsertMemory(m); err != nil {
			return err
		}
		out = MemoryDTO{
			ID: m.ID, Category: m.Category, Subject: m.Subject, Content: m.Content,
			SourceChapter: m.SourceChapter, Status: m.Status,
		}
		return nil
	})
	return out, err
}

// ResolveForeshadow 标记伏笔已回收。
func (a *App) ResolveForeshadow(id string, resolvedChapter int) error {
	if id == "" {
		return fmt.Errorf("伏笔 ID 不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return actx.Store.ResolveForeshadow(id, resolvedChapter)
	})
}

// UpdateForeshadow 更新伏笔描述。
func (a *App) UpdateForeshadow(id, description string) error {
	if id == "" {
		return fmt.Errorf("伏笔 ID 不能为空")
	}
	reg, err := a.loadRegistry()
	if err != nil {
		return err
	}
	return a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		return actx.Store.UpdateForeshadowDescription(id, description)
	})
}

// MemoryConflictDTO 同 subject 的多条记忆冲突。
type MemoryConflictDTO struct {
	Subject  string      `json:"subject"`
	Count    int         `json:"count"`
	Memories []MemoryDTO `json:"memories"`
}

// FindMemoryConflicts 列出 subject 重复的记忆冲突。
func (a *App) FindMemoryConflicts() (out []MemoryConflictDTO, err error) {
	reg, err := a.loadRegistry()
	if err != nil {
		return nil, err
	}
	err = a.session.withActive(reg.ActivePath(), func(actx *app.Context) error {
		conflicts, err := actx.Store.FindMemoryConflicts()
		if err != nil {
			return err
		}
		out = make([]MemoryConflictDTO, len(conflicts))
		for i, c := range conflicts {
			out[i] = MemoryConflictDTO{Subject: c.Subject, Count: c.Count}
			out[i].Memories = make([]MemoryDTO, len(c.Memories))
			for j, m := range c.Memories {
				out[i].Memories[j] = MemoryDTO{
					ID: m.ID, Category: m.Category, Subject: m.Subject, Content: m.Content,
					SourceChapter: m.SourceChapter, Status: m.Status,
				}
			}
		}
		return nil
	})
	return out, err
}
