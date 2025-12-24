package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rabellamy/promstrap/strategy"
	"github.com/rabellamy/server/metrics"
)

// REDMiddleware wraps an HTTP handler to collect RED metrics.
type REDMiddleware struct {
	red  *strategy.RED
	next http.Handler
}

// NewREDMiddleware creates a new RED metrics middleware.
func NewREDMiddleware(namespace string, next http.Handler) (*REDMiddleware, error) {
	red, err := metrics.NewRED(namespace, "http", []string{"path", "verb"}, []string{"path"})
	if err != nil {
		return nil, fmt.Errorf("failed to create RED metrics: %w", err)
	}

	if err := red.Register(); err != nil {
		return nil, fmt.Errorf("failed to register RED metrics: %w", err)
	}

	return &REDMiddleware{
		red:  red,
		next: next,
	}, nil
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// ServeHTTP implements the http.Handler interface.
func (m *REDMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Wrap response writer to capture status code
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Record the request (Rate)
	m.red.Requests.WithLabelValues(r.URL.Path, r.Method).Inc()

	m.next.ServeHTTP(rw, r)

	// Record duration
	duration := time.Since(start).Seconds()
	if m.red.Duration.Histogram != nil {
		m.red.Duration.Histogram.WithLabelValues(r.URL.Path).Observe(duration)
	}
	if m.red.Duration.Summary != nil {
		m.red.Duration.Summary.WithLabelValues(r.URL.Path).Observe(duration)
	}

	// Record errors (status code >= 400)
	if rw.statusCode >= 400 {
		m.red.Errors.WithLabelValues(strconv.Itoa(rw.statusCode)).Inc()
	}
}

// WriteHeader captures the status code and calls the underlying WriteHeader.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
