package app

import (
	"fmt"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
)

type Context struct {
	Config  *config.Config
	Project *project.Project
	Store   *store.Store
}

func LoadContext(projectRoot string) (*Context, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	root := projectRoot
	if root == "" {
		root, err = project.ResolveProjectRoot("")
		if err != nil {
			if cur, e := project.CurrentProjectRoot(); e == nil && cur != "" {
				root = cur
			} else {
				return nil, err
			}
		}
	}
	p, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(p.DBPath())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return &Context{Config: cfg, Project: p, Store: st}, nil
}

func (c *Context) Close() error {
	if c.Store != nil {
		return c.Store.Close()
	}
	return nil
}

func RequireAPIKey(cfg *config.Config) error {
	if cfg.OpenAIAPIKey == "" {
		return fmt.Errorf("未配置 OPENAI_API_KEY，请设置环境变量或运行 nova config set api_key <key>")
	}
	return nil
}
