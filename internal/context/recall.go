package contextbuilder

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/rag"
	"github.com/tanlian/agent_nova/internal/store"
)

const (
	recallCandidateLimit = 200
	recallDefaultTopK    = 10
	recallSemanticTopK   = 15
	recallMaxRunes       = 2400
	rrfK                 = 60.0
)

// MemoryRecallHit 单条记忆召回结果（含选中理由，供 UI 与调试）。
type MemoryRecallHit struct {
	ID       string  `json:"id"`
	Category string  `json:"category"`
	Subject  string  `json:"subject"`
	Content  string  `json:"content"`
	Source   string  `json:"source"` // rule | semantic | rrf
	Reason   string  `json:"reason"`
	Score    float64 `json:"score"`
}

type recallRanked struct {
	hit  MemoryRecallHit
	rank int
}

// RecallMemories 智能召回：规则匹配 + 可选语义检索，RRF 融合后按 token 预算截断。
func RecallMemories(ctx context.Context, st *store.Store, cfg *config.Config, in RecallInput) []MemoryRecallHit {
	if st == nil {
		return nil
	}
	keywords := in.Keywords
	if len(keywords) == 0 {
		keywords = extractKeywords(in.Outline, in.RecentSummary, in.Protagonist, in.Cheat, in.EntityNames...)
	}
	entityNames := in.EntityNames
	if len(entityNames) == 0 {
		entityNames = relatedEntityNames(st, keywords)
	}

	memories, err := st.ListActiveMemories(recallCandidateLimit)
	if err != nil || len(memories) == 0 {
		return nil
	}

	ruleRanked := ruleRecall(memories, keywords, entityNames, in.Protagonist, in.Chapter)
	semanticRanked := semanticRecall(ctx, st, cfg, in.Outline, in.RecentSummary, memories)

	merged := rrfMerge(ruleRanked, semanticRanked)
	topK := in.Limit
	if topK <= 0 {
		topK = recallDefaultTopK
	}
	if len(merged) > topK {
		merged = merged[:topK]
	}
	return applyMemoryBudget(merged, recallMaxRunes)
}

type RecallInput struct {
	Chapter       int
	Outline       string
	RecentSummary string
	Keywords      []string
	EntityNames   []string
	Protagonist   string
	Cheat         string
	Limit         int
}

func ruleRecall(memories []store.Memory, keywords, entityNames []string, protagonist string, chapter int) []recallRanked {
	type scored struct {
		m       store.Memory
		score   float64
		reasons []string
	}
	var hits []scored
	for _, m := range memories {
		s, reasons := scoreMemoryRule(m, keywords, entityNames, protagonist, chapter)
		if s <= 0 {
			continue
		}
		hits = append(hits, scored{m: m, score: s, reasons: reasons})
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].m.CreatedAt > hits[j].m.CreatedAt
	})

	out := make([]recallRanked, len(hits))
	for i, h := range hits {
		reason := strings.Join(uniqueStrings(h.reasons), "；")
		if reason == "" {
			reason = "规则匹配"
		}
		out[i] = recallRanked{
			rank: i + 1,
			hit: MemoryRecallHit{
				ID: h.m.ID, Category: h.m.Category, Subject: h.m.Subject, Content: h.m.Content,
				Source: "rule", Reason: reason, Score: h.score,
			},
		}
	}
	return out
}

func scoreMemoryRule(m store.Memory, keywords, entityNames []string, protagonist string, chapter int) (float64, []string) {
	var score float64
	var reasons []string
	text := m.Subject + " " + m.Content

	if protagonist != "" && strings.Contains(text, protagonist) {
		score += 5
		reasons = append(reasons, "主角锚点")
	}
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if strings.Contains(m.Subject, kw) {
			score += 3
			reasons = append(reasons, "章纲关键词:"+kw)
		} else if strings.Contains(m.Content, kw) {
			score += 2
			reasons = append(reasons, "内容匹配:"+kw)
		}
	}
	for _, name := range entityNames {
		if name == "" {
			continue
		}
		if strings.Contains(text, name) {
			score += 2.5
			reasons = append(reasons, "相关实体:"+name)
		}
	}
	if m.SourceChapter > 0 && chapter > 0 {
		gap := chapter - m.SourceChapter
		if gap >= 0 && gap <= 15 {
			score += 0.5
		}
	}
	return score, reasons
}

