package daemon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/tanlian/agent_nova/internal/api"
	"github.com/tanlian/agent_nova/internal/app"
	"github.com/tanlian/agent_nova/internal/jobs"
	"github.com/tanlian/agent_nova/web/dashboard"
)

const Version = "0.1.0"

type Options struct {
	ProjectRoot string
	Port        int
	SocketPath  string
}

func RunDashboard(ctx context.Context, opts Options) error {
	actx, err := app.LoadContext(opts.ProjectRoot)
	if err != nil {
		return err
	}
	defer actx.Close()

	hub := jobs.NewHub(actx.Config, actx.Project, actx.Store)
	srv := &api.Server{Project: actx.Project, Store: actx.Store, Jobs: hub}
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", srv.Handler()))
	mux.Handle("/", http.FileServer(http.FS(dashboard.FS())))

	addr := fmt.Sprintf("127.0.0.1:%d", opts.Port)
	if opts.Port <= 0 {
		opts.Port = 8765
		addr = fmt.Sprintf("127.0.0.1:%d", opts.Port)
	}

	httpServer := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Dashboard: http://%s/\n", addr)
	return httpServer.ListenAndServe()
}

func RunDaemon(ctx context.Context, opts Options) error {
	actx, err := app.LoadContext(opts.ProjectRoot)
	if err != nil {
		return err
	}
	defer actx.Close()

	hub := jobs.NewHub(actx.Config, actx.Project, actx.Store)
	srv := &api.Server{Project: actx.Project, Store: actx.Store, Jobs: hub}
	mux := http.NewServeMux()
	mux.Handle("/api/", http.StripPrefix("/api", srv.Handler()))
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))

	if opts.SocketPath != "" {
		_ = os.Remove(opts.SocketPath)
		if err := os.MkdirAll(filepath.Dir(opts.SocketPath), 0o755); err != nil {
			return err
		}
		ln, err := net.Listen("unix", opts.SocketPath)
		if err != nil {
			return err
		}
		fmt.Printf("Daemon listening on %s\n", opts.SocketPath)
		httpServer := &http.Server{Handler: mux}
		go func() {
			<-ctx.Done()
			_ = httpServer.Close()
		}()
		return httpServer.Serve(ln)
	}

	port := opts.Port
	if port <= 0 {
		port = 8787
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	httpServer := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		_ = httpServer.Shutdown(context.Background())
	}()
	fmt.Printf("Daemon: http://%s/\n", addr)
	return httpServer.ListenAndServe()
}
