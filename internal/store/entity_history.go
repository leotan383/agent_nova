package store

import (
	"fmt"
	"strings"
	"time"
)

// EntityStateSnapshot 实体在某一章的状态快照。
type EntityStateSnapshot struct {
	EntityID   string
	Chapter    int
	StateJSON  string
	RecordedAt string
}

// RecordEntityStateHistory 记录实体在某章的状态快照（同章重复提取会覆盖）。
func (s *Store) RecordEntityStateHistory(entityID string, chapter int, stateJSON string) error {
	entityID = trimEntityID(entityID)
	if entityID == "" || chapter <= 0 || stateJSON == "" {
		return nil
	}
	_, err := s.db.Exec(`
INSERT INTO entity_state_history (entity_id, chapter, state_json, recorded_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(entity_id, chapter) DO UPDATE SET
  state_json=excluded.state_json,
  recorded_at=excluded.recorded_at`,
		entityID, chapter, stateJSON, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

// ListEntityStateHistory 按章号升序返回实体全部历史快照。
func (s *Store) ListEntityStateHistory(entityID string) ([]EntityStateSnapshot, error) {
	entityID = trimEntityID(entityID)
	if entityID == "" {
		return nil, fmt.Errorf("实体不存在")
	}
	rows, err := s.db.Query(`
SELECT entity_id, chapter, state_json, recorded_at
FROM entity_state_history
WHERE entity_id=?
ORDER BY chapter ASC, recorded_at ASC`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EntityStateSnapshot
	for rows.Next() {
		var snap EntityStateSnapshot
		if err := rows.Scan(&snap.EntityID, &snap.Chapter, &snap.StateJSON, &snap.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, rows.Err()
}

func trimEntityID(id string) string {
	return strings.TrimSpace(id)
}

// reassignEntityHistory 将旧实体 id 的历史迁移到目标 id（合并重复实体时调用）。
func (s *Store) reassignEntityHistory(fromID, toID string) error {
	if fromID == "" || toID == "" || fromID == toID {
		return nil
	}
	snaps, err := s.ListEntityStateHistory(fromID)
	if err != nil {
		return err
	}
	for _, snap := range snaps {
		if err := s.RecordEntityStateHistory(toID, snap.Chapter, snap.StateJSON); err != nil {
			return err
		}
	}
	_, err = s.db.Exec(`DELETE FROM entity_state_history WHERE entity_id=?`, fromID)
	return err
}
