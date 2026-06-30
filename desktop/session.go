package main

import (
	"sync"

	"github.com/tanlian/agent_nova/internal/app"
)

// projectSession 复用当前小说的单一 DB 连接，避免 Wails 并行 binding 导致 SQLITE_BUSY。
type projectSession struct {
	mu   sync.Mutex
	root string
	ctx  *app.Context
}

func (s *projectSession) withActive(root string, fn func(*app.Context) error) error {
	if root == "" {
		return errNoActiveProject
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	actx, err := s.ensureLocked(root)
	if err != nil {
		return err
	}
	return fn(actx)
}

func (s *projectSession) ensureLocked(root string) (*app.Context, error) {
	if s.ctx != nil && s.root == root {
		return s.ctx, nil
	}
	s.closeLocked()
	actx, err := app.LoadContext(root)
	if err != nil {
		return nil, err
	}
	s.root = root
	s.ctx = actx
	return actx, nil
}

func (s *projectSession) invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closeLocked()
}

func (s *projectSession) closeLocked() {
	if s.ctx != nil {
		_ = s.ctx.Close()
		s.ctx = nil
		s.root = ""
	}
}