func semanticRecall(ctx context.Context, st *store.Store, cfg *config.Config, outline, recentSummary string, memories []store.Memory) []recallRanked {
	if cfg == nil || cfg.OpenAIAPIKey == "" {
		return nil
	}
	query := strings.TrimSpace(outline)
	if recentSummary != "" {
		query = query + "\n" + truncateRunes(recentSummary, 1200)
	}
	query = truncateRunes(strings.TrimSpace(query), 2000)
	if query == "" {
		return nil
	}

	memByID := map[string]store.Memory{}
	for _, m := range memories {
		memByID[m.ID] = m
	}

	ragIdx := rag.NewIndexer(cfg)
	embs, scores, err := ragIdx.Search(ctx, st, query, recallSemanticTopK*2)
	if err != nil {
		return nil
	}

	var out []recallRanked
	rank := 0
	for i, e := range embs {
		if e.Kind != "memory" {
			continue
		}
		m, ok := memByID[e.RefID]
		if !ok {
			if got, err := st.GetMemory(e.RefID); err == nil {
				m = got
			} else {
				continue
			}
		}
		rank++
		sim := 0.0
		if i < len(scores) {
			sim = scores[i]
		}
		out = append(out, recallRanked{
			rank: rank,
			hit: MemoryRecallHit{
				ID: m.ID, Category: m.Category, Subject: m.Subject, Content: m.Content,
				Source: "semantic",
				Reason: fmt.Sprintf("语义相似度 %.2f", sim),
				Score:  sim,
			},
		})
		if rank >= recallSemanticTopK {
			break
		}
	}
	return out
}

func rrfMerge(ruleRanked, semanticRanked []recallRanked) []MemoryRecallHit {
	type acc struct {
		hit   MemoryRecallHit
		score float64
	}
	byID := map[string]*acc{}

	add := func(list []recallRanked, src string) {
		for _, r := range list {
			rrf := 1.0 / (rrfK + float64(r.rank))
			a, ok := byID[r.hit.ID]
			if !ok {
				h := r.hit
				h.Source = src
				byID[r.hit.ID] = &acc{hit: h, score: rrf}
				continue
			}
			a.score += rrf
			if src == "semantic" && a.hit.Source == "rule" {
				a.hit.Source = "rrf"
				a.hit.Reason = a.hit.Reason + "；" + r.hit.Reason
			} else if src == "rule" && a.hit.Source == "semantic" {
				a.hit.Source = "rrf"
				a.hit.Reason = r.hit.Reason + "；" + a.hit.Reason
			}
		}
	}
	add(ruleRanked, "rule")
	add(semanticRanked, "semantic")

	var merged []*acc
	for _, a := range byID {
		a.hit.Score = a.score
		merged = append(merged, a)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].score != merged[j].score {
			return merged[i].score > merged[j].score
		}
		return merged[i].hit.Subject < merged[j].hit.Subject
	})

	out := make([]MemoryRecallHit, len(merged))
	for i, a := range merged {
		out[i] = a.hit
	}
	return out
}

func applyMemoryBudget(hits []MemoryRecallHit, maxRunes int) []MemoryRecallHit {
	if maxRunes <= 0 {
		return hits
	}
	total := 0
	var out []MemoryRecallHit
	for _, h := range hits {
		line := fmt.Sprintf("[%s/%s] %s", h.Category, h.Subject, h.Content)
		n := len([]rune(line))
		if total+n > maxRunes && len(out) > 0 {
			break
		}
		if n > maxRunes-total && len(out) == 0 {
			h.Content = truncateRunes(h.Content, maxRunes-len([]rune(fmt.Sprintf("[%s/%s] ", h.Category, h.Subject))))
		}
		out = append(out, h)
		total += len([]rune(fmt.Sprintf("[%s/%s] %s", h.Category, h.Subject, h.Content)))
	}
	return out
}

func formatMemoryRecalls(hits []MemoryRecallHit) string {
	var parts []string
	for _, h := range hits {
		parts = append(parts, fmt.Sprintf("[%s/%s] %s", h.Category, h.Subject, h.Content))
	}
	return strings.Join(parts, "\n")
}

func buildFTSQuery(keywords []string, chapter int) string {
	parts := []string{fmt.Sprintf("第%d", chapter)}
	seen := map[string]struct{}{}
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw == "" || len([]rune(kw)) < 2 {
			continue
		}
		if _, ok := seen[kw]; ok {
			continue
		}
		seen[kw] = struct{}{}
		parts = append(parts, kw)
		if len(parts) >= 6 {
			break
		}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.Join(parts, " OR ")
}

func relatedEntityNames(st *store.Store, keywords []string) []string {
	if st == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var names []string
	for _, kw := range keywords {
		if kw == "" || len([]rune(kw)) < 2 {
			continue
		}
		ents, err := st.SearchEntities(kw, 5)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if _, ok := seen[e.Name]; ok {
				continue
			}
			seen[e.Name] = struct{}{}
			names = append(names, e.Name)
		}
	}
	if len(names) > 12 {
		names = names[:12]
	}
	return names
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
