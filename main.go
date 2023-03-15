package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/grafana/go-test-runner/internal/console"

	"github.com/grafana/go-test-runner/internal/loki"
	"github.com/grafana/go-test-runner/internal/tests"
	"github.com/grafana/go-test-runner/internal/tracing"
)

type eventHandler interface {
	Handle(tests.Event) error
}

type stoppable interface {
	Stop()
}

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

	logClient, err := loki.New(r)
	if err != nil {
		panic(err)
	}

	ctx, span := tracer.Start(context.Background(), "test/go")
	traceID := span.SpanContext().TraceID().String()
	r.Fields["traceID"] = traceID
	r.Context = ctx

	handlers := []eventHandler{
		r,
		logClient,
		console.New(traceID),
	}

	goJSON := tests.NewGoJSON(os.Stdin)
	for {
		es, err := goJSON.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "error parsing line from `go test -json`!")
			continue
		}
		for _, e := range es {
			for _, handler := range handlers {
				err := handler.Handle(e)
				if err != nil {
					fmt.Fprintf(os.Stderr, "got error from handler '%T': %v", handler, err)
				}
			}
		}
	}

	for _, handler := range handlers {
		if stopper, ok := handler.(stoppable); ok {
			stopper.Stop()
		}
	}

	span.End()
}
