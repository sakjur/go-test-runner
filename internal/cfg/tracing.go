package cfg

import (
	"errors"
	"fmt"
	"strconv"
)

const (
	TracingKind         = "TRACING_KIND"
	TracingURL          = "TRACING_URL"
	TracingLogsAsEvents = "TRACING_LOGS_AS_EVENTS"
)

type TracingOptions struct {
	Kind         string
	URL          string
	LogsAsEvents bool
}

func (c Config) Tracing() (TracingOptions, error) {
	kind, rawKindErr := c.Get(TracingKind)
	url, urlErr := c.Get(TracingURL)
	rawLogsAsEvents, rawLogsAsEventsErr := c.Get(TracingLogsAsEvents)

	if err := errors.Join(urlErr, rawKindErr, rawLogsAsEventsErr); err != nil {
		return TracingOptions{}, fmt.Errorf("failed to get tracing configuration options: %w", err)
	}

	var kindErr error
	if kind != "jaeger" {
		kindErr = fmt.Errorf("kind must be 'jaeger'")
	}
	logsAsEvents, logsAsEventsErr := strconv.ParseBool(rawLogsAsEvents)

	if err := errors.Join(kindErr, logsAsEventsErr); err != nil {
		return TracingOptions{}, fmt.Errorf("failed to parse tracing configuration options: %w", err)
	}

	return TracingOptions{
		Kind:         TracingKind,
		URL:          url,
		LogsAsEvents: logsAsEvents,
	}, nil
}
