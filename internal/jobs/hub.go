package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tanlian/agent_nova/internal/config"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/store"
	"github.com/tanlian/agent_nova/internal/workflows"
)

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusDone    Status = "done"
	StatusFailed  Status = "failed"
)

type WriteJob struct {
	ID        string    `json:"id"`
	Chapter   int       `json:"chapter"`
	Volume    int       `json:"volume"`
	Status    Status    `json:"status"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
	Events    []Event   `json:"-"`
}

type Event struct {
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	Payload string    `json:"payload"`
}

type Hub struct {
	mu   sync.RWMutex
	jobs map[string]*WriteJob
	cfg  *config.Config
	proj *project.Project
	st   *store.Store
}

func NewHub(cfg *config.Config, p *project.Project, st *store.Store) *Hub {
	return &Hub{jobs: map[string]*WriteJob{}, cfg: cfg, proj: p, st: st}
}

func (h *Hub) StartWrite(ctx context.Context, chapter, volume int) (*WriteJob, error) {
	id := fmt.Sprintf("write-%d-%d", chapter, time.Now().Unix())
	job := &WriteJob{ID: id, Chapter: chapter, Volume: volume, Status: StatusPending, UpdatedAt: time.Now().UTC()}
	h.mu.Lock()
	h.jobs[id] = job
	h.mu.Unlock()
	go h.runWrite(context.Background(), job)
	return job, nil
}

func (h *Hub) runWrite(ctx context.Context, job *WriteJob) {
	h.setStatus(job, StatusRunning, "")
	wf := workflows.NewWriteWorkflow(h.cfg, h.proj, h.st)
	rep, err := wf.WriteChapter(ctx, h.proj, h.st, workflows.WriteOptions{
		Chapter: job.Chapter, Volume: job.Volume, Stream: true,
		OnDelta: func(s string) error {
			h.appendEvent(job, "delta", s)
			return nil
		},
	})
	if err != nil {
		h.setStatus(job, StatusFailed, err.Error())
		h.appendEvent(job, "error", err.Error())
		return
	}
	b, _ := json.Marshal(rep)
	h.appendEvent(job, "done", string(b))
	h.setStatus(job, StatusDone, "")
}

func (h *Hub) setStatus(job *WriteJob, st Status, errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	job.Status = st
	job.Error = errMsg
	job.UpdatedAt = time.Now().UTC()
	h.appendEventLocked(job, "status", string(st))
}

func (h *Hub) appendEvent(job *WriteJob, typ, payload string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.appendEventLocked(job, typ, payload)
}

func (h *Hub) appendEventLocked(job *WriteJob, typ, payload string) {
	job.Events = append(job.Events, Event{Time: time.Now().UTC(), Type: typ, Payload: payload})
}

func (h *Hub) Get(id string) (*WriteJob, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	j, ok := h.jobs[id]
	return j, ok
}

func (h *Hub) EventsSince(id string, since int) []Event {
	h.mu.RLock()
	defer h.mu.RUnlock()
	j, ok := h.jobs[id]
	if !ok || since >= len(j.Events) {
		return nil
	}
	return j.Events[since:]
}
