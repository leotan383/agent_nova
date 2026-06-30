package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tanlian/agent_nova/internal/jobs"
	"github.com/tanlian/agent_nova/internal/project"
	"github.com/tanlian/agent_nova/internal/status"
	"github.com/tanlian/agent_nova/internal/store"
)

type Server struct {
	Project *project.Project
	Store   *store.Store
	Jobs    *jobs.Hub
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/chapters", s.handleChapters)
	mux.HandleFunc("/entities", s.handleEntities)
	mux.HandleFunc("/foreshadows", s.handleForeshadows)
	mux.HandleFunc("/memories", s.handleMemories)
	mux.HandleFunc("/cool-points", s.handleCoolPoints)
	mux.HandleFunc("/reviews", s.handleReviews)
	mux.HandleFunc("/chapter/", s.handleChapterContent)
	mux.HandleFunc("/write", s.handleWrite)
	mux.HandleFunc("/write/", s.handleWriteJob)
	return mux
}

func (s *Server) handleWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if s.Jobs == nil {
		http.Error(w, "jobs not enabled", 503)
		return
	}
	var req struct {
		Chapter int `json:"chapter"`
		Volume  int `json:"volume"`
	}
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)
	if req.Chapter <= 0 {
		http.Error(w, "chapter required", 400)
		return
	}
	if req.Volume <= 0 {
		req.Volume = 1
	}
	job, err := s.Jobs.StartWrite(r.Context(), req.Chapter, req.Volume)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, job)
}

func (s *Server) handleWriteJob(w http.ResponseWriter, r *http.Request) {
	if s.Jobs == nil {
		http.Error(w, "jobs not enabled", 503)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/write/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "job id required", 400)
		return
	}
	id := parts[0]
	if len(parts) >= 2 && parts[1] == "events" {
		s.handleWriteEvents(w, r, id)
		return
	}
	job, ok := s.Jobs.Get(id)
	if !ok {
		http.Error(w, "not found", 404)
		return
	}
	writeJSON(w, job)
}

func (s *Server) handleWriteEvents(w http.ResponseWriter, r *http.Request, id string) {
	since := 0
	if v := r.URL.Query().Get("since"); v != "" {
		since, _ = strconv.Atoi(v)
	}
	if r.Header.Get("Accept") == "text/event-stream" || r.URL.Query().Get("stream") == "1" {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		cursor := since
		for i := 0; i < 600; i++ {
			events := s.Jobs.EventsSince(id, cursor)
			for _, ev := range events {
				b, _ := json.Marshal(ev)
				fmt.Fprintf(w, "data: %s\n\n", b)
				flusher.Flush()
				cursor++
			}
			job, ok := s.Jobs.Get(id)
			if ok && (job.Status == jobs.StatusDone || job.Status == jobs.StatusFailed) {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		return
	}
	writeJSON(w, s.Jobs.EventsSince(id, since))
}

func (s *Server) handleCoolPoints(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListCoolPoints(0)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleReviews(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListReviews()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	rep := status.Build(s.Project, s.Store, "all")
	writeJSON(w, rep)
}

func (s *Server) handleChapters(w http.ResponseWriter, r *http.Request) {
	chs, err := s.Store.ListChapters()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, chs)
}

func (s *Server) handleEntities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	items, err := s.Store.SearchEntities(q, 100)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleForeshadows(w http.ResponseWriter, r *http.Request) {
	st := r.URL.Query().Get("status")
	items, err := s.Store.ListForeshadows(st)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleMemories(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.QueryMemories("", "", 100)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleChapterContent(w http.ResponseWriter, r *http.Request) {
	numStr := strings.TrimPrefix(r.URL.Path, "/chapter/")
	numStr = strings.TrimSuffix(numStr, "/")
	var num int
	_, _ = fmtSscanf(numStr, &num)
	ch, err := s.Store.GetChapter(num)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	data, err := os.ReadFile(ch.Path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{"chapter": ch, "content": string(data)})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fmtSscanf(s string, n *int) (int, error) {
	var v int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		}
	}
	*n = v
	return 1, nil
}
