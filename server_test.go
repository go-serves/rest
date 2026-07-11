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

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestCreateRoutes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupGateway func() *runtime.ServeMux
		method       string
		path         string
		wantCode     int
	}{
		"health check exists": {
			setupGateway: func() *runtime.ServeMux { return nil },
			method:       http.MethodGet,
			path:         "/health",
			wantCode:     http.StatusOK,
		},
		"gateway mux mounted": {
			setupGateway: func() *runtime.ServeMux {
				gatewayMux := runtime.NewServeMux()
				_ = gatewayMux.HandlePath("GET", "/test", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
					w.WriteHeader(http.StatusTeapot)
				})
				return gatewayMux
			},
			method:   http.MethodGet,
			path:     "/test",
			wantCode: http.StatusTeapot,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			mux := CreateRoutes(tt.setupGateway())
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantCode, rec.Code)
		})
	}
}

func TestWithMeterProvider(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		provider otelmetric.MeterProvider
	}{
		"sets meter provider": {
			provider: metric.NewMeterProvider(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := &ServerOptions{}
			opt := WithMeterProvider(tt.provider)
			opt(opts)
			assert.Equal(t, tt.provider, opts.meterProvider)
		})
	}
}

func TestWithTracerProvider(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		provider oteltrace.TracerProvider
	}{
		"sets tracer provider": {
			provider: trace.NewTracerProvider(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := &ServerOptions{}
			opt := WithTracerProvider(tt.provider)
			opt(opts)
			assert.Equal(t, tt.provider, opts.tracerProvider)
		})
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config  Config
		service Service
		opts    []ServerOption
		wantErr bool
	}{
		"valid config": {
			config: Config{
				Namespace: "test_server",
				APIHost:   "localhost:8080",
			},
			service: nil,
			wantErr: false,
		},
		"with custom providers": {
			config: Config{
				Namespace: "test_server_custom",
				APIHost:   "localhost:8081",
			},
			service: nil,
			opts: []ServerOption{
				WithMeterProvider(metric.NewMeterProvider()),
				WithTracerProvider(trace.NewTracerProvider()),
			},
			wantErr: false,
		},
		"with trace exporter endpoint": {
			config: Config{
				Namespace:             "test_server_trace_mock",
				APIHost:               "localhost:8082",
				TraceExporterEndpoint: "dummy:4317",
			},
			opts: []ServerOption{
				withTraceExporter(func(ctx context.Context) (trace.SpanExporter, error) {
					return &mockSpanExporter{}, nil
				}),
			},
			service: nil,
			wantErr: false,
		},
		"with real trace exporter endpoint": {
			config: Config{
				Namespace:             "test_server_trace_real",
				APIHost:               "localhost:8083",
				TraceExporterEndpoint: "dummy:4317",
			},
			service: nil,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			got, err := NewServer(context.Background(), tt.config, tt.service, logger, tt.opts...)

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

func TestNewServer_PrometheusError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config    Config
		setupFail func() ServerOption
		wantErr   string
	}{
		"prometheus exporter error": {
			config: Config{
				Namespace: "test_server",
				APIHost:   "localhost:8080",
			},
			setupFail: func() ServerOption {
				return withPrometheusExporter(func() (metric.Reader, error) {
					return nil, assert.AnError
				})
			},
			wantErr: "failed to initialize prometheus exporter",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			got, err := NewServer(context.Background(), tt.config, nil, logger, tt.setupFail())
			assert.Error(t, err)
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewServer_TraceError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config    Config
		setupFail func() ServerOption
		wantErr   string
	}{
		"trace exporter error": {
			config: Config{
				Namespace:             "test_server",
				APIHost:               "localhost:8080",
				TraceExporterEndpoint: "dummy:4317",
			},
			setupFail: func() ServerOption {
				return withTraceExporter(func(ctx context.Context) (trace.SpanExporter, error) {
					return nil, assert.AnError
				})
			},
			wantErr: "failed to initialize default trace exporter",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			got, err := NewServer(context.Background(), tt.config, nil, logger, tt.setupFail())
			assert.Error(t, err)
			assert.Nil(t, got)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func withPrometheusExporter(f func() (metric.Reader, error)) ServerOption {
	return func(o *ServerOptions) {
		o.newPrometheusExporter = f
	}
}

func withTraceExporter(f func(context.Context) (trace.SpanExporter, error)) ServerOption {
	return func(o *ServerOptions) {
		o.newTraceExporter = f
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

			var service Service
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.preCancel {
				cancel()
			}

			server, err := NewServer(ctx, tt.config, service, logger)
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

			server, err := NewServer(ctx, tt.config, nil, logger)
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

type mockSpanExporter struct{}

func (e *mockSpanExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return nil
}

func (e *mockSpanExporter) Shutdown(ctx context.Context) error {
	return nil
}

type mockService struct {
	registerGateway func(context.Context, *runtime.ServeMux) error
}

func (m *mockService) RegisterGateway(ctx context.Context, mux *runtime.ServeMux) error {
	if m.registerGateway != nil {
		return m.registerGateway(ctx, mux)
	}
	return nil
}

func TestWithService(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := Config{
		APIHost:         ":0",
		MetricsHost:     ":0",
		ShutdownTimeout: 100 * time.Millisecond,
	}

	tests := map[string]struct {
		setupMock        func(*mockService, *bool)
		wantErr          bool
		wantGwRegistered bool
	}{
		"valid proto route": {
			setupMock: func(m *mockService, gwReg *bool) {
				m.registerGateway = func(ctx context.Context, mux *runtime.ServeMux) error {
					*gwReg = true
					return nil
				}
			},
			wantErr:          false,
			wantGwRegistered: true,
		},
		"gateway error": {
			setupMock: func(m *mockService, gwReg *bool) {
				m.registerGateway = func(ctx context.Context, mux *runtime.ServeMux) error {
					*gwReg = true
					return assert.AnError
				}
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var gatewayRegistered bool

			mockSvc := &mockService{}
			tt.setupMock(mockSvc, &gatewayRegistered)

			server, err := NewServer(
				context.Background(),
				cfg,
				mockSvc,
				logger,
				WithMeterProvider(metric.NewMeterProvider()),
				WithTracerProvider(trace.NewTracerProvider()),
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, server)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, server)
				assert.Equal(t, tt.wantGwRegistered, gatewayRegistered)
			}
		})
	}
}
