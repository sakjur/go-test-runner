package tests

import (
	"context"
	"go-test-runner/internal/tree"
	"strings"
	"time"
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

	testHierarchy := []string{event.Test}
	if c.SubtestDivider != "" {
		testHierarchy = strings.Split(test, c.SubtestDivider)
	}

	var t *Test
	bst := c.Tests
	for i := range testHierarchy {
		joined := strings.Join(testHierarchy[:i+1], c.SubtestDivider)
		var exists bool
		t, exists = bst.Value(joined)
		if !exists {
			t = &Test{
				Package:  pkg,
				Name:     joined,
				Subtests: &tree.RedBlack[string, *Test]{},
			}
			bst.Insert(joined, t)
		}
		bst = t.Subtests
	}

	t.Events = append(t.Events, event)
	if event, ok := event.Payload.(StateChange); ok {
		t.State = event.NewState
	}
}

type Test struct {
	Package  string
	Name     string
	Subtests *tree.RedBlack[string, *Test]

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
	Kind      string
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
