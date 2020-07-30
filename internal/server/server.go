package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/KeisukeYamashita/spinnaker-github-proxy/internal/config"
	"github.com/KeisukeYamashita/spinnaker-github-proxy/internal/github"
	proxy "github.com/KeisukeYamashita/spinnaker-github-proxy/internal/proxy"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	exitOk    = 0
	exitError = 1
)

var ErrResps = map[int]string{http.StatusForbidden: "authorization header missing", http.StatusBadGateway: "upstream server err", http.StatusBadRequest: "bad request"}

type Option func(s *Server)

func WithLogger(l *zap.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

func Run() {
	os.Exit(run(context.Background()))
}

type Server struct {
	proxy  proxy.Proxy
	mux    *http.ServeMux
	server *http.Server
	logger *zap.Logger
}

type ServerConfig struct {
	ghClient   github.Client
	allowedOrg string
}

func (s *Server) registerHandlers() {
	s.mux.Handle("/healthz", s.healthCheckHandler())
	s.mux.Handle("/", s.proxy.OAuthProxyHandler())
}

func (s *Server) healthCheckHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (s *Server) Serve(conn net.Listener) error {
	server := &http.Server{
		Handler: s.mux,
	}
	s.server = server
	if err := server.Serve(conn); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) GracefulStop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func newServer(cfg *ServerConfig, opts ...Option) (*Server, error) {
	s := &Server{}
	for _, opt := range opts {
		opt(s)
	}

	var p proxy.Proxy
	if s.logger != nil {
		p = proxy.NewProxyHandler(cfg.ghClient, proxy.WithOrganizationRestriction(cfg.allowedOrg), proxy.WithProxyLogger(s.logger.Named("proxy")))
	}

	server := &Server{
		mux:   http.NewServeMux(),
		proxy: p,
	}

	server.registerHandlers()
	return server, nil
}

func run(ctx context.Context) int {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup logger: %s\n", err)
		return exitError
	}
	defer logger.Sync()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		return exitError
	}

	conn, err := net.Listen("tcp", cfg.Address())
	if err != nil {
		return exitError
	}
	logger.Info("http server listening", zap.String("address", cfg.Address()))

	ghClient := github.NewClient()
	httpServer, err := newServer(
		&ServerConfig{
			ghClient:   ghClient,
			allowedOrg: cfg.Organization,
		}, WithLogger(logger))
	if err != nil {
		return exitError
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg, ctx := errgroup.WithContext(ctx)
	wg.Go(func() error { return httpServer.Serve(conn) })

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
	select {
	case <-sigCh:
	case <-ctx.Done():
	}

	// Gracefully shutdown server.
	if err := httpServer.GracefulStop(ctx); err != nil {
		return exitError
	}

	cancel()
	if err := wg.Wait(); err != nil {
		return exitError
	}

	return exitOk
}
