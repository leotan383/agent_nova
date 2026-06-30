package memory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

// BootstrapFromSettings seeds memories from 设定集 markdown files.
func BootstrapFromSettings(p *project.Project, st *store.Store) (int, error) {
	count := 0
	err := filepath.Walk(p.SettingsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		subject := strings.TrimSuffix(filepath.Base(path), ".md")
		content := string(data)
		if len([]rune(content)) > 600 {
			content = string([]rune(content)[:600]) + "..."
		}
		inserted, err := st.UpsertMemory(store.Memory{
			Category: "world", Subject: subject, Content: content,
			SourceChapter: 0, Status: "active", CreatedAt: project.Timestamp(),
		})
		if err != nil {
			return err
		}
		if inserted {
			count++
		}
		return nil
	})
	return count, err
}
