package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tanlian/agent_nova/internal/project"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Chapter struct {
	Number      int
	Title       string
	WordCount   int
	Path        string
	SummaryPath string
	Status      string
	UpdatedAt   string
}

type Entity struct {
	ID          string
	Type        string
	Name        string
	StateJSON   string
	LastChapter int
}

type Foreshadow struct {
	ID               string
	Description      string
	PlantedChapter   int
	ResolvedChapter  int
	Status           string
}

type Memory struct {
	ID            string
	Category      string
	Subject       string
	Content       string
	SourceChapter int
	Status        string
	CreatedAt     string
}

type ReviewRecord struct {
	ChapterNumber int
	HookScore     float64
	CoolPoint     string
	Debt          string
	ReportJSON    string
	Path          string
}

type CoolPoint struct {
	ID          string
	Chapter     int
	Type        string
	Description string
	Delivered   bool
}

type MemoryConflict struct {
	Subject  string
	Count    int
	Memories []Memory
}

type IndexStaleReport struct {
	Stale       bool
	FileCount   int
	IndexCount  int
	FTSCount    int
	Issues      []string
}

func Open(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	_, _ = db.Exec(`PRAGMA journal_mode = WAL`)
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
-- project_meta: 项目元数据镜像，与 nova.yaml 同步，便于 SQL 查询与 Dashboard 展示
CREATE TABLE IF NOT EXISTS project_meta (
  root TEXT PRIMARY KEY,   -- 项目根目录绝对路径
  title TEXT,              -- 书名
  genre TEXT,              -- 题材（玄幻/都市/科幻等）
  phase TEXT,              -- 创作阶段：init_done/planning/writing/paused
  updated_at TEXT          -- 最后同步时间（RFC3339）
);

-- chapters: 章节索引（读模型），正文真源在 正文/*.md，此处存路径、字数与状态
CREATE TABLE IF NOT EXISTS chapters (
  number INTEGER PRIMARY KEY,  -- 章号
  title TEXT,                  -- 章节标题
  word_count INTEGER,          -- 正文字数
  path TEXT,                   -- 正文文件路径
  summary_path TEXT,           -- 摘要文件路径（摘要/第NNN章.summary.md）
  status TEXT,                 -- draft|reviewed|published
  updated_at TEXT              -- 最后更新时间
);

-- entities: 故事实体状态表（角色/地点/物品等），用于一致性校验与 query 检索
CREATE TABLE IF NOT EXISTS entities (
  id TEXT PRIMARY KEY,         -- 实体唯一 ID
  type TEXT,                   -- 类型：character/location/item 等
  name TEXT,                   -- 显示名
  state_json TEXT,             -- 当前状态 JSON（境界、关系、持有物等）
  last_chapter INTEGER         -- 最后一次更新的章号
);

-- foreshadows: 伏笔追踪，记录埋设与回收，防止长篇连载遗忘 unresolved 线索
CREATE TABLE IF NOT EXISTS foreshadows (
  id TEXT PRIMARY KEY,         -- 伏笔唯一 ID
  description TEXT,            -- 伏笔描述
  planted_chapter INTEGER,     -- 埋设章号
  resolved_chapter INTEGER,    -- 回收章号（未回收为 0 或 NULL）
  status TEXT                  -- open|resolved
);

-- memories: 长期记忆，写章后自动提取或 nova learn 手动写入，写前注入 Top-K 上下文
CREATE TABLE IF NOT EXISTS memories (
  id TEXT PRIMARY KEY,         -- 记忆唯一 ID
  category TEXT,               -- style/plot/character/world
  subject TEXT,                -- 主题关键词（如主角名、写法名）
  content TEXT,                -- 记忆正文
  source_chapter INTEGER,      -- 来源章号（0 表示手动录入）
  status TEXT,                 -- active|archived
  created_at TEXT              -- 创建时间（RFC3339）
);

-- reviews: 审查结果结构化存储，对应 审查/第NNN章.review.md，含追读力指标
CREATE TABLE IF NOT EXISTS reviews (
  chapter_number INTEGER PRIMARY KEY,  -- 章号
  hook_score REAL,                     -- 章末钩子强度 0-10
  cool_point TEXT,                     -- 爽点兑现描述
  debt TEXT,                           -- 悬念债务/未兑现承诺
  report_json TEXT,                    -- 完整审查 JSON
  path TEXT                            -- 审查 Markdown 文件路径
);

-- chapters_fts: 章节全文检索（FTS5），供 nova query / 写章上下文检索
CREATE VIRTUAL TABLE IF NOT EXISTS chapters_fts USING fts5(
  chapter_number UNINDEXED,  -- 章号（不参与分词，仅作过滤）
  title,                     -- 章节标题
  content,                   -- 正文全文
  tokenize='unicode61'
);

-- settings_fts: 设定集全文检索（FTS5），检索 设定集/*.md 内容
CREATE VIRTUAL TABLE IF NOT EXISTS settings_fts USING fts5(
  file_path UNINDEXED,       -- 相对项目根的路径
  content,                   -- 设定文件全文
  tokenize='unicode61'
);

-- cool_points: 爽点追踪（章级微/中/大爽点，planned vs delivered）
CREATE TABLE IF NOT EXISTS cool_points (
  id TEXT PRIMARY KEY,
  chapter INTEGER,
  type TEXT,                 -- micro|medium|major
  description TEXT,
  delivered INTEGER          -- 0|1
);

-- embeddings: 向量索引（OpenAI embedding），供语义检索
CREATE TABLE IF NOT EXISTS embeddings (
  id TEXT PRIMARY KEY,
  kind TEXT,                 -- chapter|setting|summary
  ref_id TEXT,
  text TEXT,
  vector BLOB,
  updated_at TEXT
);

CREATE TABLE IF NOT EXISTS schema_meta (
  key TEXT PRIMARY KEY,
  value TEXT
);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return s.ensureTrigramFTS()
}

func (s *Store) ensureTrigramFTS() error {
	var v string
	_ = s.db.QueryRow(`SELECT value FROM schema_meta WHERE key='fts_tokenizer'`).Scan(&v)
	if v == "trigram" {
		return nil
	}
	stmts := []string{
		`DROP TABLE IF EXISTS chapters_fts`,
		`DROP TABLE IF EXISTS settings_fts`,
		`CREATE VIRTUAL TABLE chapters_fts USING fts5(
  chapter_number UNINDEXED, title, content, tokenize='trigram')`,
		`CREATE VIRTUAL TABLE settings_fts USING fts5(
  file_path UNINDEXED, content, tokenize='trigram')`,
		`INSERT OR REPLACE INTO schema_meta (key, value) VALUES ('fts_tokenizer', 'trigram')`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("fts migrate: %w", err)
		}
	}
	return nil
}

func (s *Store) InitProject(root string, meta project.Meta) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO project_meta (root, title, genre, phase, updated_at) VALUES (?, ?, ?, ?, ?)`,
		root, meta.Title, meta.Genre, meta.Phase, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) UpsertChapter(ch Chapter) error {
	_, err := s.db.Exec(`
INSERT INTO chapters (number, title, word_count, path, summary_path, status, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(number) DO UPDATE SET
  title=CASE WHEN excluded.title != '' THEN excluded.title ELSE chapters.title END,
  word_count=CASE WHEN excluded.word_count > 0 THEN excluded.word_count ELSE chapters.word_count END,
  path=CASE WHEN excluded.path != '' THEN excluded.path ELSE chapters.path END,
  summary_path=CASE WHEN excluded.summary_path != '' THEN excluded.summary_path ELSE chapters.summary_path END,
  status=CASE WHEN excluded.status != '' THEN excluded.status ELSE chapters.status END,
  updated_at=CASE WHEN excluded.updated_at != '' THEN excluded.updated_at ELSE chapters.updated_at END`,
		ch.Number, ch.Title, ch.WordCount, ch.Path, ch.SummaryPath, ch.Status, ch.UpdatedAt,
	)
	return err
}

func (s *Store) GetChapter(n int) (*Chapter, error) {
	row := s.db.QueryRow(`SELECT number, title, word_count, path, summary_path, status, updated_at FROM chapters WHERE number=?`, n)
	var ch Chapter
	if err := row.Scan(&ch.Number, &ch.Title, &ch.WordCount, &ch.Path, &ch.SummaryPath, &ch.Status, &ch.UpdatedAt); err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Store) ListChapters() ([]Chapter, error) {
	rows, err := s.db.Query(`SELECT number, title, word_count, path, summary_path, status, updated_at FROM chapters ORDER BY number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chapter
	for rows.Next() {
		var ch Chapter
		if err := rows.Scan(&ch.Number, &ch.Title, &ch.WordCount, &ch.Path, &ch.SummaryPath, &ch.Status, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) UpsertEntity(e Entity) error {
	_, err := s.db.Exec(`
INSERT INTO entities (id, type, name, state_json, last_chapter) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET type=excluded.type, name=excluded.name, state_json=excluded.state_json, last_chapter=excluded.last_chapter`,
		e.ID, e.Type, e.Name, e.StateJSON, e.LastChapter,
	)
	return err
}

func (s *Store) SearchEntities(query string, limit int) ([]Entity, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.Query(`SELECT id, type, name, state_json, last_chapter FROM entities WHERE name LIKE ? OR state_json LIKE ? LIMIT ?`,
		"%"+query+"%", "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.StateJSON, &e.LastChapter); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListEntities 列出实体状态，可按 type 过滤（character/location/item）。
func (s *Store) ListEntities(entityType string, limit int) ([]Entity, error) {
	if limit <= 0 {
		limit = 200
	}
	var rows *sql.Rows
	var err error
	if entityType != "" {
		rows, err = s.db.Query(`SELECT id, type, name, state_json, last_chapter FROM entities WHERE type=? ORDER BY last_chapter DESC, name LIMIT ?`, entityType, limit)
	} else {
		rows, err = s.db.Query(`SELECT id, type, name, state_json, last_chapter FROM entities ORDER BY type, last_chapter DESC, name LIMIT ?`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entity
	for rows.Next() {
		var e Entity
		if err := rows.Scan(&e.ID, &e.Type, &e.Name, &e.StateJSON, &e.LastChapter); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) UpsertForeshadow(f Foreshadow) error {
	_, err := s.db.Exec(`
INSERT INTO foreshadows (id, description, planted_chapter, resolved_chapter, status) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET description=excluded.description, planted_chapter=excluded.planted_chapter,
  resolved_chapter=excluded.resolved_chapter, status=excluded.status`,
		f.ID, f.Description, f.PlantedChapter, f.ResolvedChapter, f.Status,
	)
	return err
}

func (s *Store) ListForeshadows(status string) ([]Foreshadow, error) {
	var rows *sql.Rows
	var err error
	if status != "" {
		rows, err = s.db.Query(`SELECT id, description, planted_chapter, resolved_chapter, status FROM foreshadows WHERE status=?`, status)
	} else {
		rows, err = s.db.Query(`SELECT id, description, planted_chapter, resolved_chapter, status FROM foreshadows`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Foreshadow
	for rows.Next() {
		var f Foreshadow
		if err := rows.Scan(&f.ID, &f.Description, &f.PlantedChapter, &f.ResolvedChapter, &f.Status); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) InsertMemory(m Memory) error {
	_, err := s.UpsertMemory(m)
	return err
}

// UpsertMemory inserts or updates by category+subject. Returns true if inserted new row.
func (s *Store) UpsertMemory(m Memory) (bool, error) {
	if m.ID == "" {
		m.ID = fmt.Sprintf("mem-%d", time.Now().UnixNano())
	}
	if m.CreatedAt == "" {
		m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	var existingID string
	err := s.db.QueryRow(
		`SELECT id FROM memories WHERE category=? AND subject=? AND status='active' LIMIT 1`,
		m.Category, m.Subject,
	).Scan(&existingID)
	if err == nil && existingID != "" {
		_, err = s.db.Exec(
			`UPDATE memories SET content=?, source_chapter=?, created_at=? WHERE id=?`,
			m.Content, m.SourceChapter, m.CreatedAt, existingID,
		)
		return false, err
	}
	_, err = s.db.Exec(
		`INSERT INTO memories (id, category, subject, content, source_chapter, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Category, m.Subject, m.Content, m.SourceChapter, m.Status, m.CreatedAt,
	)
	return true, err
}

// UpdateMemoryContent 按 ID 更新记忆正文。
// UpdateMemory 按 ID 更新记忆字段。
func (s *Store) UpdateMemory(m Memory) error {
	if m.ID == "" {
		return fmt.Errorf("记忆 ID 不能为空")
	}
	if m.Status == "" {
		m.Status = "active"
	}
	m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE memories SET category=?, subject=?, content=?, source_chapter=?, status=?, created_at=? WHERE id=?`,
		m.Category, m.Subject, m.Content, m.SourceChapter, m.Status, m.CreatedAt, m.ID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("记忆不存在: %s", m.ID)
	}
	return nil
}

// SetMemoryStatus 归档或恢复记忆。
func (s *Store) SetMemoryStatus(id, status string) error {
	if id == "" {
		return fmt.Errorf("记忆 ID 不能为空")
	}
	res, err := s.db.Exec(`UPDATE memories SET status=?, created_at=? WHERE id=?`, status, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("记忆不存在: %s", id)
	}
	return nil
}

// ResolveForeshadow 标记伏笔已回收。
func (s *Store) ResolveForeshadow(id string, resolvedChapter int) error {
	if id == "" {
		return fmt.Errorf("伏笔 ID 不能为空")
	}
	res, err := s.db.Exec(
		`UPDATE foreshadows SET status='resolved', resolved_chapter=? WHERE id=?`,
		resolvedChapter, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("伏笔不存在: %s", id)
	}
	return nil
}

// UpdateForeshadowDescription 更新伏笔描述。
func (s *Store) UpdateForeshadowDescription(id, description string) error {
	if id == "" {
		return fmt.Errorf("伏笔 ID 不能为空")
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return fmt.Errorf("描述不能为空")
	}
	res, err := s.db.Exec(`UPDATE foreshadows SET description=? WHERE id=?`, description, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("伏笔不存在: %s", id)
	}
	return nil
}

func (s *Store) UpdateMemoryContent(id, content string) error {
	if id == "" {
		return fmt.Errorf("记忆 ID 不能为空")
	}
	res, err := s.db.Exec(
		`UPDATE memories SET content=?, created_at=? WHERE id=? AND status='active'`,
		content, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("记忆不存在: %s", id)
	}
	return nil
}

func (s *Store) FindMemoryConflicts() ([]MemoryConflict, error) {
	rows, err := s.db.Query(`
SELECT subject, COUNT(*) AS cnt FROM memories WHERE status='active'
GROUP BY subject HAVING cnt > 1`)
	if err != nil {
		return nil, err
	}
	var pending []MemoryConflict
	for rows.Next() {
		var c MemoryConflict
		if err := rows.Scan(&c.Subject, &c.Count); err != nil {
			_ = rows.Close()
			return nil, err
		}
		pending = append(pending, c)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	out := make([]MemoryConflict, 0, len(pending))
	for _, c := range pending {
		c.Memories, _ = s.QueryMemories("", c.Subject, 20)
		out = append(out, c)
	}
	return out, nil
}

func (s *Store) QueryMemories(category, subject string, limit int) ([]Memory, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `SELECT id, category, subject, content, source_chapter, status, created_at FROM memories WHERE 1=1`
	var args []any
	if category != "" {
		q += ` AND category=?`
		args = append(args, category)
	}
	if subject != "" {
		q += ` AND subject LIKE ?`
		args = append(args, "%"+subject+"%")
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func (s *Store) MemoryStats() (map[string]int, int, error) {
	rows, err := s.db.Query(`SELECT category, COUNT(*) FROM memories GROUP BY category`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	stats := map[string]int{}
	total := 0
	for rows.Next() {
		var cat string
		var n int
		if err := rows.Scan(&cat, &n); err != nil {
			return nil, 0, err
		}
		stats[cat] = n
		total += n
	}
	return stats, total, rows.Err()
}

func (s *Store) DumpMemories() ([]Memory, error) {
	return s.QueryMemories("", "", 10000)
}

func (s *Store) UpsertReview(r ReviewRecord) error {
	_, err := s.db.Exec(`
INSERT INTO reviews (chapter_number, hook_score, cool_point, debt, report_json, path) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(chapter_number) DO UPDATE SET hook_score=excluded.hook_score, cool_point=excluded.cool_point,
  debt=excluded.debt, report_json=excluded.report_json, path=excluded.path`,
		r.ChapterNumber, r.HookScore, r.CoolPoint, r.Debt, r.ReportJSON, r.Path)
	return err
}

func (s *Store) IndexChapterFTS(num int, title, content string) error {
	if _, err := s.db.Exec(`DELETE FROM chapters_fts WHERE chapter_number=?`, num); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO chapters_fts (chapter_number, title, content) VALUES (?, ?, ?)`, num, title, content)
	return err
}

func (s *Store) IndexSettingFTS(path, content string) error {
	if _, err := s.db.Exec(`DELETE FROM settings_fts WHERE file_path=?`, path); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO settings_fts (file_path, content) VALUES (?, ?)`, path, content)
	return err
}

func (s *Store) SearchFTS(query string, limit int) ([]map[string]string, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := s.db.Query(`
SELECT 'chapter' AS kind, chapter_number, title, snippet(chapters_fts, 2, '【', '】', '...', 32) AS snippet
FROM chapters_fts WHERE chapters_fts MATCH ?
UNION ALL
SELECT 'setting' AS kind, file_path, '', snippet(settings_fts, 1, '【', '】', '...', 32)
FROM settings_fts WHERE settings_fts MATCH ?
LIMIT ?`, query, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]string
	for rows.Next() {
		var kind, id, title, snippet string
		if err := rows.Scan(&kind, &id, &title, &snippet); err != nil {
			return nil, err
		}
		out = append(out, map[string]string{"kind": kind, "id": id, "title": title, "snippet": snippet})
	}
	return out, rows.Err()
}

func (s *Store) FTSStats() (chapterCount, settingCount int, err error) {
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM chapters_fts`).Scan(&chapterCount)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM settings_fts`).Scan(&settingCount)
	return chapterCount, settingCount, nil
}

func (s *Store) GetReview(chapter int) (ReviewRecord, error) {
	var r ReviewRecord
	err := s.db.QueryRow(
		`SELECT chapter_number, hook_score, cool_point, debt, report_json, path FROM reviews WHERE chapter_number=?`,
		chapter,
	).Scan(&r.ChapterNumber, &r.HookScore, &r.CoolPoint, &r.Debt, &r.ReportJSON, &r.Path)
	if err == sql.ErrNoRows {
		return ReviewRecord{}, fmt.Errorf("review not found: chapter %d", chapter)
	}
	return r, err
}

func (s *Store) ListReviews() ([]ReviewRecord, error) {
	rows, err := s.db.Query(`SELECT chapter_number, hook_score, cool_point, debt, report_json, path FROM reviews ORDER BY chapter_number`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReviewRecord
	for rows.Next() {
		var r ReviewRecord
		if err := rows.Scan(&r.ChapterNumber, &r.HookScore, &r.CoolPoint, &r.Debt, &r.ReportJSON, &r.Path); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func EntityStateJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
