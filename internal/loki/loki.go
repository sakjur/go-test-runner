package loki

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/go-test-runner/internal/loki/logproto"
	"github.com/grafana/go-test-runner/internal/loki/lokihttp"
	"github.com/grafana/go-test-runner/internal/tests"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
)

func SendToLoki(r *tests.Run) error {
	var lokiURL flagext.URLValue
	err := lokiURL.Set("http://localhost:3100/loki/api/v1/push")
	if err != nil {
		return err
	}

	loki, err := lokihttp.New(prometheus.NewRegistry(), lokihttp.Config{
		URL:           lokiURL,
		BatchWait:     time.Second,
		BatchSize:     1500,
		Client:        config.HTTPClientConfig{},
		BackoffConfig: backoff.Config{},
		Timeout:       3 * time.Second,
	}, log.NewLogfmtLogger(os.Stderr))
	if err != nil {
		return err
	}

	channel := loki.Chan()
	for _, event := range r.Events {
		printer, ok := event.Payload.(tests.Print)
		if !ok {
			continue
		}

		kvs := []any{
			"msg", printer.Line,
			"package", event.Package,
		}

		for key, value := range r.Fields {
			kvs = append(kvs, key, value)
		}

		if event.Test != "" {
			test, err := r.Get(event.Package, event.Test)
			if err != nil {
				fmt.Printf("Got error: %s", err)
				continue
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
				Timestamp: event.Timestamp,
				Line:      buf.String(),
			},
		}
	}

	loki.Stop()
	return nil
}
