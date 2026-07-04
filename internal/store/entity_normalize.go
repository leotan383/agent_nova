package store

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var entityParenSuffix = regexp.MustCompile(`[（(][^）)]*[）)]$`)

func normalizeEntityType(typ string) string {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return "character"
	}
	return typ
}

// CanonicalEntityName 合并同一实体的变体称呼（如「母亲（未具名）」→「母亲」）。
func CanonicalEntityName(name string) string {
	name = strings.TrimSpace(name)
	for i := 0; i < 3; i++ {
		next := strings.TrimSpace(entityParenSuffix.ReplaceAllString(name, ""))
		if next == name || next == "" {
			break
		}
		name = next
	}
	return name
}

func canonicalFromEntityID(id string) string {
	if i := strings.Index(id, ":"); i >= 0 && i < len(id)-1 {
		return CanonicalEntityName(id[i+1:])
	}
	return CanonicalEntityName(id)
}

func entityCanonicalKey(e Entity) (typ, canonical string) {
	return EntityCanonicalKey(e)
}

// EntityCanonicalKey 返回实体规范化分组键。
func EntityCanonicalKey(e Entity) (typ, canonical string) {
	typ = normalizeEntityType(e.Type)
	canonical = CanonicalEntityName(e.Name)
	if canonical == "" {
		canonical = canonicalFromEntityID(e.ID)
	}
	return typ, canonical
}

// EntityID 生成实体唯一 ID（基于规范化名称）。
func EntityID(typ, name string) string {
	return fmt.Sprintf("%s:%s", normalizeEntityType(typ), CanonicalEntityName(name))
}

// AppendEntityAlias 在 state 中记录曾出现的变体名称。
func AppendEntityAlias(state map[string]any, alias string) map[string]any {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return state
	}
	if state == nil {
		state = map[string]any{}
	}
	canonical := CanonicalEntityName(alias)
	if alias == canonical {
		return state
	}
	existing, _ := state["aliases"].([]any)
	seen := map[string]struct{}{}
	var list []string
	for _, v := range existing {
		if s, ok := v.(string); ok && s != "" {
			seen[s] = struct{}{}
			list = append(list, s)
		}
	}
	if _, ok := seen[alias]; !ok {
		list = append(list, alias)
	}
	state["aliases"] = list
	return state
}

// MergeDuplicateEntities 合并仅括号修饰不同的重复实体（幂等）。
func (s *Store) MergeDuplicateEntities() (int, error) {
	all, err := s.ListEntities("", 5000)
	if err != nil {
		return 0, err
	}
	type groupKey struct {
		typ, canonical string
	}
	groups := map[groupKey][]Entity{}
	for _, e := range all {
		typ, canonical := entityCanonicalKey(e)
		if canonical == "" {
			continue
		}
		k := groupKey{typ, canonical}
		groups[k] = append(groups[k], e)
	}
	merged := 0
	for k, group := range groups {
		if len(group) <= 1 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return group[i].LastChapter < group[j].LastChapter
		})
		targetID := EntityID(k.typ, k.canonical)
		state := map[string]any{}
		lastChapter := 0
		for _, e := range group {
			if e.LastChapter > lastChapter {
				lastChapter = e.LastChapter
			}
			AppendEntityAlias(state, e.Name)
			if e.StateJSON == "" {
				continue
			}
			var extra map[string]any
			if json.Unmarshal([]byte(e.StateJSON), &extra) != nil {
				continue
			}
			for key, val := range extra {
				if key == "aliases" {
					continue
				}
				state[key] = val
			}
		}
		if err := s.UpsertEntity(Entity{
			ID: targetID, Type: k.typ, Name: k.canonical,
			StateJSON: EntityStateJSON(state), LastChapter: lastChapter,
		}); err != nil {
			return merged, err
		}
		for _, e := range group {
			if e.ID == targetID {
				continue
			}
			if err := s.reassignEntityHistory(e.ID, targetID); err != nil {
				return merged, err
			}
			if err := s.deleteEntity(e.ID); err != nil {
				return merged, err
			}
			merged++
		}
	}
	return merged, nil
}

func (s *Store) deleteEntity(id string) error {
	_, err := s.db.Exec(`DELETE FROM entities WHERE id=?`, id)
	return err
}

// FindEntity 按 id 或规范 id（character:母亲）查找实体。
func (s *Store) FindEntity(key string) (*Entity, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("实体不存在")
	}
	row := s.db.QueryRow(`SELECT id, type, name, state_json, last_chapter FROM entities WHERE id=?`, key)
	var direct Entity
	err := row.Scan(&direct.ID, &direct.Type, &direct.Name, &direct.StateJSON, &direct.LastChapter)
	if err == nil {
		return &direct, nil
	}
	all, err := s.ListEntities("", 5000)
	if err != nil {
		return nil, err
	}
	for _, e := range all {
		if EntityID(e.Type, e.Name) == key {
			copy := e
			return &copy, nil
		}
	}
	return nil, fmt.Errorf("实体不存在")
}
