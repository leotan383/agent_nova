package store

import (
	"database/sql"
	"fmt"
)

const chapterRenumberOffset = 1_000_000

// CascadeImpact 章号偏移将影响的对象数量。
type CascadeImpact struct {
	ChaptersShifted      int `json:"chapters_shifted"`
	MemoriesAffected     int `json:"memories_affected"`
	ForeshadowsAffected  int `json:"foreshadows_affected"`
	EntitiesAffected     int `json:"entities_affected"`
	EntityHistoryAffected int `json:"entity_history_affected"`
	ReviewsAffected      int `json:"reviews_affected"`
	CoolPointsAffected   int `json:"cool_points_affected"`
}

// PreviewChapterShift 预览从 from 章起偏移 delta 的影响。
func (s *Store) PreviewChapterShift(from int, delta int) (CascadeImpact, error) {
	if from <= 0 || delta == 0 {
		return CascadeImpact{}, fmt.Errorf("无效偏移参数")
	}
	var out CascadeImpact
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM chapters WHERE number >= ?`, from).Scan(&out.ChaptersShifted)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE source_chapter >= ? AND source_chapter > 0`, from).Scan(&out.MemoriesAffected)
	_ = s.db.QueryRow(`
SELECT COUNT(*) FROM foreshadows
WHERE planted_chapter >= ? OR (resolved_chapter >= ? AND resolved_chapter > 0)`, from, from).Scan(&out.ForeshadowsAffected)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM entities WHERE last_chapter >= ?`, from).Scan(&out.EntitiesAffected)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM entity_state_history WHERE chapter >= ?`, from).Scan(&out.EntityHistoryAffected)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE chapter_number >= ?`, from).Scan(&out.ReviewsAffected)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM cool_points WHERE chapter >= ?`, from).Scan(&out.CoolPointsAffected)
	return out, nil
}

// ShiftChapterReferences 将从 from 章起的所有章号引用偏移 delta（+1 或 -1）。
func (s *Store) ShiftChapterReferences(from int, delta int) error {
	if from <= 0 || delta == 0 {
		return fmt.Errorf("无效偏移参数")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := shiftChapterPK(tx, from, delta); err != nil {
		return err
	}
	stmts := []struct {
		q string
		a []any
	}{
		{`UPDATE memories SET source_chapter = source_chapter + ? WHERE source_chapter >= ? AND source_chapter > 0`, []any{delta, from}},
		{`UPDATE foreshadows SET planted_chapter = planted_chapter + ? WHERE planted_chapter >= ?`, []any{delta, from}},
		{`UPDATE foreshadows SET resolved_chapter = resolved_chapter + ? WHERE resolved_chapter >= ? AND resolved_chapter > 0`, []any{delta, from}},
		{`UPDATE entities SET last_chapter = last_chapter + ? WHERE last_chapter >= ?`, []any{delta, from}},
		{`UPDATE cool_points SET chapter = chapter + ? WHERE chapter >= ?`, []any{delta, from}},
	}
	for _, st := range stmts {
		if _, err := tx.Exec(st.q, st.a...); err != nil {
			return err
		}
	}
	if err := shiftEntityHistoryPK(tx, from, delta); err != nil {
		return err
	}
	if err := shiftReviewPK(tx, from, delta); err != nil {
		return err
	}
	return tx.Commit()
}

func shiftChapterPK(tx *sql.Tx, from, delta int) error {
	if _, err := tx.Exec(`UPDATE chapters SET number = number + ? WHERE number >= ?`, chapterRenumberOffset, from); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE chapters SET number = number + ? - ? WHERE number >= ?`,
		delta, chapterRenumberOffset, from+chapterRenumberOffset); err != nil {
		return err
	}
	return nil
}

func shiftEntityHistoryPK(tx *sql.Tx, from, delta int) error {
	rows, err := tx.Query(`SELECT entity_id, chapter, state_json, recorded_at FROM entity_state_history WHERE chapter >= ?`, from)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		entityID, stateJSON, recordedAt string
		chapter                         int
	}
	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.entityID, &r.chapter, &r.stateJSON, &r.recordedAt); err != nil {
			return err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM entity_state_history WHERE chapter >= ?`, from); err != nil {
		return err
	}
	for _, r := range items {
		newCh := r.chapter + delta
		if _, err := tx.Exec(`
INSERT INTO entity_state_history (entity_id, chapter, state_json, recorded_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(entity_id, chapter) DO UPDATE SET state_json=excluded.state_json, recorded_at=excluded.recorded_at`,
			r.entityID, newCh, r.stateJSON, r.recordedAt); err != nil {
			return err
		}
	}
	return nil
}

func shiftReviewPK(tx *sql.Tx, from, delta int) error {
	rows, err := tx.Query(`SELECT chapter_number, hook_score, cool_point, debt, report_json, path FROM reviews WHERE chapter_number >= ?`, from)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		chapter int
		hook    float64
		cool    string
		debt    string
		report  string
		path    string
	}
	var items []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.chapter, &r.hook, &r.cool, &r.debt, &r.report, &r.path); err != nil {
			return err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM reviews WHERE chapter_number >= ?`, from); err != nil {
		return err
	}
	for _, r := range items {
		if _, err := tx.Exec(`
INSERT INTO reviews (chapter_number, hook_score, cool_point, debt, report_json, path)
VALUES (?, ?, ?, ?, ?, ?)`, r.chapter+delta, r.hook, r.cool, r.debt, r.report, r.path); err != nil {
			return err
		}
	}
	return nil
}

// DeleteChapterRecord 删除 chapters 表中的章记录。
func (s *Store) DeleteChapterRecord(number int) error {
	_, err := s.db.Exec(`DELETE FROM chapters WHERE number=?`, number)
	return err
}

// DeleteChapterReferences 删除指定章号的关联记录（删章时用）。
func (s *Store) DeleteChapterReferences(number int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []string{
		`DELETE FROM reviews WHERE chapter_number=?`,
		`DELETE FROM entity_state_history WHERE chapter=?`,
		`DELETE FROM cool_points WHERE chapter=?`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q, number); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM chapters WHERE number=?`, number); err != nil {
		return err
	}
	return tx.Commit()
}
