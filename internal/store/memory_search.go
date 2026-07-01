package store

import (
	"database/sql"
	"fmt"
)

// ListActiveMemories 列出 active 状态记忆（按时间倒序）。
func (s *Store) ListActiveMemories(limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := s.db.Query(`
SELECT id, category, subject, content, source_chapter, status, created_at FROM memories
WHERE status IS NULL OR status='' OR status='active'
ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemories(rows)
}

// GetMemory 按 ID 读取记忆。
func (s *Store) GetMemory(id string) (Memory, error) {
	var m Memory
	err := s.db.QueryRow(`
SELECT id, category, subject, content, source_chapter, status, created_at FROM memories WHERE id=?`, id).
		Scan(&m.ID, &m.Category, &m.Subject, &m.Content, &m.SourceChapter, &m.Status, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return Memory{}, fmt.Errorf("memory not found: %s", id)
	}
	return m, err
}

func scanMemories(rows *sql.Rows) ([]Memory, error) {
	var out []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.Category, &m.Subject, &m.Content, &m.SourceChapter, &m.Status, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
