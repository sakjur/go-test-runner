package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/grafana/go-test-runner/internal/grafana/explore"
	"github.com/grafana/go-test-runner/internal/loki"
	"github.com/grafana/go-test-runner/internal/tests"
	"github.com/grafana/go-test-runner/internal/tracing"
)

func main() {
	fields := tests.Tags{}
	flag.Var(&fields, "t", "Add a key=value pair to the log output for each test")
	flag.Parse()

	tp, err := tracing.JaegerProvider("http://localhost:14268/api/traces")
	defer tp.ForceFlush(context.Background())
	if err != nil {
		panic(err)
	}
	tracer := tp.Tracer("go-test-runner")

	r := tests.New(tracer, fields)
	r.CollectionDivider = "/"

	ctx, span := tracer.Start(context.Background(), "test/go")
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
		r.Add(ctx, es...)
	}
	traceID := span.SpanContext().TraceID().String()
	span.End()

	fmt.Println("TraceID: ", traceID)
	fmt.Println(explore.ExploreLink{
		GrafanaURL:    "http://localhost:3000",
		DataSource:    "loki",
		DataSourceUID: "loki",
		TraceID:       traceID,
	})
	r.Fields["traceID"] = traceID

	err = loki.SendToLoki(r)
	if err != nil {
		panic(fmt.Errorf("got error when trying to send log to loki: %w", err))
	}
}
