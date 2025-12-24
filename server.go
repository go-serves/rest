package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type httpServer struct {
	mainServer    http.Server
	metricsServer http.Server
	ctx           context.Context
	logger        *slog.Logger
	config        Config
}

type Routes map[string]func(w http.ResponseWriter, r *http.Request)

func CreateRoutes(routes Routes) *http.ServeMux {
	mux := http.NewServeMux()

	for path, route := range routes {
		mux.HandleFunc(path, route)
	}

	health := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Status: %v", http.StatusOK)
	}

	mux.HandleFunc("/health", health)

	return mux
}

func NewServer(ctx context.Context, config Config, routes Routes, logger *slog.Logger) (*httpServer, error) {
	mainMux := CreateRoutes(routes)
	handler, err := NewREDMiddleware(config.Namespace, mainMux)
	if err != nil {
		return nil, err
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	s := httpServer{
		mainServer: http.Server{
			Addr:           config.APIHost,
			Handler:        handler,
			ReadTimeout:    config.ReadTimeout,
			WriteTimeout:   config.WriteTimeout,
			IdleTimeout:    config.IdleTimeout,
			MaxHeaderBytes: config.MaxHeaderBytes,
		},
		metricsServer: http.Server{
			Addr:    config.MetricsHost,
			Handler: metricsMux,
		},
		logger: logger,
		ctx:    ctx,
		config: config,
	}

	return &s, nil
}

func (s *httpServer) Run() error {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	return s.run(shutdown)
}

func (s *httpServer) run(shutdown <-chan os.Signal) error {
	// With a buffer of 2, matching the number of producers, guarantees
	// that neither goroutine will ever block on sending
	serverErrors := make(chan error, 2)

	go func() {
		s.logger.Info("startup", "status", "metrics server started", "host", s.config.MetricsHost)
		serverErrors <- s.metricsServer.ListenAndServe()
	}()

	go func() {
		s.logger.Info("startup", "status", "main server started", "host", s.config.APIHost)
		serverErrors <- s.mainServer.ListenAndServe()
	}()

	select {
	case <-s.ctx.Done():
		return s.shutdownServers(s.ctx, nil)
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		ctx, cancel := context.WithTimeout(s.ctx, s.config.ShutdownTimeout)
		defer cancel()

		return s.shutdownServers(ctx, sig)
	}
}

func (s *httpServer) shutdownServers(ctx context.Context, signal os.Signal) error {
	servers := []struct {
		name   string
		server *http.Server
	}{
		{"main", &s.mainServer},
		{"metrics", &s.metricsServer},
	}

	// We can assume that if the signal is nil, it is context cancelled
	// by internal application logic
	sig := "context_cancelled"
	if signal != nil {
		sig = signal.String()
	}

	for _, srv := range servers {
		s.logger.Info("shutdown", "server", srv.name, "status", "shutdown started", "signal", sig)
		defer s.logger.Info("shutdown", "server", srv.name, "status", "shutdown complete", "signal", sig)
		if err := srv.server.Shutdown(ctx); err != nil {
			srv.server.Close()
			return fmt.Errorf("%s server could not stopped gracefully: %w", srv.name, err)
		}
	}
	return nil
}
