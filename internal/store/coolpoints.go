package store

import "database/sql"

func (s *Store) UpsertCoolPoint(cp CoolPoint) error {
	delivered := 0
	if cp.Delivered {
		delivered = 1
	}
	_, err := s.db.Exec(`
INSERT INTO cool_points (id, chapter, type, description, delivered) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET chapter=excluded.chapter, type=excluded.type,
  description=excluded.description, delivered=excluded.delivered`,
		cp.ID, cp.Chapter, cp.Type, cp.Description, delivered)
	return err
}

func (s *Store) ListCoolPoints(chapter int) ([]CoolPoint, error) {
	var rows *sql.Rows
	var err error
	if chapter > 0 {
		rows, err = s.db.Query(`SELECT id, chapter, type, description, delivered FROM cool_points WHERE chapter=? ORDER BY chapter`, chapter)
	} else {
		rows, err = s.db.Query(`SELECT id, chapter, type, description, delivered FROM cool_points ORDER BY chapter DESC LIMIT 100`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CoolPoint
	for rows.Next() {
		var cp CoolPoint
		var d int
		if err := rows.Scan(&cp.ID, &cp.Chapter, &cp.Type, &cp.Description, &d); err != nil {
			return nil, err
		}
		cp.Delivered = d == 1
		out = append(out, cp)
	}
	return out, rows.Err()
}
