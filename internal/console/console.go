package console

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grafana/go-test-runner/internal/cfg"
	"github.com/grafana/go-test-runner/internal/grafana"
	"github.com/grafana/go-test-runner/internal/tests"
)

type Console struct {
	printLevel     cfg.PrintLevel
	grafanaOptions cfg.GrafanaOptions
	failedTests    map[string][]string
	traceID        string
}

func New(traceID string, opts cfg.ConsoleOptions, grafanaOpts cfg.GrafanaOptions) *Console {
	return &Console{
		printLevel:     opts.PrintLevel,
		failedTests:    map[string][]string{},
		traceID:        traceID,
		grafanaOptions: grafanaOpts,
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
		if c.printLevel == cfg.PrintLevelRaw {
			fmt.Print(ev.Line)
		}
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
	if c.grafanaOptions.URL != "" {
		fmt.Println(grafana.LokiExploreLink{
			GrafanaURL:    c.grafanaOptions.URL,
			DataSource:    c.grafanaOptions.LokiDatasource,
			DataSourceUID: c.grafanaOptions.LokiDatasourceUID,
			TraceID:       c.traceID,
		})
	}
}
