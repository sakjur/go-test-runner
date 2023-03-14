package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
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

	logFields := tags{}
	flag.Var(&logFields, "t", "Add a key=value pair to the log output for each test")
	flag.Parse()

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
	}

	t := &tests.Tracer{
		Run:    r,
		Tracer: tp.Tracer("go-test-runner"),
	}

	traceID := t.Report(context.Background())
	fmt.Println(traceID)

	logFields["traceID"] = traceID

	err = sendToLoki(r, logFields)
	if err != nil {
		panic(fmt.Errorf("got error when trying to send log to loki: %w", err))
	}
	tp.ForceFlush(context.Background())
}

func sendToLoki(r *tests.Run, logFields tags) error {
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

		for key, value := range logFields {
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

type tags map[string]string

func (t *tags) String() string {
	ts := make([]string, 0, len(*t))
	for key, value := range *t {
		ts = append(ts, fmt.Sprintf("-t %s=%s", key, strconv.Quote(value)))
	}
	return strings.Join(ts, " ")
}

func (t *tags) Set(s string) error {
	values := strings.SplitN(s, "=", 2)
	if len(values) != 2 {
		return fmt.Errorf("expected tags to have the format 'key=value'")
	}

	key := values[0]
	val := values[1]

	if strings.Contains(key, " ") {
		return fmt.Errorf("keys must not contain spaces")
	}

	if strings.HasPrefix(val, "\"") {
		var err error
		val, err = strconv.Unquote(val)
		if err != nil {
			return fmt.Errorf("failed to unquote value for tag %s", key)
		}
	}
	(*t)[key] = val
	return nil
}
