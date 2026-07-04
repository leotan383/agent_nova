package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// SettleResult 记忆沉淀到设定集的结果。
type SettleResult struct {
	RelPath string // 相对设定集路径，如 角色/林枫.md
}

// SettleCharacterMemoryToSetting 将角色类记忆追加到 设定集/角色/{subject}.md 的「性格」段，并归档该记忆。
func SettleCharacterMemoryToSetting(p *project.Project, st *store.Store, memoryID string) (SettleResult, error) {
	if st == nil {
		return SettleResult{}, fmt.Errorf("数据库不可用")
	}
	m, err := st.GetMemory(memoryID)
	if err != nil {
		return SettleResult{}, err
	}
	if m.Category != "character" {
		return SettleResult{}, fmt.Errorf("仅支持角色类记忆沉淀到设定集")
	}
	subject := project.SanitizeSettingTitle(m.Subject)
	if subject == "" {
		return SettleResult{}, fmt.Errorf("记忆主题（角色名）不能为空")
	}
	content := strings.TrimSpace(m.Content)
	if content == "" {
		return SettleResult{}, fmt.Errorf("记忆内容不能为空")
	}

	entry := formatSettleEntry(content, m.SourceChapter)
	rel := filepath.ToSlash(filepath.Join(project.SettingsSubCharacter, subject+".md"))
	if err := project.ValidateSettingRelPath(rel); err != nil {
		return SettleResult{}, err
	}

	path := filepath.Join(p.SettingsDir(), filepath.FromSlash(rel))
	var body string
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return SettleResult{}, err
		}
		body = project.SettingBodyTemplate(project.SettingsSubCharacter, subject, "character", p.Meta)
		body = appendToMarkdownSection(body, "性格", entry)
	} else if err != nil {
		return SettleResult{}, err
	} else {
		raw, err := os.ReadFile(path)
		if err != nil {
			return SettleResult{}, err
		}
		body = appendToMarkdownSection(string(raw), "性格", entry)
	}

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return SettleResult{}, err
	}
	_ = st.IndexSettingFTS(rel, body)
	if err := st.SetMemoryStatus(memoryID, "archived"); err != nil {
		return SettleResult{}, err
	}
	return SettleResult{RelPath: rel}, nil
}

func formatSettleEntry(content string, sourceChapter int) string {
	if sourceChapter > 0 {
		return fmt.Sprintf("- %s（第 %d 章沉淀）", content, sourceChapter)
	}
	return "- " + content
}

// appendToMarkdownSection 在指定 ## 标题段末尾追加一行；无该段则新建。
func appendToMarkdownSection(body, section, line string) string {
	heading := "## " + section
	idx := strings.Index(body, heading)
	if idx < 0 {
		return strings.TrimRight(body, "\n") + "\n\n" + heading + "\n\n" + line + "\n"
	}
	afterHeading := idx + len(heading)
	rest := body[afterHeading:]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx < 0 {
		return strings.TrimRight(body, "\n") + "\n" + line + "\n"
	}
	insertAt := afterHeading + nextIdx
	return body[:insertAt] + "\n" + line + body[insertAt:]
}
