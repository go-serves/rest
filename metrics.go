package rest

import (
	"fmt"
	"regexp"

	"github.com/rabellamy/promstrap/strategy"
)

// NewRED creates a new RED metrics instance.
func NewRED(namespace, requestType string, requestLabels, durationLabels []string) (*strategy.RED, error) {
	// regex matches Prometheus metric name limits
	// see: https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	metricNameRegex := regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	if !metricNameRegex.MatchString(namespace) {
		return nil, fmt.Errorf("namespace must match %s", metricNameRegex.String())
	}

	red, err := strategy.NewRED(strategy.REDOpts{
		Namespace: namespace,
		RequestsOpt: strategy.REDRequestsOpt{
			RequestType:   requestType,
			RequestLabels: requestLabels,
		},
		ErrorsOpt: strategy.REDErrorsOpt{
			ErrorLabels: []string{"error"},
		},
		DurationOpt: strategy.REDDurationOpt{
			DurationLabels: durationLabels,
		},
	})

	if err != nil {
		return nil, err
	}

	return red, nil
}
