package inspiration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/tanlian/agent_nova/internal/paths"
)

const fileName = "inspirations.json"

const (
	StatusSeed       = "seed"
	StatusDeveloping = "developing"
	StatusReady      = "ready"
	StatusUsed       = "used"
	StatusArchived   = "archived"
)

// Inspiration 全局灵感条目，独立于具体书籍。
type Inspiration struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Spark       string    `json:"spark"`
	Genre       string    `json:"genre,omitempty"`
	Style       string    `json:"style,omitempty"`
	Synopsis    string    `json:"synopsis,omitempty"`
	Protagonist string    `json:"protagonist,omitempty"`
	Cheat       string    `json:"cheat,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Status      string    `json:"status"`
	Pinned      bool      `json:"pinned"`
	Archived    bool      `json:"archived"`
	NovelID     string    `json:"novel_id,omitempty"`
	NovelPath   string    `json:"novel_path,omitempty"`
	NovelTitle  string    `json:"novel_title,omitempty"`
	UsedAt      time.Time `json:"used_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Card 供 UI 列表展示。
type Card struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Spark       string   `json:"spark"`
	SparkPreview string  `json:"spark_preview"`
	Genre       string   `json:"genre"`
	Style       string   `json:"style"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	Pinned      bool     `json:"pinned"`
	Archived    bool     `json:"archived"`
	NovelID     string   `json:"novel_id,omitempty"`
	NovelTitle  string   `json:"novel_title,omitempty"`
	UpdatedAt   string   `json:"updated_at"`
	CreatedAt   string   `json:"created_at"`
}

type fileData struct {
	Inspirations []Inspiration `json:"inspirations"`
}

// Store 灵感库持久化。
type Store struct {
	Inspirations []Inspiration `json:"inspirations"`
}

func storePath() string {
	return filepath.Join(paths.Global().ConfigDir, fileName)
}

// Load 读取灵感库；不存在则返回空库。
func Load() (*Store, error) {
	path := storePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, err
	}
	var f fileData
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse inspirations: %w", err)
	}
	return &Store{Inspirations: f.Inspirations}, nil
}

// Save 持久化灵感库。
func (s *Store) Save() error {
	dir := filepath.Dir(storePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(fileData{Inspirations: s.Inspirations}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storePath(), data, 0o644)
}

func (s *Store) find(id string) (*Inspiration, int) {
	for i := range s.Inspirations {
		if s.Inspirations[i].ID == id {
			return &s.Inspirations[i], i
		}
	}
	return nil, -1
}

// Get 按 ID 获取灵感。
func (s *Store) Get(id string) (Inspiration, error) {
	insp, _ := s.find(id)
	if insp == nil {
		return Inspiration{}, fmt.Errorf("未找到灵感: %s", id)
	}
	return *insp, nil
}

// ListFilter 列表筛选。
type ListFilter struct {
	Query           string
	Status          string
	Genre           string
	Tag             string
	IncludeArchived bool
}

// ListCards 返回卡片列表。
func (s *Store) ListCards(f ListFilter) []Card {
	q := strings.ToLower(strings.TrimSpace(f.Query))
	tag := strings.TrimSpace(f.Tag)
	out := make([]Card, 0, len(s.Inspirations))
	for _, insp := range s.Inspirations {
		if insp.Archived && !f.IncludeArchived {
			continue
		}
		if f.IncludeArchived && !insp.Archived {
			continue
		}
		if f.Status != "" && insp.Status != f.Status {
			continue
		}
		if f.Genre != "" && insp.Genre != f.Genre {
			continue
		}
		if tag != "" && !hasTag(insp.Tags, tag) {
			continue
		}
		if q != "" && !matchesQuery(insp, q) {
			continue
		}
		out = append(out, BuildCard(insp))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	return out
}

func hasTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func matchesQuery(insp Inspiration, q string) bool {
	fields := []string{
		insp.Title,
		insp.Spark,
		insp.Synopsis,
		insp.Protagonist,
		insp.Genre,
		strings.Join(insp.Tags, " "),
	}
	for _, f := range fields {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// CreateInput 新建灵感。
type CreateInput struct {
	Spark       string
	Title       string
	Genre       string
	Style       string
	Synopsis    string
	Protagonist string
	Cheat       string
	Tags        []string
}

// Create 新建灵感。
func (s *Store) Create(in CreateInput) (Inspiration, error) {
	spark := strings.TrimSpace(in.Spark)
	if spark == "" {
		return Inspiration{}, fmt.Errorf("灵感内容不能为空")
	}
	now := time.Now().UTC()
	insp := Inspiration{
		ID:          uuid.New().String(),
		Title:       strings.TrimSpace(in.Title),
		Spark:       spark,
		Genre:       strings.TrimSpace(in.Genre),
		Style:       strings.TrimSpace(in.Style),
		Synopsis:    strings.TrimSpace(in.Synopsis),
		Protagonist: strings.TrimSpace(in.Protagonist),
		Cheat:       strings.TrimSpace(in.Cheat),
		Tags:        normalizeTags(in.Tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	insp.Status = ComputeStatus(insp)
	s.Inspirations = append(s.Inspirations, insp)
	return insp, s.Save()
}

// UpdateInput 更新灵感。
type UpdateInput struct {
	Title       string
	Spark       string
	Genre       string
	Style       string
	Synopsis    string
	Protagonist string
	Cheat       string
	Tags        []string
}

// Update 更新灵感。
func (s *Store) Update(id string, in UpdateInput) (Inspiration, error) {
	insp, idx := s.find(id)
	if insp == nil {
		return Inspiration{}, fmt.Errorf("未找到灵感: %s", id)
	}
	spark := strings.TrimSpace(in.Spark)
	if spark == "" {
		return Inspiration{}, fmt.Errorf("灵感内容不能为空")
	}
	insp.Title = strings.TrimSpace(in.Title)
	insp.Spark = spark
	insp.Genre = strings.TrimSpace(in.Genre)
	insp.Style = strings.TrimSpace(in.Style)
	insp.Synopsis = strings.TrimSpace(in.Synopsis)
	insp.Protagonist = strings.TrimSpace(in.Protagonist)
	insp.Cheat = strings.TrimSpace(in.Cheat)
	insp.Tags = normalizeTags(in.Tags)
	if insp.Status != StatusUsed {
		insp.Status = ComputeStatus(*insp)
	}
	insp.UpdatedAt = time.Now().UTC()
	s.Inspirations[idx] = *insp
	return *insp, s.Save()
}

// Delete 删除灵感。
func (s *Store) Delete(id string) error {
	_, idx := s.find(id)
	if idx < 0 {
		return fmt.Errorf("未找到灵感: %s", id)
	}
	s.Inspirations = append(s.Inspirations[:idx], s.Inspirations[idx+1:]...)
	return s.Save()
}

// SetPinned 置顶或取消置顶。
func (s *Store) SetPinned(id string, pinned bool) error {
	insp, idx := s.find(id)
	if insp == nil {
		return fmt.Errorf("未找到灵感: %s", id)
	}
	insp.Pinned = pinned
	insp.UpdatedAt = time.Now().UTC()
	s.Inspirations[idx] = *insp
	return s.Save()
}

// SetArchived 归档或取消归档。
func (s *Store) SetArchived(id string, archived bool) error {
	insp, idx := s.find(id)
	if insp == nil {
		return fmt.Errorf("未找到灵感: %s", id)
	}
	insp.Archived = archived
	if archived {
		insp.Status = StatusArchived
	} else if insp.Status == StatusArchived {
		insp.Status = ComputeStatus(*insp)
	}
	insp.UpdatedAt = time.Now().UTC()
	s.Inspirations[idx] = *insp
	return s.Save()
}

// MarkUsed 创建书成功后关联灵感。
func (s *Store) MarkUsed(id, novelID, novelPath, novelTitle string) error {
	insp, idx := s.find(id)
	if insp == nil {
		return fmt.Errorf("未找到灵感: %s", id)
	}
	insp.Status = StatusUsed
	insp.NovelID = novelID
	insp.NovelPath = novelPath
	insp.NovelTitle = novelTitle
	insp.UsedAt = time.Now().UTC()
	insp.UpdatedAt = time.Now().UTC()
	s.Inspirations[idx] = *insp
	return s.Save()
}

// DisplayTitle 展示用标题。
func DisplayTitle(insp Inspiration) string {
	if t := strings.TrimSpace(insp.Title); t != "" {
		return t
	}
	return truncateFirstLine(insp.Spark, 30)
}

// SparkPreview 卡片摘要。
func SparkPreview(spark string) string {
	line := strings.TrimSpace(strings.Split(spark, "\n")[0])
	return truncateFirstLine(line, 80)
}

func truncateFirstLine(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "未命名灵感"
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "…"
}

// ComputeStatus 根据内容推断状态。
func ComputeStatus(insp Inspiration) string {
	if insp.Archived {
		return StatusArchived
	}
	if insp.Status == StatusUsed {
		return StatusUsed
	}
	hasStruct := strings.TrimSpace(insp.Title) != "" ||
		strings.TrimSpace(insp.Synopsis) != "" ||
		strings.TrimSpace(insp.Protagonist) != "" ||
		strings.TrimSpace(insp.Cheat) != "" ||
		strings.TrimSpace(insp.Genre) != ""
	if hasStruct && strings.TrimSpace(insp.Spark) != "" {
		return StatusReady
	}
	if hasStruct {
		return StatusDeveloping
	}
	return StatusSeed
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// BuildCard 构建展示卡片。
func BuildCard(insp Inspiration) Card {
	return Card{
		ID:           insp.ID,
		Title:        DisplayTitle(insp),
		Spark:        insp.Spark,
		SparkPreview: SparkPreview(insp.Spark),
		Genre:        insp.Genre,
		Style:        insp.Style,
		Tags:         insp.Tags,
		Status:       insp.Status,
		Pinned:       insp.Pinned,
		Archived:     insp.Archived,
		NovelID:      insp.NovelID,
		NovelTitle:   insp.NovelTitle,
		UpdatedAt:    insp.UpdatedAt.Format(time.RFC3339),
		CreatedAt:    insp.CreatedAt.Format(time.RFC3339),
	}
}
