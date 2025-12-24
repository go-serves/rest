package rest

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateRoutes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		routes Routes
		path   string
		want   int
	}{
		"health check exists by default": {
			routes: Routes{},
			path:   "/health",
			want:   http.StatusOK,
		},
		"custom route works": {
			routes: Routes{
				"/foo": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusTeapot)
				},
			},
			path: "/foo",
			want: http.StatusTeapot,
		},
		"custom route does not overwrite health check (unless explicitly set)": {
			routes: Routes{
				"/bar": func(w http.ResponseWriter, r *http.Request) {},
			},
			path: "/health",
			want: http.StatusOK,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mux := CreateRoutes(tt.routes)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
		})
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config  Config
		routes  Routes
		wantErr bool
	}{
		"valid config": {
			config: Config{
				Namespace: "test_server",
				APIHost:   "localhost:8080",
			},
			routes:  Routes{},
			wantErr: false,
		},
		"invalid namespace": {
			config: Config{
				Namespace: "123invalid",
			},
			routes:  Routes{},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			got, err := NewServer(context.Background(), tt.config, tt.routes, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config     Config
		wantErr    bool
		cancelCtx  bool
		sendSignal bool
		preCancel  bool
	}{
		"context cancellation shutdown": {
			config: Config{
				Namespace:       "test_run_ctx",
				APIHost:         "localhost:0",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr:    false,
			cancelCtx:  true,
			sendSignal: false,
		},
		"signal shutdown": {
			config: Config{
				Namespace:       "test_run_signal",
				APIHost:         "localhost:0",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr:    false,
			cancelCtx:  false,
			sendSignal: true,
		},
		"invalid metrics host": {
			config: Config{
				Namespace:       "test_run_invalid",
				APIHost:         "localhost:0",
				MetricsHost:     "invalid-host:port",
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr:    true,
			cancelCtx:  false,
			sendSignal: false,
		},
		"context pre-cancelled": {
			config: Config{
				Namespace:   "test_run_pre_cancel",
				APIHost:     "localhost:0",
				MetricsHost: "localhost:0",
			},
			wantErr:   false,
			preCancel: true,
		},
		"invalid api host": {
			config: Config{
				Namespace:       "test_run_invalid_api",
				APIHost:         "invalid-host:port",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			routes := Routes{}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.preCancel {
				cancel()
			}

			server, err := NewServer(ctx, tt.config, routes, logger)
			assert.NoError(t, err)

			shutdownChan := make(chan os.Signal, 1)
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.run(shutdownChan)
			}()

			// Give server time to start
			time.Sleep(50 * time.Millisecond)

			if tt.sendSignal {
				// Send mock signal directly to the channel
				shutdownChan <- os.Interrupt
			} else if tt.cancelCtx {
				cancel()
			}

			if tt.wantErr {
				err = <-errChan
				assert.Error(t, err)
			} else {
				err = <-errChan
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunExported(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config Config
	}{
		"context cancellation": {
			config: Config{
				Namespace:       "test_run_exported",
				APIHost:         "localhost:0",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			ctx, cancel := context.WithCancel(context.Background())

			server, err := NewServer(ctx, tt.config, Routes{}, logger)
			assert.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run()
			}()

			// Give it a moment to start
			time.Sleep(50 * time.Millisecond)

			// Cancel context to trigger shutdown
			cancel()

			err = <-errChan
			assert.NoError(t, err)
		})
	}
}

func TestShutdownServers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ctxTimeout time.Duration
		wantErr    bool
		signal     os.Signal
	}{
		"successful shutdown": {
			ctxTimeout: 5 * time.Second,
			wantErr:    false,
			signal:     nil,
		},
		"context already cancelled": {
			ctxTimeout: 0, // Instant timeout/cancellation
			wantErr:    true,
			signal:     os.Interrupt,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a listener to know the port
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)

			mux := http.NewServeMux()
			// Add a blocking handler for the error case
			blockCh := make(chan struct{})
			mux.HandleFunc("/block", func(w http.ResponseWriter, r *http.Request) {
				<-blockCh
			})

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			s := &httpServer{
				logger: logger,
				mainServer: http.Server{
					Handler: mux,
				},
				metricsServer: http.Server{
					Addr: "127.0.0.1:0",
				},
			}

			// Start main server
			go s.mainServer.Serve(ln)

			// Start metrics server (just to have it running)
			go s.metricsServer.ListenAndServe()

			// If we expect an error (timeout/cancellation), we need the server to be busy
			// so Shutdown doesn't return immediately.
			if tt.wantErr {
				go func() {
					// Make a request that will block
					http.Get("http://" + ln.Addr().String() + "/block")
				}()
				// Give the request time to reach the handler
				time.Sleep(50 * time.Millisecond)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			if tt.ctxTimeout == 0 {
				cancel()
			} else {
				defer cancel()
			}

			err = s.shutdownServers(ctx, tt.signal)

			// Unblock the handler to clean up
			close(blockCh)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
