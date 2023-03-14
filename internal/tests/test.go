package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/go-test-runner/internal/tree"
)

type Run struct {
	Collection *tree.RedBlack[string, *Collection]

	EarliestEvent time.Time
	LastEvent     time.Time
}

func New() *Run {
	return &Run{
		Collection: &tree.RedBlack[string, *Collection]{},
	}
}

func (r *Run) Add(ctx context.Context, events ...Event) {
	for _, event := range events {
		if !event.Timestamp.IsZero() {
			if r.EarliestEvent.IsZero() || r.EarliestEvent.After(event.Timestamp) {
				r.EarliestEvent = event.Timestamp
			}
			if r.LastEvent.Before(event.Timestamp) {
				r.LastEvent = event.Timestamp
			}
		}

		pkg := event.Package
		c, exists := r.Collection.Value(pkg)
		if !exists {
			c = &Collection{
				Package:        event.Package,
				SubtestDivider: "/",
				Tests:          &tree.RedBlack[string, *Test]{},
			}
			r.Collection.Insert(pkg, c)
		}

		c.add(ctx, event)
	}
}

func (r *Run) Get(pkg string, test string) (*Test, error) {
	val, ok := r.Collection.Value(pkg)
	if !ok {
		return nil, fmt.Errorf("package %s is not part of the test hierarchy", pkg)
	}
	tst, ok := val.Tests.Value(test)
	if !ok {
		return nil, fmt.Errorf("test %s / %s is not part of the test hierarchy", pkg, test)
	}
	return tst, nil
}

type Collection struct {
	Package        string
	Tests          *tree.RedBlack[string, *Test]
	SubtestDivider string

	State         State
	Events        []Event
	EarliestEvent time.Time
	LastEvent     time.Time
}

func (c *Collection) add(ctx context.Context, event Event) {
	if !event.Timestamp.IsZero() {
		if c.EarliestEvent.IsZero() || c.EarliestEvent.After(event.Timestamp) {
			c.EarliestEvent = event.Timestamp
		}
		if c.LastEvent.Before(event.Timestamp) {
			c.LastEvent = event.Timestamp
		}
	}

	pkg := event.Package
	test := event.Test

	if test == "" {
		c.Events = append(c.Events, event)
		if event, ok := event.Payload.(StateChange); ok {
			c.State = event.NewState
		}
		return
	}

	t, ok := c.Tests.Value(test)
	if !ok {
		t = &Test{
			Package: pkg,
			Name:    test,
		}
		c.Tests.Insert(test, t)
	}

	t.Events = append(t.Events, event)
	if event, ok := event.Payload.(StateChange); ok {
		t.State = event.NewState
	}
}

type Test struct {
	Package string
	Name    string

	State  State
	Events []Event
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
