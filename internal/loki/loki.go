package loki

import (
	"bytes"
	"os"
	"time"

	"github.com/prometheus/common/config"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/go-test-runner/internal/cfg"
	"github.com/grafana/go-test-runner/internal/loki/logproto"
	"github.com/grafana/go-test-runner/internal/loki/lokihttp"
	"github.com/grafana/go-test-runner/internal/tests"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

type EventSender struct {
	client lokihttp.Client
	r      *tests.Run
}

func New(r *tests.Run, conf cfg.LokiOptions) (*EventSender, error) {
	var lokiURL flagext.URLValue
	err := lokiURL.Set(conf.URL)
	if err != nil {
		return nil, err
	}

	loki, err := lokihttp.New(prometheus.NewRegistry(), lokihttp.Config{
		URL:       lokiURL,
		BatchWait: conf.BatchWait,
		BatchSize: conf.BatchSize,
		Client:    config.HTTPClientConfig{},
		BackoffConfig: backoff.Config{
			MinBackoff: 100 * time.Millisecond,
			MaxBackoff: 2 * time.Second,
			MaxRetries: conf.Retries,
		},
		Timeout: conf.Timeout,
	}, log.NewLogfmtLogger(os.Stderr))
	if err != nil {
		return nil, err
	}

	return &EventSender{client: loki, r: r}, nil
}

func (e EventSender) Handle(event tests.Event) error {
	channel := e.client.Chan()
	printer, ok := event.Payload.(tests.Print)
	if !ok {
		return nil
	}

	kvs := []any{
		"msg", printer.Line,
		"package", event.Package,
	}

	for key, value := range e.r.Fields {
		kvs = append(kvs, key, value)
	}

	if event.Test != "" {
		test, err := e.r.Get(event.Package, event.Test)
		if err != nil {
			return err
		}

		kvs = append(kvs,
			"test", event.Test,
			"state", test.State,
		)
	}

	buf := &bytes.Buffer{}
	logger := log.NewLogfmtLogger(buf)
	logger.Log(kvs...)

	channel <- lokihttp.Entry{
		Labels: model.LabelSet{"source": "go-test-runner"},
		Entry: logproto.Entry{
			//Timestamp: event.Timestamp,
			Timestamp: time.Now(),
			Line:      buf.String(),
		},
	}
	return nil
}

func (e *EventSender) Stop() {
	e.client.Stop()
}
