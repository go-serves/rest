package rest

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// We cannot run this in parallel because it modifies environment variables
	// t.Parallel()

	tests := map[string]struct {
		prefix string
		env    map[string]string
		want   Config
		err    error
	}{
		"defaults": {
			prefix: "test_defaults",
			env:    map[string]string{},
			want: Config{
				ReadTimeout:        5 * time.Second,
				WriteTimeout:       10 * time.Second,
				IdleTimeout:        120 * time.Second,
				ShutdownTimeout:    20 * time.Second,
				APIHost:            "0.0.0.0:3000",
				DebugHost:          "0.0.0.0:3010",
				MetricsHost:        "0.0.0.0:2112",
				CorsAllowedOrigins: []string{"*"},
				MaxHeaderBytes:     0,
				Build:              "dev",
				Desc:               "example server",
				Namespace:          "test_defaults",
			},
			err: nil,
		},
		"env vars set": {
			prefix: "test_env",
			env: map[string]string{
				"TEST_ENV_APIHOST":   "127.0.0.1:9090",
				"TEST_ENV_NAMESPACE": "custom_namespace",
				"TEST_ENV_BUILD":     "prod",
				"TEST_ENV_DEBUGHOST": "127.0.0.1:9091",
			},
			want: Config{
				ReadTimeout:        5 * time.Second,
				WriteTimeout:       10 * time.Second,
				IdleTimeout:        120 * time.Second,
				ShutdownTimeout:    20 * time.Second,
				APIHost:            "127.0.0.1:9090",
				DebugHost:          "127.0.0.1:9091",
				MetricsHost:        "0.0.0.0:2112",
				CorsAllowedOrigins: []string{"*"},
				MaxHeaderBytes:     0,
				Build:              "prod",
				Desc:               "example server",
				Namespace:          "custom_namespace",
			},
			err: nil,
		},
		"invalid duration format": {
			prefix: "test_invalid_dur",
			env: map[string]string{
				"TEST_INVALID_DUR_READTIMEOUT": "not-a-duration",
			},
			want: Config{},
			err:  assert.AnError,
		},
		"invalid int format": {
			prefix: "test_invalid_int",
			env: map[string]string{
				"TEST_INVALID_INT_MAXHEADERBYTES": "not-an-int",
			},
			want: Config{},
			err:  assert.AnError,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got, err := LoadConfig(tt.prefix)
			if tt.err != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
