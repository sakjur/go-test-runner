package tests

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/otel/codes"

	"go.opentelemetry.io/otel/trace"
)

type Run struct {
	Collection        map[string]*Collection
	CollectionDivider string

	Events        []Event
	EarliestEvent time.Time
	LastEvent     time.Time

	Context context.Context
	Fields  Tags
	Tracer  trace.Tracer
}

func New(tracer trace.Tracer, fields Tags) *Run {
	return &Run{
		Collection: map[string]*Collection{},
		Fields:     fields,
		Tracer:     tracer,
	}
}

func (r *Run) findCollectionParent(test string) *Collection {
	if r.CollectionDivider == "" {
		return nil
	}
	parts := strings.Split(test, r.CollectionDivider)
	for i := range parts {
		candidate := strings.Join(parts[:(len(parts)-i)], r.CollectionDivider)
		if c, ok := r.Collection[candidate]; ok {
			return c
		}
	}
	return nil
}

func (r *Run) Handle(event Event) error {
	pkg := event.Package
	c, exists := r.Collection[pkg]
	if !exists {
		ctx := r.Context
		if parent := r.findCollectionParent(event.Package); parent != nil {
			ctx = parent.ctx
		}
		ctx, span := r.Tracer.Start(ctx, "test/package")
		span.SetAttributes(attribute.String("packageName", event.Package))

		c = &Collection{
			Package:        event.Package,
			SubtestDivider: "/",
			Tests:          map[string]*Test{},
			ctx:            ctx,
			tracer:         r.Tracer,
		}
		r.Collection[pkg] = c
	}

	c.add(event)

	r.Events = append(r.Events, event)
	return nil
}

func (r *Run) Get(pkg string, test string) (*Test, error) {
	val, ok := r.Collection[pkg]
	if !ok {
		return nil, fmt.Errorf("package %s is not part of the test hierarchy", pkg)
	}
	tst, ok := val.Tests[test]
	if !ok {
		return nil, fmt.Errorf("test %s / %s is not part of the test hierarchy", pkg, test)
	}
	return tst, nil
}

type Collection struct {
	Package        string
	Tests          map[string]*Test
	SubtestDivider string

	State  State
	Events []Event

	ctx    context.Context
	tracer trace.Tracer
}

func (c *Collection) add(event Event) {
	pkg := event.Package
	test := event.Test

	if test == "" {
		c.Events = append(c.Events, event)
		handlePayload(c, event.Payload)
		return
	}

	t, ok := c.Tests[test]
	if !ok {
		ctx := c.ctx
		if parent := c.findTestParent(test); parent != nil {
			ctx = parent.ctx
		}
		ctx, span := c.tracer.Start(ctx, "test/runTest")
		span.SetAttributes(attribute.String("name", test), attribute.String("package", pkg))
		t = &Test{
			Package: pkg,
			Name:    test,
			ctx:     ctx,
		}
		c.Tests[test] = t
	}

	t.Events = append(t.Events, event)
	handlePayload(t, event.Payload)
}

func (c *Collection) findTestParent(test string) *Test {
	if c.SubtestDivider == "" {
		return nil
	}
	parts := strings.Split(test, c.SubtestDivider)
	for i := range parts {
		candidate := strings.Join(parts[:(len(parts)-i)], c.SubtestDivider)
		if t, ok := c.Tests[candidate]; ok {
			return t
		}
	}
	return nil
}

func (c *Collection) Context() context.Context {
	return c.ctx
}

func (c *Collection) SetState(state State) {
	c.State = state
}

type updateState interface {
	Context() context.Context
	SetState(State)
}

func handlePayload(handler updateState, payload EventPayload) {
	span := trace.SpanFromContext(handler.Context())
	switch ev := payload.(type) {
	case StateChange:
		state := ev.NewState
		handler.SetState(state)
		span.SetAttributes(
			attribute.String("state", state.String()),
		)
		switch state {
		case StatePassed:
			span.SetStatus(codes.Ok, "test passed")
			span.End()
		case StateFailed:
			span.SetStatus(codes.Error, "test failed")
			span.End()
		case StateSkipped:
			span.SetName("test/skipPackage")
			span.SetStatus(codes.Ok, "test skipped")
			span.End()
		}
	case Print:
		span.AddEvent(ev.Line)
	}
}

type Test struct {
	Package string
	Name    string

	State  State
	Events []Event

	ctx context.Context
}

func (t *Test) Context() context.Context {
	return t.ctx
}

func (t *Test) SetState(state State) {
	t.State = state
}

func (t *Test) TimeRange() (time.Time, time.Time) {
	if len(t.Events) == 0 {
		return time.Time{}, time.Time{}
	}
	return t.Events[0].Timestamp, t.Events[len(t.Events)-1].Timestamp
}

type State int8

const (
	StateUnknown State = iota
	StateRunning
	StatePassed
	StateFailed
	StateSkipped
)

func (s State) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StatePassed:
		return "passed"
	case StateFailed:
		return "failed"
	case StateSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

type Event struct {
	Package string
	Test    string

	Timestamp time.Time
	Payload   EventPayload
}

type EventPayload interface {
	isEventPayload()
}

type StateChange struct {
	NewState State `json:"new_state"`
}

func (StateChange) isEventPayload() {}

type Print struct {
	Line string `json:"line"`
}

func (Print) isEventPayload() {}

type Tags map[string]string

func (t Tags) String() string {
	ts := make([]string, 0, len(t))
	for key, value := range t {
		ts = append(ts, fmt.Sprintf("-t %s=%s", key, strconv.Quote(value)))
	}
	return strings.Join(ts, " ")
}

func (t Tags) Set(s string) error {
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
	t[key] = val
	return nil
}
