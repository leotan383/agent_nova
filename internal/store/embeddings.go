package store

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

type Embedding struct {
	ID        string
	Kind      string
	RefID     string
	Text      string
	Vector    []float32
	UpdatedAt string
}

func (s *Store) UpsertEmbedding(e Embedding) error {
	blob := encodeVector(e.Vector)
	if e.UpdatedAt == "" {
		e.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`
INSERT INTO embeddings (id, kind, ref_id, text, vector, updated_at) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET kind=excluded.kind, ref_id=excluded.ref_id, text=excluded.text,
  vector=excluded.vector, updated_at=excluded.updated_at`,
		e.ID, e.Kind, e.RefID, e.Text, blob, e.UpdatedAt)
	return err
}

func (s *Store) SearchEmbeddings(queryVec []float32, limit int) ([]Embedding, []float64, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.Query(`SELECT id, kind, ref_id, text, vector FROM embeddings`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	type scored struct {
		e     Embedding
		score float64
	}
	var all []scored
	for rows.Next() {
		var e Embedding
		var blob []byte
		if err := rows.Scan(&e.ID, &e.Kind, &e.RefID, &e.Text, &blob); err != nil {
			return nil, nil, err
		}
		e.Vector = decodeVector(blob)
		all = append(all, scored{e: e, score: cosineSimilarity(queryVec, e.Vector)})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	// simple top-k sort
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].score > all[i].score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	if len(all) > limit {
		all = all[:limit]
	}
	out := make([]Embedding, len(all))
	scores := make([]float64, len(all))
	for i, sc := range all {
		out[i] = sc.e
		scores[i] = sc.score
	}
	return out, scores, nil
}

func encodeVector(v []float32) []byte {
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

func decodeVector(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := 0; i < n; i++ {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func EmbeddingID(kind, refID string) string {
	return fmt.Sprintf("%s:%s", kind, refID)
}
