package console

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grafana/go-test-runner/internal/grafana"

	"github.com/grafana/go-test-runner/internal/tests"
)

type Console struct {
	failedTests map[string][]string
	traceID     string
}

func New(traceID string) *Console {
	return &Console{
		failedTests: map[string][]string{},
		traceID:     traceID,
	}
}

func (c *Console) FailedTests() []string {
	lines := []string{}
	for pkg, ts := range c.failedTests {
		sort.Strings(ts)
		for i, t := range ts {
			ts[i] = strconv.Quote(t)
		}

		lines = append(lines, fmt.Sprintf("Failures in %s: [%s]", pkg, strings.Join(ts, ", ")))
	}
	sort.Strings(lines)
	return lines
}

func (c *Console) Handle(e tests.Event) error {
	switch ev := e.Payload.(type) {
	case tests.Print:
		fmt.Print(ev.Line)
	case tests.StateChange:
		if e.Test != "" && ev.NewState == tests.StateFailed {
			c.failedTests[e.Package] = append(c.failedTests[e.Package], e.Test)
		}
	}

	return nil
}

func (c *Console) Stop() {
	for _, line := range c.FailedTests() {
		fmt.Println(line)
	}

	fmt.Println("TraceID: ", c.traceID)
	fmt.Println(grafana.ExploreLink{
		GrafanaURL:    "http://localhost:3000",
		DataSource:    "loki",
		DataSourceUID: "loki",
		TraceID:       c.traceID,
	})
}
