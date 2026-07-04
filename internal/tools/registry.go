package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	openai "github.com/sashabaranov/go-openai"
)

type Registry struct {
	root  string
	store *store.Store
	tools map[string]func([]byte) (string, error)
	defs  []openai.Tool
}

func NewRegistry() *Registry {
	r := &Registry{tools: map[string]func([]byte) (string, error){}}
	return r
}

func (r *Registry) BindProject(root string, st *store.Store) {
	r.root = root
	r.store = st
	r.tools = map[string]func([]byte) (string, error){
		"read_file":             r.readFile,
		"write_file":            r.writeFile,
		"search_project":        r.searchProject,
		"query_entity":          r.queryEntity,
		"query_foreshadow":      r.queryForeshadow,
		"update_memory":         r.updateMemory,
		"get_chapter_outline":   r.getChapterOutline,
		"list_chapters":         r.listChapters,
	}
	r.defs = []openai.Tool{
		toolDef("read_file", "读取项目内相对路径文件", map[string]any{
			"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}},
			"required": []string{"path"},
		}),
		toolDef("write_file", "写入项目内相对路径文件", map[string]any{
			"type": "object", "properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			}, "required": []string{"path", "content"},
		}),
		toolDef("search_project", "FTS 检索章节和设定", map[string]any{
			"type": "object", "properties": map[string]any{
				"query": map[string]any{"type": "string"},
				"limit": map[string]any{"type": "integer"},
			}, "required": []string{"query"},
		}),
		toolDef("query_entity", "查询实体（角色/地点/物品）", map[string]any{
			"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}},
			"required": []string{"query"},
		}),
		toolDef("query_foreshadow", "查询伏笔", map[string]any{
			"type": "object", "properties": map[string]any{"status": map[string]any{"type": "string"}},
		}),
		toolDef("update_memory", "写入长期记忆", map[string]any{
			"type": "object", "properties": map[string]any{
				"category": map[string]any{"type": "string"},
				"subject":  map[string]any{"type": "string"},
				"content":  map[string]any{"type": "string"},
				"chapter":  map[string]any{"type": "integer"},
			}, "required": []string{"category", "subject", "content"},
		}),
		toolDef("get_chapter_outline", "读取章纲（从卷纲文件）", map[string]any{
			"type": "object", "properties": map[string]any{
				"chapter": map[string]any{"type": "integer"},
				"volume":  map[string]any{"type": "integer"},
			}, "required": []string{"chapter"},
		}),
		toolDef("list_chapters", "列出已写章节", map[string]any{"type": "object", "properties": map[string]any{}}),
	}
}

// BindProjectPlan 绑定规划任务可用的只读工具（不含 write_file / update_memory）。
func (r *Registry) BindProjectPlan(root string, st *store.Store) {
	r.root = root
	r.store = st
	r.tools = map[string]func([]byte) (string, error){
		"read_file":           r.readFile,
		"search_project":      r.searchProject,
		"query_entity":        r.queryEntity,
		"query_foreshadow":    r.queryForeshadow,
		"get_chapter_outline": r.getChapterOutline,
		"list_chapters":       r.listChapters,
	}
	r.defs = []openai.Tool{
		toolDef("read_file", "读取项目内相对路径文件", map[string]any{
			"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}},
			"required": []string{"path"},
		}),
		toolDef("search_project", "FTS 检索章节和设定", map[string]any{
			"type": "object", "properties": map[string]any{
				"query": map[string]any{"type": "string"},
				"limit": map[string]any{"type": "integer"},
			}, "required": []string{"query"},
		}),
		toolDef("query_entity", "查询实体（角色/地点/物品）", map[string]any{
			"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}},
			"required": []string{"query"},
		}),
		toolDef("query_foreshadow", "查询伏笔", map[string]any{
			"type": "object", "properties": map[string]any{"status": map[string]any{"type": "string"}},
		}),
		toolDef("get_chapter_outline", "读取章纲（从卷纲文件）", map[string]any{
			"type": "object", "properties": map[string]any{
				"chapter": map[string]any{"type": "integer"},
				"volume":  map[string]any{"type": "integer"},
			}, "required": []string{"chapter"},
		}),
		toolDef("list_chapters", "列出已写章节", map[string]any{"type": "object", "properties": map[string]any{}}),
	}
}

func toolDef(name, desc string, params map[string]any) openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        name,
			Description: desc,
			Parameters:  params,
		},
	}
}

func (r *Registry) Tools() []openai.Tool { return r.defs }

func (r *Registry) Execute(name string, args []byte) (string, error) {
	fn, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return fn(args)
}

func (r *Registry) safePath(rel string) (string, error) {
	rel = filepath.Clean(rel)
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path escapes project root")
	}
	return filepath.Join(r.root, rel), nil
}

func (r *Registry) readFile(args []byte) (string, error) {
	var in struct{ Path string `json:"path"` }
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	path, err := r.safePath(in.Path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]string{"path": in.Path, "content": string(data)})
	return string(out), nil
}

func (r *Registry) writeFile(args []byte) (string, error) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	path, err := r.safePath(in.Path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(in.Content), 0o644); err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]string{"path": in.Path, "status": "written"})
	return string(out), nil
}

func (r *Registry) searchProject(args []byte) (string, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if r.store == nil {
		return `{"results":[]}`, nil
	}
	results, err := r.store.SearchFTS(in.Query, in.Limit)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]any{"results": results})
	return string(out), nil
}

func (r *Registry) queryEntity(args []byte) (string, error) {
	var in struct{ Query string `json:"query"` }
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	entities, err := r.store.SearchEntities(in.Query, 20)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(entities)
	return string(out), nil
}

func (r *Registry) queryForeshadow(args []byte) (string, error) {
	var in struct{ Status string `json:"status"` }
	_ = json.Unmarshal(args, &in)
	items, err := r.store.ListForeshadows(in.Status)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(items)
	return string(out), nil
}

func (r *Registry) updateMemory(args []byte) (string, error) {
	var in struct {
		Category string `json:"category"`
		Subject  string `json:"subject"`
		Content  string `json:"content"`
		Chapter  int    `json:"chapter"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	id := uuid.New().String()
	_, err := r.store.UpsertMemory(store.Memory{
		ID: id, Category: in.Category, Subject: in.Subject, Content: in.Content,
		SourceChapter: in.Chapter, Status: "active", CreatedAt: project.Timestamp(),
	})
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]string{"id": id, "status": "inserted"})
	return string(out), nil
}

func (r *Registry) getChapterOutline(args []byte) (string, error) {
	var in struct {
		Chapter int `json:"chapter"`
		Volume  int `json:"volume"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	vol := in.Volume
	if vol <= 0 {
		vol = 1
	}
	path := filepath.Join(r.root, "大纲", fmt.Sprintf("第%02d卷.md", vol))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(map[string]any{"volume": vol, "chapter": in.Chapter, "outline": string(data)})
	return string(out), nil
}

func (r *Registry) listChapters(args []byte) (string, error) {
	chs, err := r.store.ListChapters()
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(chs)
	return string(out), nil
}
