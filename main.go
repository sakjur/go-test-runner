package main

import (
	"errors"
	"flag"
	"io"
	"os"

	"github.com/go-kit/log"
	"github.com/grafana/go-test-runner/internal/cfg"
	"github.com/grafana/go-test-runner/internal/console"
	"github.com/grafana/go-test-runner/internal/loki"
	"github.com/grafana/go-test-runner/internal/tests"
)

type eventHandler interface {
	Handle(tests.Event) error
}

type stoppable interface {
	Stop()
}

func main() {
	conf := cfg.Config{}
	fields := cfg.Tags{}
	flag.Var(&fields, "t", "Add a key=value pair to the log output for each test")
	file := flag.String("c", "", "Path to configuration file")
	flag.Parse()

	logger := log.NewLogfmtLogger(os.Stderr)

	if *file != "" {
		f, err := os.Open(*file)
		if err != nil {
			logger.Log("msg", "Failed to open configuration file", "filename", *file, "error", err)
			os.Exit(-1)
		}
		conf, err = conf.Parse(*file, f)
		if err != nil {
			logger.Log("msg", "Failed to parse configuration file", "filename", *file, "error", err)
			os.Exit(-1)
		}
	}

	tracingOptions, traceErr := conf.Tracing()
	lokiOptions, lokiErr := conf.Loki()
	consoleOptions, consoleErr := conf.Console()
	grafanaOptions, grafanaErr := conf.Grafana()
	if err := errors.Join(traceErr, lokiErr, consoleErr, grafanaErr); err != nil {
		logger.Log("msg", "Failed to parse configuration for services", "error", err)
		os.Exit(-1)
	}

	r, err := tests.New(fields, tracingOptions)
	if err != nil {
		logger.Log("msg", "Failed to initialize test parser", "error", err)
		os.Exit(-1)
	}
	r.CollectionDivider = "/"

	logClient, err := loki.New(r, lokiOptions)
	if err != nil {
		logger.Log("msg", "Failed to initialize Loki sender", "error", err)
		os.Exit(-1)
	}

	handlers := []eventHandler{
		r,
		logClient,
		console.New(r.TraceID, consoleOptions, grafanaOptions),
	}

	failCount := 0
	goJSON := tests.NewGoJSON(os.Stdin)
	for {
		es, err := goJSON.ReadLine()
		if err != nil {
			failCount++
			if err == io.EOF {
				break
			}
			logger.Log("msg", "Error parsing line from `go test -json`!", "error", err)

			if failCount > 9 {
				logger.Log("msg", "Too many subsequent parsing errors, stopping processing", "error", err)
				os.Exit(-1)
			}
			continue
		} else {
			failCount = 0
		}
		for _, e := range es {
			for _, handler := range handlers {
				err := handler.Handle(e)
				if err != nil {
					logger.Log("msg", "Error from handler '%T': %v", handler, err)
				}
			}
		}
	}

	for _, handler := range handlers {
		if stopper, ok := handler.(stoppable); ok {
			stopper.Stop()
		}
	}
}
