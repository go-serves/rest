package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rabellamy/server/metrics"
	"github.com/stretchr/testify/assert"
)

func TestNewREDMiddleware(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace string
		next      http.Handler
		want      *REDMiddleware
		wantErr   bool
		setup     func() error
	}{
		"base case": {
			namespace: "foo",
			next:      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			want:      &REDMiddleware{},
			wantErr:   false,
		},
		"empty namespace": {
			namespace: "",
			next:      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			want:      nil,
			wantErr:   true,
		},
		"invalid namespace: starts with a number": {
			namespace: "123invalid",
			next:      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			want:      nil,
			wantErr:   true,
		},
		"invalid namespace: contains invalid characters": {
			namespace: "invalid-char!",
			next:      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			want:      nil,
			wantErr:   true,
		},
		"nil handler": {
			namespace: "bar",
			next:      nil,
			want:      &REDMiddleware{},
			wantErr:   false,
		},
		"register fail": {
			namespace: "test_register_fail",
			next:      nil,
			want:      nil,
			wantErr:   true,
			setup: func() error {
				red, err := metrics.NewRED("test_register_fail", "http", []string{"path", "verb"}, []string{"path"})
				if err != nil {
					return err
				}
				return red.Register()
			},
		},
		"invalid namespace": {
			namespace: "invalid-namespace",
			next:      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			want:      nil,
			wantErr:   true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatal(err)
				}
			}
			got, gotErr := NewREDMiddleware(tt.namespace, tt.next)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("NewREDMiddleware() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("NewREDMiddleware() succeeded unexpectedly")
			}
			assert.EqualExportedValues(t, *tt.want, *got)
		})
	}
}

func TestREDMiddlewareServeHTTP(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace  string
		method     string
		path       string
		statusCode int
		wantErr    bool
	}{
		"200 OK": {
			namespace:  "test_serve_http_foo",
			method:     http.MethodGet,
			path:       "/test",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		"400 Bad Request": {
			namespace:  "test_serve_http_bar",
			method:     http.MethodGet,
			path:       "/bad",
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
		"500 Internal Server Error": {
			namespace:  "test_serve_http_baz",
			method:     http.MethodGet,
			path:       "/error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
		"POST request": {
			namespace:  "test_serve_http_jazz",
			method:     http.MethodPost,
			path:       "/post",
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			middleware, err := NewREDMiddleware(tt.namespace, handler)
			assert.NoError(t, err)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			middleware.ServeHTTP(rec, req)

			assert.Equal(t, tt.statusCode, rec.Code)
		})
	}
}
