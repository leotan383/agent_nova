package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/tanlian/agent_nova/internal/project"
)

func Create(p *project.Project, label string) error {
	ts := time.Now().UTC().Format("20060102-150405")
	dest := filepath.Join(p.BackupDir(), fmt.Sprintf("%s-%s", label, ts))
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	dirs := []string{"设定集", "大纲", "正文"}
	for _, d := range dirs {
		src := filepath.Join(p.Root, d)
		if err := copyDir(src, filepath.Join(dest, d)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	metaSrc := filepath.Join(p.Root, project.MetaFile)
	metaDst := filepath.Join(dest, project.MetaFile)
	return copyFile(metaSrc, metaDst)
}

func List(p *project.Project) ([]string, error) {
	entries, err := os.ReadDir(p.BackupDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func Restore(p *project.Project, name string) error {
	src := filepath.Join(p.BackupDir(), name)
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("backup not found: %s", name)
	}
	dirs := []string{"设定集", "大纲", "正文"}
	for _, d := range dirs {
		if err := copyDir(filepath.Join(src, d), filepath.Join(p.Root, d)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return copyFile(filepath.Join(src, project.MetaFile), filepath.Join(p.Root, project.MetaFile))
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
