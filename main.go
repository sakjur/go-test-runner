package main

import (
	"context"
	"fmt"
	"go-test-runner/internal/tests"
	"io"
	"os"

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

	io.Pipe()

	r := tests.New()
	goJSON := tests.NewGoJSON(os.Stdin)
	for {
		events, err := goJSON.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		r.Add(context.Background(), events...)
	}

	t := &tests.Tracer{
		Run:    r,
		Tracer: tp.Tracer("go-test-runner"),
	}

	traceID := t.Report(context.Background())
	fmt.Println(traceID)
	tp.ForceFlush(context.Background())
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
