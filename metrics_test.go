package rest

import (
	"testing"

	"github.com/rabellamy/promstrap/strategy"
	"github.com/stretchr/testify/assert"
)

func TestNewRED(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace string
		want      *strategy.RED
		wantErr   bool
	}{
		"valid namespace": {
			namespace: "test_metrics",
			want:      &strategy.RED{},
			wantErr:   false,
		},
		"empty namespace": {
			namespace: "",
			want:      nil,
			wantErr:   true,
		},
		"invalid namespace": {
			namespace: "123invalid",
			want:      &strategy.RED{},
			wantErr:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRED(tt.namespace, "http", []string{"path", "verb"}, []string{"path"})

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
