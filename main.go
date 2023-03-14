package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/grafana/go-test-runner/internal/loki/logproto"
	"github.com/prometheus/common/model"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/backoff"
	"github.com/grafana/dskit/flagext"
	"github.com/grafana/go-test-runner/internal/loki/lokihttp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/config"

	"github.com/grafana/go-test-runner/internal/tests"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func main() {
	tp, err := tracerProvider("http://localhost:14268/api/traces")
	if err != nil {
		panic(err)
	}

	events := []tests.Event{}
	r := tests.New()
	goJSON := tests.NewGoJSON(os.Stdin)
	for {
		es, err := goJSON.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		for _, e := range es {
			if printer, ok := e.Payload.(tests.Print); ok {
				fmt.Print(printer.Line)
			}
		}
		r.Add(context.Background(), es...)
		events = append(events, es...)
	}

	t := &tests.Tracer{
		Run:    r,
		Tracer: tp.Tracer("go-test-runner"),
	}

	traceID := t.Report(context.Background())
	fmt.Println(traceID)

	var lokiURL flagext.URLValue
	err = lokiURL.Set("https://loki.e127.se/loki/api/v1/push")
	if err != nil {
		panic(err)
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
		panic(err)
	}

	channel := loki.Chan()
	for _, event := range events {
		printer, ok := event.Payload.(tests.Print)
		if !ok {
			continue
		}

		kvs := []any{
			"msg", printer.Line,
			"traceID", traceID,
			"package", event.Package,
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

	tp.ForceFlush(context.Background())
	loki.Stop()
}

func tracerProvider(url string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("go-test-runner"),
		)),
	)
	return tp, nil
}
