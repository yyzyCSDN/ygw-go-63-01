package main

import (
	"context"
	"net/http"
	"time"

	"modelrouter/internal/health"
	"modelrouter/internal/gateway"
	"modelrouter/internal/metric"
	"modelrouter/internal/registry"
	"modelrouter/web"
)

// Server owns the HTTP listener and the gateway components.
type Server struct {
	cfg     *Config
	gw      *gateway.Gateway
	metrics *metric.Metrics
	health  *health.Tracker
	checker *health.Checker
	reg     *registry.Registry
	http    *http.Server
	started time.Time
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewServer builds the HTTP mux and server.
func NewServer(
	cfg *Config,
	gw *gateway.Gateway,
	metrics *metric.Metrics,
	healthTracker *health.Tracker,
	checker *health.Checker,
	reg *registry.Registry,
) *Server {
	s := &Server{
		cfg:     cfg,
		gw:      gw,
		metrics: metrics,
		health:  healthTracker,
		checker: checker,
		reg:     reg,
		started: time.Now(),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/healthz", s.handleHealthz)
	mux.HandleFunc("GET /v1/console", s.handleConsole)
	mux.HandleFunc("POST /v1/models/{model}/infer", s.handleInfer)
	mux.HandleFunc("POST /v1/admin/register", s.handleRegister)
	mux.HandleFunc("POST /v1/admin/publish", s.handlePublish)
	mux.HandleFunc("POST /v1/admin/alias", s.handleAlias)
	mux.HandleFunc("POST /v1/admin/retire", s.handleRetire)
	mux.HandleFunc("POST /v1/admin/sync", s.handleSync)
	mux.HandleFunc("GET /v1/admin/models", s.handleModels)
	mux.HandleFunc("GET /v1/admin/instances", s.handleInstances)
	mux.HandleFunc("GET /v1/metrics", s.handleMetrics)
	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// Start launches the HTTP listener.
func (s *Server) Start() error {
	go s.checker.Run(s.ctx)
	go func() {
		_ = s.http.ListenAndServe()
	}()
	return nil
}

// Shutdown gracefully stops the HTTP listener.
func (s *Server) Shutdown(ctx context.Context) error {
	s.cancel()
	return s.http.Shutdown(ctx)
}

// ServeConsole returns the embedded console page.
func ServeConsole() ([]byte, error) {
	return web.ConsoleHTML()
}
