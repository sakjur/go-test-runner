package tests

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Tracer struct {
	Run    *Run
	Tracer trace.Tracer
}

func (t Tracer) Report(ctx context.Context) string {
	ctx, span := t.Tracer.Start(ctx, "tests", trace.WithTimestamp(t.Run.EarliestEvent))

	t.Run.Collection.Walk(ctx, func(packageName string, collection *Collection) {
		ctx, span := t.Tracer.Start(ctx, "test/package", trace.WithTimestamp(collection.EarliestEvent))
		span.SetAttributes(attribute.String("packageName", packageName))
		if collection.State == StateSkipped {
			span.SetName("test/packageSkipped")
		}

		collection.Tests.Walk(ctx, reportTest(ctx, t.Tracer))
		span.End(trace.WithTimestamp(collection.LastEvent))
	})

	traceID := span.SpanContext().TraceID().String()
	span.End(trace.WithTimestamp(t.Run.LastEvent))
	return traceID
}

func reportTest(ctx context.Context, tracer trace.Tracer) func(testName string, test *Test) {
	return func(testName string, test *Test) {
		ctx := ctx
		if len(test.Events) != 0 {
			min, max := test.TimeRange()
			var span trace.Span
			ctx, span = tracer.Start(ctx, "test/run", trace.WithTimestamp(min))
			span.SetAttributes(
				attribute.String("testName", testName),
				attribute.String("state", test.State.String()),
			)

			switch test.State {
			case StateSkipped:
				span.SetName("test/testSkipped")
			case StateFailed:
				span.SetStatus(codes.Error, "failed running test")
			case StatePassed:
				span.SetStatus(codes.Ok, "successfully finished test")
			default:
				// no-op
			}

			for _, e := range test.Events {
				if ln, ok := e.Payload.(Print); ok {
					span.AddEvent(ln.Line, trace.WithTimestamp(e.Timestamp))
				}
			}

			span.End(trace.WithTimestamp(max))

		}
	}
}
